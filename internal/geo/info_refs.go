package geo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
)

const infoRefsContentTypeHeader = "application/x-git-upload-pack-request"

// InfoRefsResponse contains the response payload from
// http(s)://<primary>/<namespace>/<repo>/.git/info/refs?service=git-receive-pack
//
type InfoRefsResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

// ProxyGitPushSSHInfoRefs gets the necessary authorization credentials from
// rails and then calls handleGetInfoRefs()
//
func ProxyGitPushSSHInfoRefs(a *api.API) http.Handler {
	return a.CustomActionHandler(func(w http.ResponseWriter, r *http.Request, a *api.Response) {
		// Performs a HTTP POST to
		// http(s)://<primary>/<namespace>/<repo>/.git/info/refs?service=git-receive-pack
		//
		proxyGitPushSSHData, err := parseRailsResponse(a.ProxyGitPushSSH)
		if err != nil {
			logHTTPError(w, err, "Failed to parse ProxyGitPushSSH JSON")
			return
		}

		req, err := newInfoRefsRequest(proxyGitPushSSHData)
		if err != nil {
			logHTTPError(w, err, "Failed to create new GET info_refs request")
			return
		}

		rawBody, err := makeInfoRefsCall(req)
		if err != nil {
			logHTTPError(w, err, "Failed to GET info_refs from primary")
			return
		}

		jsonResponse, err := processInfoRefsResponse(rawBody)
		if err != nil {
			logHTTPError(w, err, "Failed to process info_refs from primary")
			return
		}

		fmt.Fprint(w, jsonResponse)
	})
}

func parseRailsResponse(proxyGitPushSSH string) (*api.ProxyGitPushSSH, error) {
	proxyGitPushSSHData := &api.ProxyGitPushSSH{CustomActionData: &api.CustomActionData{}}
	err := json.Unmarshal([]byte(proxyGitPushSSH), proxyGitPushSSHData)

	return proxyGitPushSSHData, err
}

func newInfoRefsRequest(proxyGitPushSSHData *api.ProxyGitPushSSH) (*http.Request, error) {
	url := fmt.Sprintf("%s/info/refs?service=git-receive-pack", proxyGitPushSSHData.CustomActionData.PrimaryRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", infoRefsContentTypeHeader)
	req.Header.Set("Geo-GL-Id", proxyGitPushSSHData.CustomActionData.GlID)
	req.Header.Set("Authorization", proxyGitPushSSHData.Authorization)

	return req, nil
}

func makeInfoRefsCall(req *http.Request) (string, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	resp.Body.Close()

	return string(rawBody), nil
}

// HTTP(S) and SSH responses are very similar, except for the fragment below.
// As we're performing a git HTTP(S) request here, we'll get a HTTP(s)
// suitable git response.  However, we're executing in the context of an
// SSH session so we need to make the response suitable for what git over
// SSH expects.
//
// See Downloading Data > HTTP(S) section at:
// https://git-scm.com/book/en/v2/Git-Internals-Transfer-Protocols
func processInfoRefsResponse(rawBody string) (string, error) {
	re := regexp.MustCompile(`\A001f# service=git-receive-pack\n0000`)
	body := []byte(re.ReplaceAllString(rawBody, ``))
	base64Result := base64.URLEncoding.EncodeToString(body)
	response := InfoRefsResponse{Status: true, Message: "", Result: base64Result}

	json, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(json), err
}
