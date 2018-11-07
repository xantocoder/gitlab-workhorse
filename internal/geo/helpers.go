package geo

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
)

func parseRailsResponse(proxyGitPushSSH string) (*api.ProxyGitPushSSH, error) {
	proxyGitPushSSHData := &api.ProxyGitPushSSH{CustomActionData: &api.CustomActionData{}}
	err := json.Unmarshal([]byte(proxyGitPushSSH), proxyGitPushSSHData)

	return proxyGitPushSSHData, err
}

func performRequest(req *http.Request) (string, error) {
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
