package upstream

import (
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/gorilla/websocket"

	"gitlab.com/gitlab-org/labkit/tracing"

	apipkg "gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/artifacts"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/builds"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/channel"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/git"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/imageresizer"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/lfs"
	proxypkg "gitlab.com/gitlab-org/gitlab-workhorse/internal/proxy"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/queueing"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/redis"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/secret"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/senddata"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/sendfile"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/sendurl"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/staticpages"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/upload"
)

type matcherFunc func(*http.Request) bool

type routeEntry struct {
	method   string
	regex    *regexp.Regexp
	handler  http.Handler
	matchers []matcherFunc
}

type routeOptions struct {
	tracing  bool
	matchers []matcherFunc
}

type uploadPreparers struct {
	artifacts upload.Preparer
	lfs       upload.Preparer
	packages  upload.Preparer
	uploads   upload.Preparer
}

const (
	apiPattern           = `^/api/`
	ciAPIPattern         = `^/ci/api/`
	gitProjectPattern    = `^/([^/]+/){1,}[^/]+\.git/`
	projectPattern       = `^/([^/]+/){1,}[^/]+/`
	snippetUploadPattern = `^/uploads/personal_snippet`
	userUploadPattern    = `^/uploads/user`
	importPattern        = `^/import/`
)

func compileRegexp(regexpStr string) *regexp.Regexp {
	if len(regexpStr) == 0 {
		return nil
	}

	return regexp.MustCompile(regexpStr)
}

func withMatcher(f matcherFunc) func(*routeOptions) {
	return func(options *routeOptions) {
		options.matchers = append(options.matchers, f)
	}
}

func withoutTracing() func(*routeOptions) {
	return func(options *routeOptions) {
		options.tracing = false
	}
}

func route(method, regexpStr string, handler http.Handler, opts ...func(*routeOptions)) routeEntry {
	// Instantiate a route with the defaults
	options := routeOptions{
		tracing: true,
	}

	for _, f := range opts {
		f(&options)
	}

	handler = denyWebsocket(handler)                      // Disallow websockets
	handler = instrumentRoute(handler, method, regexpStr) // Add prometheus metrics
	if options.tracing {
		// Add distributed tracing
		handler = tracing.Handler(handler)
	}

	return routeEntry{
		method:   method,
		regex:    compileRegexp(regexpStr),
		handler:  handler,
		matchers: options.matchers,
	}
}

func wsRoute(regexpStr string, handler http.Handler, matchers ...matcherFunc) routeEntry {
	return routeEntry{
		method:   "GET",
		regex:    compileRegexp(regexpStr),
		handler:  instrumentRoute(handler, "GET", regexpStr),
		matchers: append(matchers, websocket.IsWebSocketUpgrade),
	}
}

// Creates matcherFuncs for a particular content type.
func isContentType(contentType string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return helper.IsContentType(contentType, r.Header.Get("Content-Type"))
	}
}

func (ro *routeEntry) isMatch(cleanedPath string, req *http.Request) bool {
	if ro.method != "" && req.Method != ro.method {
		return false
	}

	if ro.regex != nil && !ro.regex.MatchString(cleanedPath) {
		return false
	}

	ok := true
	for _, matcher := range ro.matchers {
		ok = matcher(req)
		if !ok {
			break
		}
	}

	return ok
}

func buildProxy(backend *url.URL, version string, rt http.RoundTripper, cfg config.Config) http.Handler {
	proxier := proxypkg.NewProxy(backend, version, rt)

	return senddata.SendData(
		sendfile.SendFile(apipkg.Block(proxier)),
		git.SendArchive,
		git.SendBlob,
		git.SendDiff,
		git.SendPatch,
		git.SendSnapshot,
		artifacts.SendEntry,
		sendurl.SendURL,
		imageresizer.NewResizer(cfg),
	)
}

// Routing table
// We match against URI not containing the relativeUrlRoot:
// see upstream.ServeHTTP

func (u *upstream) configureRoutes() {
	api := apipkg.NewAPI(
		u.Backend,
		u.Version,
		u.RoundTripper,
	)

	static := &staticpages.Static{DocumentRoot: u.DocumentRoot, AltDocumentRoot: u.AltDocumentRoot}
	proxy := buildProxy(u.Backend, u.Version, u.RoundTripper, u.Config)
	cableProxy := proxypkg.NewProxy(u.CableBackend, u.Version, u.CableRoundTripper)

	signingTripper := secret.NewRoundTripper(u.RoundTripper, u.Version)
	signingProxy := buildProxy(u.Backend, u.Version, signingTripper, u.Config)

	preparers := createUploadPreparers(u.Config)
	uploadPath := path.Join(u.DocumentRoot, "uploads/tmp")
	uploadAccelerateProxy := upload.Accelerate(&upload.SkipRailsAuthorizer{TempPath: uploadPath}, proxy, preparers.uploads)
	ciAPIProxyQueue := queueing.QueueRequests("ci_api_job_requests", uploadAccelerateProxy, u.APILimit, u.APIQueueLimit, u.APIQueueTimeout)
	ciAPILongPolling := builds.RegisterHandler(ciAPIProxyQueue, redis.WatchKey, u.APICILongPollingDuration)

	// Serve static files or forward the requests
	defaultUpstream := static.ServeExisting(
		u.URLPrefix,
		staticpages.CacheDisabled,
		static.DeployPage(static.ErrorPagesUnless(u.DevelopmentMode, staticpages.ErrorFormatHTML, uploadAccelerateProxy)),
	)
	probeUpstream := static.ErrorPagesUnless(u.DevelopmentMode, staticpages.ErrorFormatJSON, proxy)
	healthUpstream := static.ErrorPagesUnless(u.DevelopmentMode, staticpages.ErrorFormatText, proxy)

	u.Routes = []routeEntry{
		// Git Clone
		route("GET", gitProjectPattern+`info/refs\z`, git.GetInfoRefsHandler(api)),
		route("POST", gitProjectPattern+`git-upload-pack\z`, contentEncodingHandler(git.UploadPack(api)), withMatcher(isContentType("application/x-git-upload-pack-request"))),
		route("POST", gitProjectPattern+`git-receive-pack\z`, contentEncodingHandler(git.ReceivePack(api)), withMatcher(isContentType("application/x-git-receive-pack-request"))),
		route("PUT", gitProjectPattern+`gitlab-lfs/objects/([0-9a-f]{64})/([0-9]+)\z`, lfs.PutStore(api, signingProxy, preparers.lfs), withMatcher(isContentType("application/octet-stream"))),

		// CI Artifacts
		route("POST", apiPattern+`v4/jobs/[0-9]+/artifacts\z`, contentEncodingHandler(artifacts.UploadArtifacts(api, signingProxy, preparers.artifacts))),
		route("POST", ciAPIPattern+`v1/builds/[0-9]+/artifacts\z`, contentEncodingHandler(artifacts.UploadArtifacts(api, signingProxy, preparers.artifacts))),

		// ActionCable websocket
		wsRoute(`^/-/cable\z`, cableProxy),

		// Terminal websocket
		wsRoute(projectPattern+`-/environments/[0-9]+/terminal.ws\z`, channel.Handler(api)),
		wsRoute(projectPattern+`-/jobs/[0-9]+/terminal.ws\z`, channel.Handler(api)),

		// Proxy Job Services
		wsRoute(projectPattern+`-/jobs/[0-9]+/proxy.ws\z`, channel.Handler(api)),

		// Long poll and limit capacity given to jobs/request and builds/register.json
		route("", apiPattern+`v4/jobs/request\z`, ciAPILongPolling),
		route("", ciAPIPattern+`v1/builds/register.json\z`, ciAPILongPolling),

		// Maven Artifact Repository
		route("PUT", apiPattern+`v4/projects/[0-9]+/packages/maven/`, upload.BodyUploader(api, signingProxy, preparers.packages)),

		// Conan Artifact Repository
		route("PUT", apiPattern+`v4/packages/conan/`, upload.BodyUploader(api, signingProxy, preparers.packages)),
		route("PUT", apiPattern+`v4/projects/[0-9]+/packages/conan/`, upload.BodyUploader(api, signingProxy, preparers.packages)),

		// Generic Packages Repository
		route("PUT", apiPattern+`v4/projects/[0-9]+/packages/generic/`, upload.BodyUploader(api, signingProxy, preparers.packages)),

		// NuGet Artifact Repository
		route("PUT", apiPattern+`v4/projects/[0-9]+/packages/nuget/`, upload.Accelerate(api, signingProxy, preparers.packages)),

		// PyPI Artifact Repository
		route("POST", apiPattern+`v4/projects/[0-9]+/packages/pypi`, upload.Accelerate(api, signingProxy, preparers.packages)),

		// We are porting API to disk acceleration
		// we need to declare each routes until we have fixed all the routes on the rails codebase.
		// Overall status can be seen at https://gitlab.com/groups/gitlab-org/-/epics/1802#current-status
		route("POST", apiPattern+`v4/projects/[0-9]+/wikis/attachments\z`, uploadAccelerateProxy),
		route("POST", apiPattern+`graphql\z`, uploadAccelerateProxy),
		route("POST", apiPattern+`v4/groups/import`, upload.Accelerate(api, signingProxy, preparers.uploads)),
		route("POST", apiPattern+`v4/projects/import`, upload.Accelerate(api, signingProxy, preparers.uploads)),

		// Project Import via UI upload acceleration
		route("POST", importPattern+`gitlab_project`, upload.Accelerate(api, signingProxy, preparers.uploads)),
		// Group Import via UI upload acceleration
		route("POST", importPattern+`gitlab_group`, upload.Accelerate(api, signingProxy, preparers.uploads)),

		// Explicitly proxy API requests
		route("", apiPattern, proxy),
		route("", ciAPIPattern, proxy),

		// Serve assets
		route(
			"", `^/assets/`,
			static.ServeExisting(
				u.URLPrefix,
				staticpages.CacheExpireMax,
				NotFoundUnless(u.DevelopmentMode, proxy),
			),
			withoutTracing(), // Tracing on assets is very noisy
		),

		// Uploads
		route("POST", projectPattern+`uploads\z`, upload.Accelerate(api, signingProxy, preparers.uploads)),
		route("POST", snippetUploadPattern, upload.Accelerate(api, signingProxy, preparers.uploads)),
		route("POST", userUploadPattern, upload.Accelerate(api, signingProxy, preparers.uploads)),

		// For legacy reasons, user uploads are stored under the document root.
		// To prevent anybody who knows/guesses the URL of a user-uploaded file
		// from downloading it we make sure requests to /uploads/ do _not_ pass
		// through static.ServeExisting.
		route("", `^/uploads/`, static.ErrorPagesUnless(u.DevelopmentMode, staticpages.ErrorFormatHTML, proxy)),

		// health checks don't intercept errors and go straight to rails
		// TODO: We should probably not return a HTML deploy page?
		//       https://gitlab.com/gitlab-org/gitlab-workhorse/issues/230
		route("", "^/-/(readiness|liveness)$", static.DeployPage(probeUpstream)),
		route("", "^/-/health$", static.DeployPage(healthUpstream)),

		// This route lets us filter out health checks from our metrics.
		route("", "^/-/", defaultUpstream),

		route("", "", defaultUpstream),
	}
}

func createUploadPreparers(cfg config.Config) uploadPreparers {
	defaultPreparer := upload.NewObjectStoragePreparer(cfg)

	return uploadPreparers{
		artifacts: defaultPreparer,
		lfs:       lfs.NewLfsUploadPreparer(cfg, defaultPreparer),
		packages:  defaultPreparer,
		uploads:   defaultPreparer,
	}
}

func denyWebsocket(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if websocket.IsWebSocketUpgrade(r) {
			helper.HTTPError(w, r, "websocket upgrade not allowed", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
