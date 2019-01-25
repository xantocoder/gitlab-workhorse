package upstream

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"
)

func TestConfigureRoutesWithUserContentDomain(t *testing.T) {
	userContentDomain := "example.com"
	routesWithout := upstreamRoutes(config.Config{})
	routesBlank := upstreamRoutes(config.Config{UserContentDomain: ""})
	routesWith := upstreamRoutes(config.Config{UserContentDomain: userContentDomain})

	assert.NotEmpty(t, routesWithout)
	assert.Len(t, routesBlank, len(routesWithout))
	assert.Len(t, routesWith, len(routesWithout)+1)

	// Ensuring that the first route catches only requests from the user content domain
	matcherFunc := routesWith[0].matchers[0]
	r, _ := http.NewRequest("GET", "http://"+userContentDomain, nil)
	assert.True(t, matcherFunc(r))

	r, _ = http.NewRequest("GET", "http://foo.com", nil)
	assert.False(t, matcherFunc(r))
}

func upstreamRoutes(cfg config.Config) []routeEntry {
	u := upstream{Config: cfg}
	u.configureRoutes()

	return u.Routes
}
