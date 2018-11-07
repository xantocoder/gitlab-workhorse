package geo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRailsResponse(t *testing.T) {
	proxyGitPushSSHData, err := parseRailsResponse(`{
		"gitlab_shell_custom_action_data": {
			"gl_id": "user-1",
			"primary_repo": "http://primary.geo/user/repo.git"
		},
		"authorization": "Fake abcd1234"
	}`)

	assert.Nil(t, err)
	assert.Equal(t, "user-1", proxyGitPushSSHData.CustomActionData.GlID)
	assert.Equal(t, "http://primary.geo/user/repo.git", proxyGitPushSSHData.CustomActionData.PrimaryRepo)
	assert.Equal(t, "Fake abcd1234", proxyGitPushSSHData.Authorization)
}
func TestPerformRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("info_refs response"))
	}))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	rawBody, err := performRequest(req)

	assert.Nil(t, err)
	assert.Equal(t, rawBody, "info_refs response")
}
