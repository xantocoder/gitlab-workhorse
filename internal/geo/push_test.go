package geo

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPushRequest(t *testing.T) {
	proxyGitPushSSHData, _ := parseRailsResponse(`{
		"gitlab_shell_custom_action_data": {
			"gl_id": "user-1",
			"primary_repo": "http://primary.geo/user/repo.git"
		},
		"gitlab_shell_output": "ZmFrZSBjb250ZW50IGhlcmU=\n",
		"authorization": "Fake abcd1234"
	}`)

	req, err := newPushRequest(proxyGitPushSSHData)

	assert.Nil(t, err)
	assert.Equal(t, pushContentTypeHeader, req.Header.Get("Content-Type"))
	assert.Equal(t, pushAcceptHeader, req.Header.Get("Accept"))
	assert.Equal(t, "user-1", req.Header.Get("Geo-GL-Id"))
	assert.Equal(t, "Fake abcd1234", req.Header.Get("Authorization"))

	body, err := ioutil.ReadAll(req.Body)
	assert.Nil(t, err)
	assert.Equal(t, []uint8([]byte{0x66, 0x61, 0x6b, 0x65, 0x20, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x20, 0x68, 0x65, 0x72, 0x65}), body)
}

func TestProcessPushResponse(t *testing.T) {
	rawBody := "fake response from push"

	json, err := processPushResponse(rawBody)

	assert.Nil(t, err)
	assert.Equal(t, `{"status":true,"message":"","result":"ZmFrZSByZXNwb25zZSBmcm9tIHB1c2g="}`, json)
}
