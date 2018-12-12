package api

// ProxyGitPushSSH contains necessary data and authorization credentials from
// rails (via gitlab-shell)
//
type ProxyGitPushSSH struct {
	// See GitlabShellCustomActionData below
	CustomActionData *CustomActionData `json:"gitlab_shell_custom_action_data"`

	// Authorization header content
	Authorization string `json:"authorization"`
}

// GitlabShellCustomActionData contains the gl_id and the full primary_repo URL so we
// (the secondary in the executing context) can perform necessary calls
//
type CustomActionData struct {
	// GitLab ID
	GlID string `json:"gl_id"`

	// Full URL to repository on the primary http(s)://<primary>/<namespace>/<repo>/.git
	PrimaryRepo string `json:"primary_repo"`
}
