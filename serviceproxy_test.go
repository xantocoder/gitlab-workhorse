package main

import (
	"bytes"
	"context"
	"encoding/pem"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/secret"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/serviceproxy"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/testhelper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/upstream"
)

var (
	servicesPreAuthPathRegex     = regexp.MustCompile(`^/api/v4/jobs/` + exampleJobID + `/proxy/authorize`)
	testUserContentDomain        = "randomdomain." + testUserContentPrimaryDomain
	testUserContentPrimaryDomain = "example.com"
	exampleRailsToken            = "foobar"
	exampleService               = "serviceFoo"
	exampleJobID                 = "1234"
	examplePort                  = "80"
	tokenParam                   = "token"
	serviceParam                 = "service"
	buildParam                   = "build"
	portParam                    = "port"
	domainParam                  = "domain"
	authorizationHeader          = "validAuthorization"
)

func TestServiceProxyErrors(t *testing.T) {
	tests := []struct {
		name               string
		proxyDomain        string
		params             func() url.Values
		expectedStatusCode int
	}{
		{
			name:        "When making request from main domain",
			proxyDomain: testUserContentPrimaryDomain,
			params: func() url.Values {
				return validProxyAuthParameters(t)
			},
			expectedStatusCode: http.StatusForbidden,
		}, {
			name:        "When no token param",
			proxyDomain: testUserContentDomain,
			params: func() url.Values {
				params := validProxyAuthParameters(t)
				params.Del(tokenParam)

				return params
			},
			expectedStatusCode: http.StatusBadRequest,
		}, {
			name:        "When invalid token",
			proxyDomain: testUserContentDomain,
			params: func() url.Values {
				params := validProxyAuthParameters(t)
				params.Set(tokenParam, "foobar")

				return params
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
		}, {
			name:        "When requesting from a domain different from the one stored in token",
			proxyDomain: "differentdomain." + testUserContentPrimaryDomain,
			params: func() url.Values {
				return validProxyAuthParameters(t)
			},
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ws := startWorkhorseServerWithUserContent("", testUserContentPrimaryDomain)
			defer ws.Close()

			proxyURL := serviceProxyURL(t, ws.URL, test.proxyDomain, "", test.params())

			client := serviceHTTPClient(t, ws.URL, false)
			resp, err := makeServiceProxyGetRequest(t, client, proxyURL)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
		})
	}
}

func TestServiceProxyForbiddenNotInSubdomain(t *testing.T) {
	ws := startWorkhorseServerWithUserContent("", testUserContentPrimaryDomain)
	defer ws.Close()

	proxyURL := serviceProxyURL(t, ws.URL, testUserContentPrimaryDomain, "", nil)

	client := serviceHTTPClient(t, ws.URL, false)
	resp, err := makeServiceProxyGetRequest(t, client, proxyURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestServiceProxyRequestFromSubSubDomainFails(t *testing.T) {
	ws := startWorkhorseServerWithUserContent("", testUserContentPrimaryDomain)
	defer ws.Close()

	proxyURL := serviceProxyURL(t, ws.URL, "foo."+testUserContentDomain, "", nil)

	client := serviceHTTPClient(t, ws.URL, false)
	resp, err := makeServiceProxyGetRequest(t, client, proxyURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPreAuthorizationErrors(t *testing.T) {
	tests := []struct {
		name                       string
		authServerURL              *regexp.Regexp
		authServerParams           url.Values
		authServerReturnStatusCode int
		authServerReturnMessage    string
		expectedStatusCode         int
	}{
		{
			name:                       "Pre-authorization fails",
			authServerURL:              nil,
			authServerParams:           nil,
			authServerReturnStatusCode: http.StatusForbidden,
			authServerReturnMessage:    "Access denied",
			expectedStatusCode:         http.StatusForbidden,
		}, {
			name:                       "With invalid pre-authorization endpoint",
			authServerURL:              regexp.MustCompile(`^/invalid_path`),
			authServerParams:           nil,
			authServerReturnStatusCode: http.StatusOK,
			expectedStatusCode:         http.StatusNotFound,
		}, {
			name:                       "With invalid pre-authorization params",
			authServerURL:              nil,
			authServerParams:           url.Values{"foo": []string{"bar"}},
			authServerReturnStatusCode: http.StatusOK,
			expectedStatusCode:         http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := testAuthServer(test.authServerURL, test.authServerParams, test.authServerReturnStatusCode, test.authServerReturnMessage)
			defer ts.Close()

			ws := startWorkhorseServerWithUserContent(ts.URL, testUserContentPrimaryDomain)
			defer ws.Close()

			params := validProxyAuthParameters(t)
			proxyURL := serviceProxyURL(t, ws.URL, testUserContentDomain, "", params)
			client := serviceHTTPClient(t, ws.URL, false)
			resp, err := makeServiceProxyGetRequest(t, client, proxyURL)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
		})
	}
}

func TestServiceProxyPreAuthSuccessRedirectRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ts := testAuthServer(servicesPreAuthPathRegex, nil, http.StatusOK, validAuthResponse(srv))
	defer ts.Close()

	ws := startWorkhorseServerWithUserContent(ts.URL, testUserContentPrimaryDomain)
	defer ws.Close()

	params := validProxyAuthParameters(t)
	proxyURL := serviceProxyURL(t, ws.URL, testUserContentDomain, "", params)
	client := serviceHTTPClient(t, ws.URL, false)
	resp, err := makeServiceProxyGetRequest(t, client, proxyURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)

	rootURL := serviceProxyURL(t, ws.URL, testUserContentDomain, "", nil)
	assert.Equal(t, rootURL, resp.Header.Get("Location"))
}

func TestServiceProxyRequest(t *testing.T) {
	postEndpoint := "try_post"
	message := []byte("hello world")
	params := validProxyAuthParameters(t)

	handlerFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, authorizationHeader, r.Header.Get("Authorization"))

		if r.URL.Path == "/"+postEndpoint {
			data, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, message, data)
			assert.Equal(t, http.MethodPost, r.Method)
		} else {
			assert.Equal(t, http.MethodGet, r.Method)
		}

		_, err := w.Write(message)
		assert.NoError(t, err)
	})

	tests := []struct {
		name           string
		createServerFn func(handler http.Handler) *httptest.Server
	}{
		{
			name:           "HTTP server",
			createServerFn: httptest.NewServer,
		}, {
			name:           "HTTPS server",
			createServerFn: httptest.NewTLSServer,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := test.createServerFn(handlerFn)
			defer srv.Close()

			ts := testAuthServer(servicesPreAuthPathRegex, validAuthServerParams(), http.StatusOK, validAuthResponse(srv))
			defer ts.Close()

			ws := startWorkhorseServerWithUserContent(ts.URL, testUserContentPrimaryDomain)
			defer ws.Close()

			// Authenticating request and redirecting to the proxy root
			proxyURL := serviceProxyURL(t, ws.URL, testUserContentDomain, "", params)
			client := serviceHTTPClient(t, ws.URL, true)
			resp, err := makeServiceProxyGetRequest(t, client, proxyURL)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			defer resp.Body.Close()
			body := resp.Body
			data, err := ioutil.ReadAll(body)
			assert.NoError(t, err)
			assert.Equal(t, message, data)

			// After the authorization, the following requests does not need to authenticate
			postURL := serviceProxyURL(t, ws.URL, testUserContentDomain, postEndpoint, nil)
			respPost, err := makeServiceProxyPostRequest(t, client, postURL, message)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			defer respPost.Body.Close()
			body = respPost.Body
			data, err = ioutil.ReadAll(body)
			assert.NoError(t, err)
			assert.Equal(t, message, data)
		})
	}
}

func startWorkhorseServerWithUserContent(authBackend string, userContentDomain string) *httptest.Server {
	if authBackend == "" {
		authBackend = upstream.DefaultBackend.String()
	}

	cfg := newUpstreamConfig(authBackend)
	cfg.UserContentDomain = userContentDomain

	return startWorkhorseServerWithConfig(cfg)
}

func serviceProxyURL(t *testing.T, wsURL string, domain string, path string, params url.Values) string {
	u, err := url.Parse(wsURL)
	assert.NoError(t, err)

	_, port, err := net.SplitHostPort(u.Host)
	assert.NoError(t, err)

	u.Host = net.JoinHostPort(domain, port)
	u.Path = path

	if params != nil {
		u.RawQuery = params.Encode()
	}

	return u.String()
}

func serviceHTTPClient(t *testing.T, wsURL string, followRedirects bool) *http.Client {
	u, err := url.Parse(wsURL)
	assert.NoError(t, err)

	// Necessary in tests to make the http Client send the cookies
	// As the doc says: If Jar is nil, cookies are only sent if they are explicitly
	jar, _ := cookiejar.New(nil)

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, u.Host)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if followRedirects {
				return nil
			}
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}
}

func makeServiceProxyGetRequest(t *testing.T, client *http.Client, userContentURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, userContentURL, nil)
	assert.NoError(t, err)

	return client.Do(req)
}

func makeServiceProxyPostRequest(t *testing.T, client *http.Client, userContentURL string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, userContentURL, bytes.NewBuffer(data))
	assert.NoError(t, err)

	return client.Do(req)
}

func generateJWTToken(t *testing.T, params url.Values) string {
	testhelper.ConfigureSecret()

	claims := serviceproxy.TokenInfo{
		BuildServiceInfo: serviceproxy.BuildService{
			Domain:      testUserContentDomain,
			ServiceName: params.Get(serviceParam),
			JobID:       params.Get(buildParam),
			Port:        params.Get(portParam),
		},
		Token:          params.Get(tokenParam),
		StandardClaims: jwt.StandardClaims{},
	}

	tokenString, err := secret.JWTTokenString(claims)
	assert.NoError(t, err)

	return tokenString
}

func validProxyAuthParameters(t *testing.T) url.Values {
	authServerParams := validAuthServerParams()

	authServerParams.Set(buildParam, exampleJobID)

	return url.Values{
		tokenParam: []string{generateJWTToken(t, authServerParams)},
	}
}

func validAuthServerParams() url.Values {
	return url.Values{
		tokenParam:   []string{exampleRailsToken},
		serviceParam: []string{exampleService},
		portParam:    []string{examplePort},
		domainParam:  []string{testUserContentDomain},
	}
}

func validAuthResponse(remote *httptest.Server) *api.Response {
	scheme := "http"

	if remote.TLS != nil {
		scheme = "https"
	}

	resp := &api.Response{
		Service: &api.ServiceProxySettings{
			Url:    scheme + "://" + remote.Listener.Addr().String(),
			Header: http.Header{"Authorization": []string{authorizationHeader}},
		},
	}

	if remote.TLS != nil && len(remote.TLS.Certificates) > 0 {
		data := bytes.NewBuffer(nil)
		pem.Encode(data, &pem.Block{Type: "CERTIFICATE", Bytes: remote.TLS.Certificates[0].Certificate[0]})
		resp.Service.CAPem = data.String()
	}

	return resp
}
