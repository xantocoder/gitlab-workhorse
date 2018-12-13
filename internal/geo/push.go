package geo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
)

const pushContentTypeHeader = "application/x-git-receive-pack-request"
const pushAcceptHeader = "application/x-git-receive-pack-result"

type PushResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func ProxyGitPushSSHPush(a *api.API) http.Handler {
	return a.CustomActionHandler(func(w http.ResponseWriter, r *http.Request, a *api.Response) {

		proxyGitPushSSHData, err := parseRailsResponse(a.ProxyGitPushSSH)
		if err != nil {
			logHTTPError(w, err, "Failed to parse ProxyGitPushSSH JSON")
			return
		}

		req, err := newPushRequest(proxyGitPushSSHData)
		if err != nil {
			logHTTPError(w, err, "Failed to create new GET push request")
			return
		}

		rawBody, err := performRequest(req)
		if err != nil {
			logHTTPError(w, err, "Failed to GET push from primary")
			return
		}

		jsonResponse, err := processPushResponse(rawBody)
		if err != nil {
			logHTTPError(w, err, "Failed to process push from primary")
			return
		}

		fmt.Fprint(w, jsonResponse)
	})
}

func newPushRequest(proxyGitPushSSHData *api.ProxyGitPushSSH) (*http.Request, error) {
	url := fmt.Sprintf("%s/git-receive-pack", proxyGitPushSSHData.CustomActionData.PrimaryRepo)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", pushContentTypeHeader)
	req.Header.Set("Accept", pushAcceptHeader)
	req.Header.Set("Geo-GL-Id", proxyGitPushSSHData.CustomActionData.GlID)
	req.Header.Set("Authorization", proxyGitPushSSHData.Authorization)

	base64Result, err := base64.StdEncoding.DecodeString(proxyGitPushSSHData.Output)
	if err != nil {
		return nil, err
	}

	req.Body = ioutil.NopCloser(strings.NewReader(string(base64Result)))

	return req, nil
}

func processPushResponse(rawBody string) (string, error) {
	base64Result := base64.URLEncoding.EncodeToString([]byte(rawBody))
	response := InfoRefsResponse{Status: true, Message: "", Result: base64Result}

	json, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(json), err
}
