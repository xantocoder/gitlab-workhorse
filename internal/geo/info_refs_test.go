package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const glID = "user-1"
const primaryRepo = "http://primary.geo/user/repo.git"
const authorization = "Fake abcd1234"

func TestNewInfoRefsRequest(t *testing.T) {
	proxyGitPushSSHData, _ := parseRailsResponse(`{
		"gitlab_shell_custom_action_data": {
			"gl_id": "user-1",
			"primary_repo": "http://primary.geo/user/repo.git"
		},
		"gitlab_shell_output": "",
		"authorization": "Fake abcd1234"
	}`)

	req, err := newInfoRefsRequest(proxyGitPushSSHData)

	assert.Nil(t, err)
	assert.Equal(t, infoRefsContentTypeHeader, req.Header.Get("Content-Type"))
	assert.Equal(t, "user-1", req.Header.Get("Geo-GL-Id"))
	assert.Equal(t, "Fake abcd1234", req.Header.Get("Authorization"))
}

func TestProcessInfoRefsResponse(t *testing.T) {
	rawBody := "001f# service=git-receive-pack\n0000the good stuff"

	json, err := processInfoRefsResponse(rawBody)

	assert.Nil(t, err)
	assert.Equal(t, `{"status":true,"message":"","result":"dGhlIGdvb2Qgc3R1ZmY="}`, json)
}
