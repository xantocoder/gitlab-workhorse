package geo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const glID = "user-1"
const primaryRepo = "http://primary.geo/user/repo.git"
const authorization = "Fake abcd1234"

func TestParseRailsResponse(t *testing.T) {
	proxyGitPushSSHData, err := parseRailsResponse(`{
		"gitlab_shell_custom_action_data": {
			"gl_id": "user-1",
			"primary_repo": "http://primary.geo/user/repo.git"
		},
		"authorization": "Fake abcd1234"
	}`)

	assert.Nil(t, err)
	assert.Equal(t, glID, proxyGitPushSSHData.CustomActionData.GlID)
	assert.Equal(t, primaryRepo, proxyGitPushSSHData.CustomActionData.PrimaryRepo)
	assert.Equal(t, authorization, proxyGitPushSSHData.Authorization)
}

func TestNewInfoRefsRequest(t *testing.T) {
	proxyGitPushSSHData, _ := parseRailsResponse(`{
		"gitlab_shell_custom_action_data": {
			"gl_id": "user-1",
			"primary_repo": "http://primary.geo/user/repo.git"
		},
		"authorization": "Fake abcd1234"
	}`)

	req, err := newInfoRefsRequest(proxyGitPushSSHData)

	assert.Nil(t, err)
	assert.Equal(t, infoRefsContentTypeHeader, req.Header.Get("Content-Type"))
	assert.Equal(t, glID, req.Header.Get("Geo-GL-Id"))
	assert.Equal(t, authorization, req.Header.Get("Authorization"))
}

func TestMakeInfoRefsCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("info_refs response"))
	}))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	rawBody, err := makeInfoRefsCall(req)

	assert.Nil(t, err)
	assert.Equal(t, rawBody, "info_refs response")
}

func TestProcessInfoRefsResponse(t *testing.T) {
	rawBody := "001f# service=git-receive-pack\n0000the good stuff"

	json, err := processInfoRefsResponse(rawBody)

	assert.Nil(t, err)
	assert.Equal(t, `{"status":true,"message":"","result":"dGhlIGdvb2Qgc3R1ZmY="}`, json)
}
