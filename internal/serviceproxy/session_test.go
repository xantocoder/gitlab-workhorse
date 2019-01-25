package serviceproxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apipkg "gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/testhelper"
)

var exampleURL = "http://example.com"

func TestInitSessionSecretNotLoaded(t *testing.T) {
	p := New(nil, "")
	err := p.initSessionStore()

	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "secret.setBytes"))
}

func TestInitSessionStore(t *testing.T) {
	p, _, _ := buildTestData(t, exampleURL)

	assert.NoError(t, p.initSessionStore())
	assert.NotNil(t, p.sessionStore)
	assert.Equal(t, sessionExpirationTime, p.sessionStore.Options.MaxAge)
	assert.True(t, p.sessionStore.Options.HttpOnly)
	assert.Equal(t, "/", p.sessionStore.Options.Path)
}

func TestInitSessionStoreWithExistingSessionStore(t *testing.T) {
	p, _, _ := buildTestData(t, exampleURL)
	assert.NoError(t, p.initSessionStore())

	existingSession := p.sessionStore

	assert.NoError(t, p.initSessionStore())
	assert.Equal(t, existingSession, p.sessionStore)
}

func TestGetSessionInitSessionStore(t *testing.T) {
	p, r, _ := buildTestData(t, exampleURL)

	session, err := p.getSession(r)
	assert.NoError(t, err)

	assert.NotNil(t, p.sessionStore)
	assert.NotNil(t, session)
}

func TestGetAndSaveSession(t *testing.T) {
	p, r, w := buildTestData(t, exampleURL)
	session, err := p.getSession(r)
	assert.NoError(t, err)

	session.Values["test"] = "foobar"
	assert.NoError(t, p.saveSession(w, r, session))

	session, err = p.getSession(r)
	assert.NoError(t, err)
	assert.Equal(t, "foobar", session.Values["test"])
}

func TestGetAndSaveSessionInfo(t *testing.T) {
	exampleURL := "http://example.com"
	s := &sessionInfo{RunnerSessionInfo: &apipkg.ServiceProxySettings{Url: exampleURL}}
	p, r, w := buildTestData(t, exampleURL)

	sessionInfo, err := p.getSessionInfo(r)
	assert.NoError(t, err)
	assert.Nil(t, sessionInfo.RunnerSessionInfo)

	assert.NoError(t, p.saveSessionInfo(w, r, s))

	sessionInfo, err = p.getSessionInfo(r)
	assert.NoError(t, err)
	assert.NotNil(t, sessionInfo.RunnerSessionInfo)
	assert.Equal(t, exampleURL, sessionInfo.RunnerSessionInfo.Url)
}

func TestGetSessionInfoError(t *testing.T) {
	p, r, w := buildTestData(t, exampleURL)
	session, err := p.getSession(r)
	assert.NoError(t, err)

	// Storing not a SessionInfo object
	session.Values[sessionInfoCookie] = "foobar"
	assert.NoError(t, session.Save(r, w))

	_, err = p.getSessionInfo(r)
	assert.Error(t, err)
	assert.Equal(t, errInvalidSessionInfo.Error(), err.Error())
}

func TestGetAndSaveBuildRunnerSession(t *testing.T) {
	headers := http.Header{"foo": []string{"bar"}}
	s := &apipkg.ServiceProxySettings{Url: exampleURL, Header: headers}
	p, r, w := buildTestData(t, exampleURL)

	runnerInfo, err := p.getSessionBuildRunnerSession(r)
	assert.NoError(t, err)
	assert.Nil(t, runnerInfo)

	assert.NoError(t, p.saveSessionBuildRunnerSession(w, r, s))

	runnerInfo, err = p.getSessionBuildRunnerSession(r)
	assert.NoError(t, err)
	assert.NotNil(t, runnerInfo)
	assert.Equal(t, exampleURL, runnerInfo.Url)
	assert.Equal(t, headers, runnerInfo.Header)
}

func buildTestData(t *testing.T, url string) (*Proxy, *http.Request, http.ResponseWriter) {
	testhelper.ConfigureSecret()

	p := New(nil, "")
	assert.Nil(t, p.sessionStore)

	r, err := http.NewRequest(http.MethodGet, url, nil)
	assert.NoError(t, err)

	return p, r, httptest.NewRecorder()
}
