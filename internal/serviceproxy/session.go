package serviceproxy

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"

	apipkg "gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/secret"
)

const (
	proxySessionName  = "gitlab_workhorse_proxy"
	sessionInfoCookie = "session_info"

	// In seconds
	sessionExpirationTime = 6 * 3600
)

type sessionInfo struct {
	RunnerSessionInfo *apipkg.ServiceProxySettings `json:"runner_session"`
}

var (
	errInvalidSessionInfo = errors.New("invalid session info")
)

// Inits session store if not set
func (p *Proxy) initSessionStore() error {
	if p.sessionStore != nil {
		return nil
	}

	key, err := secret.Bytes()
	if err != nil {
		return err
	}

	p.sessionStore = sessions.NewCookieStore(key)
	p.sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionExpirationTime,
		HttpOnly: true,
	}

	return nil
}

func (p *Proxy) getSession(r *http.Request) (*sessions.Session, error) {
	if err := p.initSessionStore(); err != nil {
		return nil, err
	}

	session, err := p.sessionStore.Get(r, proxySessionName)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (p *Proxy) saveSession(w http.ResponseWriter, r *http.Request, s *sessions.Session) error {
	if err := s.Save(r, w); err != nil {
		return err
	}

	return nil
}

func (p *Proxy) getSessionInfo(r *http.Request) (*sessionInfo, error) {
	session, err := p.getSession(r)
	if err != nil {
		return nil, err
	}

	var s sessionInfo
	info := session.Values[sessionInfoCookie]
	if info == nil {
		return &s, nil
	}

	data, ok := info.([]byte)
	if !ok {
		return nil, errInvalidSessionInfo
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return nil, errInvalidSessionInfo
	}

	return &s, nil
}

func (p *Proxy) saveSessionInfo(w http.ResponseWriter, r *http.Request, s *sessionInfo) error {
	session, err := p.getSession(r)
	if err != nil {
		return nil
	}

	data, err := json.Marshal(&s)
	if err != nil {
		return err
	}

	session.Values[sessionInfoCookie] = data

	return p.saveSession(w, r, session)
}

func (p *Proxy) getSessionBuildRunnerSession(r *http.Request) (*apipkg.ServiceProxySettings, error) {
	info, err := p.getSessionInfo(r)
	if err != nil {
		return nil, err
	}

	return info.RunnerSessionInfo, nil
}

func (p *Proxy) saveSessionBuildRunnerSession(w http.ResponseWriter, r *http.Request, s *apipkg.ServiceProxySettings) error {
	info, err := p.getSessionInfo(r)
	if err != nil {
		return err
	}

	info.RunnerSessionInfo = s

	return p.saveSessionInfo(w, r, info)
}
