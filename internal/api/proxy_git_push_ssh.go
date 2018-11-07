package api

// ProxyGitPushSSH contains necessary data and authorization credentials from
// rails (via gitlab-shell)
//
type ProxyGitPushSSH struct {
	// See GitlabShellCustomActionData below
	CustomActionData *CustomActionData `json:"gitlab_shell_custom_action_data"`

	// Output contains any necessary content from say a previous action
	// e.g. for an /info/refs?service=git-receive-pack request, Output will
	// contain that output which is utilised in geo.newPushRequest()
	Output string `json:"gitlab_shell_output"`

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
