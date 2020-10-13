package staticpages

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/gitlab-org/labkit/log"
	"gitlab.com/gitlab-org/labkit/mask"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/urlprefix"
)

type CacheMode int

const (
	CacheDisabled CacheMode = iota
	CacheExpireMax
)

// BUG/QUIRK: If a client requests 'foo%2Fbar' and 'foo/bar' exists,
// handleServeFile will serve foo/bar instead of passing the request
// upstream.
func (s *Static) ServeExisting(prefix urlprefix.Prefix, cache CacheMode, notFoundHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchRoots := []string{s.DocumentRoot, s.AltDocumentRoot}
		path := prefix.Strip(r.URL.Path)
		filenames := make([]string, 2)

		for _, root := range searchRoots {
			if root == "" {
				continue
			}

			// The filepath.Join does Clean traversing directories up
			file := filepath.Join(root, path)
			if !strings.HasPrefix(file, root) {
				helper.Fail500(w, r, &os.PathError{
					Op:   "open",
					Path: file,
					Err:  os.ErrInvalid,
				})
				return
			}

			filenames = append(filenames, file)
		}

		var found bool
		for _, f := range filenames {
			found = serveFileIfExists(f, cache, w, r)
			if found {
				return
			}
		}

		if notFoundHandler != nil {
			notFoundHandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)

		}
	})
}

func serveFileIfExists(file string, cache CacheMode, w http.ResponseWriter, r *http.Request) bool {
	var content *os.File
	var fi os.FileInfo
	var err error

	// Serve pre-gzipped assets
	if acceptEncoding := r.Header.Get("Accept-Encoding"); strings.Contains(acceptEncoding, "gzip") {
		content, fi, err = helper.OpenFile(file + ".gz")
	}

	// If not found, open the original file
	if content == nil || err != nil {
		content, fi, err = helper.OpenFile(file)
	} else {
		w.Header().Set("Content-Encoding", "gzip")
	}

	if content == nil || err != nil {
		return false
	}
	defer content.Close()

	switch cache {
	case CacheExpireMax:
		// Cache statically served files for 1 year
		cacheUntil := time.Now().AddDate(1, 0, 0).Format(http.TimeFormat)
		w.Header().Set("Cache-Control", "public")
		w.Header().Set("Expires", cacheUntil)
	}

	log.WithContextFields(r.Context(), log.Fields{
		"file":     file,
		"encoding": w.Header().Get("Content-Encoding"),
		"method":   r.Method,
		"uri":      mask.URL(r.RequestURI),
	}).Info("Send static file")

	http.ServeContent(w, r, filepath.Base(file), fi.ModTime(), content)

	return true
}
