package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path"
	"slices"
	"strings"
	"testing"
)

var publicExtensions = []string{".html", ".css", ".js", ".png", ".jpg", ".txt"}

func newSiteHandler(t *testing.T) http.Handler {
	t.Helper()

	site, err := publicSite()
	if err != nil {
		t.Fatal(err)
	}

	return newHandler(nil, nil, site)
}

func TestIndexIsServed(t *testing.T) {
	rec := httptest.NewRecorder()
	newSiteHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for the liveness probe", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Yivi Casino") {
		t.Error("body does not look like the casino page")
	}
}

func TestOnlyWebAssetsEmbedded(t *testing.T) {
	site, err := publicSite()
	if err != nil {
		t.Fatal(err)
	}

	err = fs.WalkDir(site, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		if !slices.Contains(publicExtensions, path.Ext(name)) {
			t.Errorf("%s is embedded but is not a web asset", name)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexReferencesExist(t *testing.T) {
	site, err := publicSite()
	if err != nil {
		t.Fatal(err)
	}

	index, err := fs.ReadFile(site, "index.html")
	if err != nil {
		t.Fatal(err)
	}

	for _, ref := range localReferences(string(index)) {
		// vendor/ is filled by `npm run vendor` and absent in a plain checkout.
		if strings.HasPrefix(ref, "vendor/") {
			continue
		}

		if _, err := fs.Stat(site, ref); err != nil {
			t.Errorf("index.html references %s, which is not embedded", ref)
		}
	}
}

func TestRepoFilesNotServed(t *testing.T) {
	paths := []string{
		"/package.json",
		"/go.mod",
		"/go.sum",
		"/main.go",
		"/server.go",
		"/Dockerfile",
		"/Dockerfile.test",
		"/docker-compose.yml",
		"/README.md",
		"/CLAUDE.md",
		"/.git/config",
		"/.gitignore",
		"/data/demo-privkey.pem",
		"/data/demo-publickey.pem",
		"/www/index.html",
		"/node_modules/jquery/package.json",
		"/vendor/../go.mod",
		"/../go.mod",
		"/%2e%2e/go.mod",
		"/css/../../go.mod",
	}

	handler := newSiteHandler(t)

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))

			if rec.Code == http.StatusOK {
				t.Errorf("GET %s = 200, want it not to be served", p)
			}
		})
	}
}

func TestDirectoriesNotListed(t *testing.T) {
	handler := newSiteHandler(t)

	for _, p := range []string{"/css/", "/img/", "/js/"} {
		t.Run(p, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))

			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s = %d, want 404 instead of a file index", p, rec.Code)
			}
		})
	}
}

// localReferences returns the relative src and href targets in html.
func localReferences(html string) []string {
	var refs []string

	for _, attr := range []string{`src="`, `href="`} {
		rest := html

		for {
			_, after, found := strings.Cut(rest, attr)
			if !found {
				break
			}

			ref, remainder, _ := strings.Cut(after, `"`)
			rest = remainder

			if strings.Contains(ref, "://") || strings.HasPrefix(ref, "/") {
				continue
			}

			refs = append(refs, ref)
		}
	}

	return refs
}
