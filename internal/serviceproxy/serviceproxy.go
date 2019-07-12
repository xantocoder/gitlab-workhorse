package serviceproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	apipkg "gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/secret"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/transporthelper"
)

const (
	proxyURLTemplate = "%s/api/v4/jobs/%s/proxy"
)

type Proxy struct {
	api          *apipkg.API
	ProxyDomain  string // ProxyDomain holds the UserContentDomain param
	ListenAddr   string
	sessionStore *sessions.CookieStore
}

type BuildService struct {
	Domain      string `json:"domain"`
	JobID       string `json:"job_id"`
	ServiceName string `json:"service"`
	Port        string `json:"port"`
}

type TokenInfo struct {
	BuildServiceInfo BuildService `json:"service_info"`
	Token            string       `json:"token"`
	jwt.StandardClaims
}

var (
	transportWithTimeouts = transporthelper.TransportWithTimeouts()
	httpTransport         = transporthelper.TracingRoundTripper(transportWithTimeouts)

	errNoToken        = errors.New("token param not present")
	errInvalidToken   = errors.New("invalid token param")
	errInvalidParams  = errors.New("invalid parameters")
	errInvalidRequest = errors.New("invalid request")
)

func (b *BuildService) isValid() error {
	if b.Domain != "" &&
		b.JobID != "" &&
		b.ServiceName != "" {
		return nil
	}

	return errInvalidParams
}

func (t *TokenInfo) isValid() error {
	if t.Token == "" {
		return errNoToken
	}

	return t.BuildServiceInfo.isValid()
}

func New(api *apipkg.API, proxyDomain string, listenAddr string) *Proxy {
	return &Proxy{api: api, ProxyDomain: proxyDomain, ListenAddr: listenAddr}
}

var (
	config = oauth2.Config{
		ClientID:     "32c2da5b239f34d774111cc1f3f91ae4d874008074a07a3aa7e6b339d0c0ea4e",
		ClientSecret: "df0f8012d02d3d8fd648df17a7a1b0204f57c3154def1cec4b9c7d4a9218fc02",
		Scopes:       []string{"api"},
		RedirectURL:  "http://172.16.2.2.xip.io:3001/oauth2",
		// This points to our Authorization Server
		// if our Client ID and Client Secret are valid
		// it will attempt to authorize our user
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://172.16.2.2:3001/oauth/authorize",
			TokenURL: "http://172.16.2.2:3001/oauth/token",
		},
	}
)

// func Handler(myAPI *api.API) http.Handler {
// 	return myAPI.PreAuthorizeHandler(func(w http.ResponseWriter, r *http.Request, a *api.Response) {

func (p *Proxy) Authorize() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("***** Authorize")
		r.ParseForm()
		state := r.Form.Get("state")
		if state != "xyz" {
			http.Error(w, "State invalid", http.StatusBadRequest)
			return
		}

		code := r.Form.Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}

		token, err := config.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sessionInfo, err := p.getSessionInfo(r)
		if err != nil {
			helper.Fail500(w, r, err)
			return
		}

		sessionInfo.AccessToken = token.AccessToken

		if err := p.saveSessionInfo(w, r, sessionInfo); err != nil {
			helper.Fail500(w, r, err)
			return
		}

		http.Redirect(w, r, r.URL.Query().Get("ide_url"), http.StatusFound)
	})
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Disallowing requests from the main domain. Because
	// we're using sessions, we only allow requests from subdomains
	// if !isSubdomain(r.Host, p.ProxyDomain) {
	// 	helper.CaptureAndFail(w, r, fmt.Errorf("invalid domain"), http.StatusText(http.StatusForbidden), http.StatusForbidden)
	// 	return
	// }
	sessionInfo, err := p.getSessionInfo(r)
	if err != nil {
		helper.Fail500(w, r, err)
		return
	}
	if r.Host != sessionInfo.SessionDomain {
		sessionInfo = &SessionInfo{SessionDomain: r.Host}
		p.saveSessionInfo(w, r, sessionInfo)
	}

	if sessionInfo.AccessToken == "" {
		u, _ := url.Parse(config.RedirectURL)

		r.URL.Host = r.Host
		r.URL.Scheme = "http"

		q := url.Values{"ide_url": []string{r.URL.String()}}
		u.RawQuery = q.Encode()
		config.RedirectURL = u.String()

		oauthURL := config.AuthCodeURL("xyz")

		http.Redirect(w, r, oauthURL, http.StatusFound)
		return
	}

	runnerSession, err := p.getSessionBuildRunnerSession(r)
	if err != nil {
		helper.Fail500(w, r, err)
		return
	}

	// If the runner session exists means that we have already authenticated the request
	if runnerSession != nil {
		p.proxyRequest(w, r, runnerSession)
	} else {
		p.authenticateRequest(w, r)
	}
}

func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, s *apipkg.ServiceProxySettings) {
	var err error

	origPath := r.URL.Path
	sessionInfo, err := p.getSessionInfo(r)
	if err != nil {
		helper.Fail500(w, r, err)
		return
	}

	if sessionInfo.IdeURL == "" {
		fmt.Println("PROXING")
		// Saving the original custom domain
		// origHost := r.Host

		// Getting the url to proxy to the runner
		u, err := s.URL()
		if err != nil {
			helper.Fail500(w, r, err)
			return
		}

		// Updating the URL params needed to proxy to the runner
		r.URL.Scheme = u.Scheme
		r.URL.Path = path.Join(u.Path, r.URL.Path)
		r.URL.Host = u.Host
		r.Host = r.URL.Host

		// Adding the auth header for the runner
		r.Header.Add("Authorization", s.Header.Get("Authorization"))
		if len(s.CAPem) > 0 {
			// pool, err := x509.SystemCertPool()
			// if err != nil {
			// 	helper.Fail500(w, r, err)
			// 	return
			// }
			// pool.AppendCertsFromPEM([]byte(s.CAPem))
			// transportWithTimeouts.TLSClientConfig = &tls.Config{RootCAs: pool}

			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(s.CAPem))
			transportWithTimeouts.TLSClientConfig = &tls.Config{RootCAs: pool}
		}

		resp, err := httpTransport.RoundTrip(r)
		if err != nil {
			helper.CaptureAndFail(w, r, err, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		sessionInfo.IdeURL = resp.Header.Get("Location")
		p.saveSessionInfo(w, r, sessionInfo)
	}

	u, err := url.Parse(sessionInfo.IdeURL)
	// r.URL.Scheme = u.Scheme
	r.URL.Path = origPath

	r.Header.Set("Authorization", "Bearer "+sessionInfo.AccessToken)
	reverseProxy := httputil.NewSingleHostReverseProxy(u)
	reverseProxy.ServeHTTP(w, r)

	// r.Header.Set("Authorization", "Bearer "+session.Values["access_token"].(string))
	// log.Printf("r.URL.String(): %#+v\n", r.URL.String())
	// resp, err := httpTransport.RoundTrip(r)
	// if err != nil {
	// 	helper.CaptureAndFail(w, r, err, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	// 	return
	// }
	// defer resp.Body.Close()

	// copyHeader(w.Header(), resp.Header)

	// // transformRedirection(resp, w, origHost)
	// w.WriteHeader(resp.StatusCode)
	// io.Copy(w, resp.Body)
}

func (p *Proxy) authenticateRequest(w http.ResponseWriter, r *http.Request) {
	// Checking this first because, if no token or the token is invalid,
	// we can prevent a call to the Rails API
	token, err := p.decodeToken(r)
	if err != nil {
		if err == errInvalidToken {
			helper.Fail422(w, r, err)
			return
		}

		helper.Fail400(w, r, err)
		return
	}

	// Validating that the domain used in the requests is
	// the same one the user should access (set already in the token)
	if err := p.checkRequestValid(r, token.BuildServiceInfo.Domain); err != nil {
		helper.Fail400(w, r, err)
		return
	}

	fmt.Println("ES UN TOKEN VAAAAALIDO")

	// Building the Rails API endpoint for authorization
	u, err := p.authorizeAPIEndPointURL(r, token)
	if err != nil {
		helper.Fail500(w, r, err)
		return
	}

	// Updating the request URL to the api authorize endpoint
	origHost := r.Host
	r.URL = u
	r.URL.Host = u.Host
	r.Host = u.Host

	p.api.PreAuthorizeHandler(func(w http.ResponseWriter, r *http.Request, a *apipkg.Response) {
		if err := a.Service.Validate(); err != nil {
			helper.Fail500(w, r, err)
			return
		}

		// We save the build runner session info in the session
		err = p.saveSessionBuildRunnerSession(w, r, a.Service)
		if err != nil {
			helper.Fail500(w, r, err)
			return
		}

		// Redirecting to the proxy domain used without any query params
		u := fmt.Sprintf("%s://%s", protocolFor(r), origHost)
		http.Redirect(w, r, u, http.StatusMovedPermanently)
	}, "authorize").ServeHTTP(w, r)
}

func (p *Proxy) authorizeAPIEndPointURL(r *http.Request, token *TokenInfo) (*url.URL, error) {
	generated := fmt.Sprintf(proxyURLTemplate, p.api.URL.String(), token.BuildServiceInfo.JobID)
	u, err := url.Parse(generated)
	if err != nil {
		return nil, err
	}

	// Passing the necessary query params to the Rails API endpoint
	q := url.Values{
		"service": []string{token.BuildServiceInfo.ServiceName},
		"port":    []string{token.BuildServiceInfo.Port},
		"token":   []string{token.Token},
		"domain":  []string{token.BuildServiceInfo.Domain},
	}
	u.RawQuery = q.Encode()

	return u, nil
}

// The token obtained from rails is encoded using JWT and Workhorse secret.
// If the decodification fails it's because it's not the right token and we
// can avoid a request to Rails
func (p *Proxy) decodeToken(r *http.Request) (*TokenInfo, error) {
	// If it doesn't exist means that we need to authenticate the request agains Rails
	// If no token present to authenticate we can return an error
	tokenParam := r.URL.Query().Get("token")
	if tokenParam == "" {
		return nil, errNoToken
	}

	serviceToken, err := jwt.ParseWithClaims(tokenParam, &TokenInfo{}, func(token *jwt.Token) (interface{}, error) {
		secretBytes, _ := secret.Bytes()
		return secretBytes, nil
	})

	if err != nil {
		return nil, errInvalidToken
	}

	if claims, ok := serviceToken.Claims.(*TokenInfo); ok && serviceToken.Valid {
		if err := claims.isValid(); err != nil {
			return nil, err
		}

		return claims, nil
	}

	return nil, errInvalidToken
}

func (p *Proxy) checkRequestValid(r *http.Request, proxyDomain string) error {
	if !helper.ExactDomain(r.Host, proxyDomain) {
		return errInvalidRequest
	}

	return nil
}
