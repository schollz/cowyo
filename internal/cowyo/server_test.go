package cowyo

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/microcosm-cc/bluemonday"
	"github.com/schollz/cowyo2/internal/database"
)

func TestLoadEnvironment(t *testing.T) {
	const key = "COWYO2_TEST_DATABASE_CREDENTIAL"
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset test environment variable: %v", err)
	}
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, previous)
		} else {
			os.Unsetenv(key)
		}
	})

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=loaded\n"), 0600); err != nil {
		t.Fatalf("write test .env: %v", err)
	}

	if err := loadEnvironment(path); err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if got := os.Getenv(key); got != "loaded" {
		t.Errorf("%s = %q, want %q", key, got, "loaded")
	}
}

func TestBuiltFrontendIncludesCowActions(t *testing.T) {
	index, err := fs.ReadFile(siteContent, "index.html")
	if err != nil {
		t.Fatalf("read built index: %v", err)
	}

	builtIndex := string(index)
	for description, marker := range map[string]string{
		"cow save control":      `class="cow-icon"`,
		"cow tooltip":           `data-tooltip="yo"`,
		"action menu":           `id="saveMenu"`,
		"theme action":          `id="themeAction"`,
		"dark theme icon":       `data-lucide="moon"`,
		"light theme icon":      `data-lucide="sun"`,
		"copy tooltip":          `data-tooltip="Copy paste text"`,
		"encryption action":     `data-lucide="shield-keyhole"`,
		"encryption tooltip":    `data-tooltip="Encrypt paste"`,
		"private action":        `id="privateAction"`,
		"private action icon":   `data-lucide="key-round"`,
		"publish action":        `id="publishAction"`,
		"publish icon":          `data-lucide="globe-2"`,
		"publish tooltip":       `data-tooltip="Publish page"`,
		"page-lock action":      `id="pageLockAction"`,
		"page-lock icon":        `data-lucide="lock-keyhole"`,
		"page-lock tooltip":     `data-tooltip="Lock page"`,
		"self-destruct action":  `id="selfDestructAction"`,
		"self-destruct icon":    `data-lucide="bomb"`,
		"self-destruct tooltip": `data-tooltip="Self destruct after next load"`,
		"about action":          `href="/about"`,
		"about tooltip":         `data-tooltip="About cowyo"`,
		"meta description":      `name="description"`,
		"canonical link":        `rel="canonical"`,
		"Open Graph metadata":   `property="og:title"`,
		"X card metadata":       `name="twitter:card"`,
		"JSON-LD metadata":      `type="application/ld+json"`,
		"password dialog":       `id="cryptoDialog"`,
		"password dialog label": `aria-labelledby="cryptoSubmit"`,
		"password mount":        `id="cryptoPasswordField"`,
		"remote cursor overlay": `id="cursorOverlay"`,
	} {
		if !strings.Contains(builtIndex, marker) {
			t.Errorf("built index does not contain %s", description)
		}
	}
	if strings.Contains(builtIndex, `type="password"`) {
		t.Error("built index contains a password input before the password dialog opens")
	}
	if strings.Contains(builtIndex, ` title="`) {
		t.Error("built index contains a browser-native title tooltip")
	}
	for description, marker := range map[string]string{
		"password dialog title":   `id="cryptoDialogTitle"`,
		"password dialog close":   `id="cryptoDialogClose"`,
		"global action separator": `save-menu-global-start`,
	} {
		if strings.Contains(builtIndex, marker) {
			t.Errorf("built index contains removed %s", description)
		}
	}

	actionOrder := []string{
		`id="copyTextAction"`,
		`id="cryptoAction"`,
		`id="privateAction"`,
		`id="publishAction"`,
		`id="pageLockAction"`,
		`id="selfDestructAction"`,
		`id="themeAction"`,
		`href="/about"`,
	}
	editorStart := strings.Index(builtIndex, `id="saveActions"`)
	if editorStart == -1 {
		t.Fatal("built index does not contain editor actions")
	}
	editorIndex := builtIndex[editorStart:]
	previousIndex := -1
	for _, marker := range actionOrder {
		currentIndex := strings.Index(editorIndex, marker)
		if currentIndex <= previousIndex {
			t.Fatalf("built action %q is out of order", marker)
		}
		previousIndex = currentIndex
	}
	scripts, err := fs.Glob(siteContent, "static/index-*.js")
	if err != nil {
		t.Fatalf("find built JavaScript: %v", err)
	}
	if len(scripts) != 1 {
		t.Fatalf("built JavaScript files = %v, want one optimized bundle", scripts)
	}
}

func TestIsCurlRequest(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{name: "curl", userAgent: "curl/8.7.1", want: true},
		{name: "case insensitive", userAgent: "CURL/8.0.0", want: true},
		{name: "browser", userAgent: "Mozilla/5.0", want: false},
		{name: "empty", userAgent: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/paste", nil)
			request.Header.Set("User-Agent", tt.userAgent)

			if got := isCurlRequest(request); got != tt.want {
				t.Fatalf("isCurlRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrowserRootRendersLandingPage(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for description, marker := range map[string]string{
		"landing page":          `class="landing-page"`,
		"primary action":        `href="/?new=1"`,
		"private action":        `href="/?new=private"`,
		"landing headline":      `Write it down.`,
		"open-source link":      `https://github.com/schollz/cowyo`,
		"indexing directive":    `content="` + robotsDirective(true) + `"`,
		"sponsorship link":      `https://github.com/sponsors/schollz`,
		"other-tools menu":      `<summary>other tools</summary>`,
		"protection icon row":   `class="landing-control-icons"`,
		"publish menu icon":     `data-lucide="globe-2"`,
		"page-lock menu icon":   `data-lucide="lock-keyhole"`,
		"encryption menu icon":  `data-lucide="shield-keyhole"`,
		"private menu icon":     `data-lucide="key-round"`,
		"self-destruct icon":    `data-lucide="bomb"`,
		"croc tool":             `https://getcroc.com`,
		"wthrtxt tool":          `https://wthrtxt.com`,
		"yesnotice tool":        `https://yesnotice.com`,
		"zero-account message":  `No account. Free and`,
		"assurance source link": `class="landing-assurance-source"`,
		"longevity message":     `Powering quick notes on the web for more than 10 years.`,
		"private key note":      `Private scratchpads encrypt every save in your browser.`,
		"terminal read":         `curl https://cowyo.com/my-notes`,
		"terminal write":        `curl --data-binary @notes.txt https://cowyo.com/`,
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("landing response does not contain %s", description)
		}
	}
	for description, marker := range map[string]string{
		"paste editor": `<textarea`,
		"paste menu":   `id="saveMenu"`,
	} {
		if strings.Contains(body, marker) {
			t.Errorf("landing response unexpectedly contains %s", description)
		}
	}
	if got := response.Header().Get("X-Robots-Tag"); got != robotsDirective(true) {
		t.Errorf("X-Robots-Tag = %q, want %q", got, robotsDirective(true))
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want HTML", got)
	}
}

func TestBrowserAboutRendersDedicatedPage(t *testing.T) {
	t.Setenv(siteURLEnvironment, "https://cowyo.example")
	setUpHandlerTest(t, Page{
		Title: "about",
		Text:  "this stored paste must not replace the dedicated page",
	})

	request := httptest.NewRequest(http.MethodGet, "/about", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for description, marker := range map[string]string{
		"about page":           `class="about-main"`,
		"about headline":       `a blank page + a link`,
		"start action":         `href="/?new=1"`,
		"private action":       `href="/?new=private"`,
		"unpublished detail":   `Unpublished`,
		"page-lock detail":     `Locked`,
		"encryption detail":    `Encrypted block`,
		"private detail":       `Private E2EE`,
		"private how note":     `Choose <strong>Private scratchpad</strong> before typing`,
		"self-destruct detail": `Self-destruct`,
		"publish menu icon":    `data-lucide="globe-2"`,
		"page-lock menu icon":  `data-lucide="lock-keyhole"`,
		"encryption menu icon": `data-lucide="shield-keyhole"`,
		"private menu icon":    `data-lucide="key-round"`,
		"self-destruct icon":   `data-lucide="bomb"`,
		"curl read example":    `curl https://cowyo.com/my-notes`,
		"curl create example":  `curl --data-binary @notes.txt`,
		"curl stdin example":   `curl --data-binary @-`,
		"curl advantages":      `Command-line advantages`,
		"page-control API":     `/api/v1/pages/my-notes/operations`,
		"sponsorship link":     `https://github.com/sponsors/schollz`,
		"canonical metadata":   `rel="canonical" href="https://cowyo.example/about"`,
		"indexing directive":   `content="` + robotsDirective(true) + `"`,
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("about response does not contain %s", description)
		}
	}
	for description, marker := range map[string]string{
		"paste editor":      `<textarea`,
		"stored paste text": `this stored paste must not replace the dedicated page`,
	} {
		if strings.Contains(body, marker) {
			t.Errorf("about response unexpectedly contains %s", description)
		}
	}
	if got := response.Header().Get("Link"); got != `<https://cowyo.example/about>; rel="canonical"` {
		t.Errorf("Link header = %q", got)
	}
	if got := response.Header().Get("X-Robots-Tag"); got != robotsDirective(true) {
		t.Errorf("X-Robots-Tag = %q, want %q", got, robotsDirective(true))
	}
}

func TestAboutRejectsPosts(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodPost, "/about", strings.NewReader("hidden paste"))
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want GET, HEAD", got)
	}
}

func TestRootNewQueryRedirectsToAlliterativeDocument(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodGet, "/?new=1", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", response.Code, http.StatusFound)
	}

	location := response.Header().Get("Location")
	name := strings.TrimPrefix(location, "/")
	if location == name || !isAlliterativeDocumentName(name) {
		t.Errorf("Location = %q, want /adjective-animal with matching initials", location)
	}
}

func TestRootPrivateQueryRedirectsToTrackerFreeBootstrap(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodGet, "/?new=private", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}

	location := response.Header().Get("Location")
	name := strings.TrimSuffix(strings.TrimPrefix(location, "/"), "?private=1")
	if !strings.HasSuffix(location, "?private=1") || !isAlliterativeDocumentName(name) {
		t.Fatalf("Location = %q, want /adjective-animal?private=1", location)
	}
}

func TestCurlRootStillRedirectsToAlliterativeDocument(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("User-Agent", "curl/8.7.1")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", response.Code, http.StatusFound)
	}

	location := response.Header().Get("Location")
	name := strings.TrimPrefix(location, "/")
	if location == name || !isAlliterativeDocumentName(name) {
		t.Errorf("Location = %q, want /adjective-animal with matching initials", location)
	}
}

func TestCurlGetsRawPasteText(t *testing.T) {
	const pasteText = "<b>hello & goodbye</b>\nline two"

	setUpHandlerTest(t, Page{
		Title: "paste",
		Text:  pasteText,
	})

	request := httptest.NewRequest(http.MethodGet, "/paste", nil)
	request.Header.Set("User-Agent", "curl/8.7.1")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	result := response.Result()
	defer result.Body.Close()

	if got := result.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
	}
	if got := result.Header.Get("Vary"); got != "User-Agent" {
		t.Errorf("Vary = %q, want %q", got, "User-Agent")
	}
	if got := response.Body.String(); got != pasteText {
		t.Errorf("body = %q, want %q", got, pasteText)
	}
}

func TestBrowserStillGetsPasteEditor(t *testing.T) {
	t.Setenv(googleTagEnvironment, "")
	setUpHandlerTest(t, Page{
		Title: "paste",
		Text:  "hello",
	})

	request := httptest.NewRequest(http.MethodGet, "/paste", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if got := response.Body.String(); !strings.Contains(got, "<textarea") || !strings.Contains(got, "hello") {
		t.Errorf("browser response does not contain the paste editor: %q", got)
	}
	if got := response.Header().Get("X-Robots-Tag"); got != robotsDirective(false) {
		t.Errorf("X-Robots-Tag = %q, want %q", got, robotsDirective(false))
	}
	if got := response.Body.String(); !strings.Contains(
		got,
		`content="`+robotsDirective(false)+`"`,
	) {
		t.Error("unpublished browser response does not contain a noindex directive")
	}
}

func TestBrowserPageIncludesConfiguredGoogleTag(t *testing.T) {
	const googleTag = "G-5QZV1WLQC1"
	t.Setenv(googleTagEnvironment, googleTag)
	setUpHandlerTest(t, Page{Title: "tagged"})

	request := httptest.NewRequest(http.MethodGet, "/tagged", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	body := response.Body.String()
	if !strings.Contains(body, "https://www.googletagmanager.com/gtag/js?id="+googleTag) {
		t.Error("browser response does not load the configured Google tag")
	}
	if !strings.Contains(body, `gtag("config", "`+googleTag+`");`) {
		t.Error("browser response does not configure the Google tag")
	}
}

func TestBrowserPageOmitsUnconfiguredOrInvalidGoogleTag(t *testing.T) {
	for _, tt := range []struct {
		name string
		tag  string
	}{
		{name: "unconfigured"},
		{name: "invalid", tag: `G-test"></script><script>alert(1)</script>`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(googleTagEnvironment, tt.tag)
			setUpHandlerTest(t, Page{Title: "untagged"})

			request := httptest.NewRequest(http.MethodGet, "/untagged", nil)
			request.Header.Set("User-Agent", "Mozilla/5.0")
			response := httptest.NewRecorder()

			if err := handle(response, request); err != nil {
				t.Fatalf("handle() error = %v", err)
			}
			if body := response.Body.String(); strings.Contains(body, "googletagmanager.com") {
				t.Errorf("browser response includes a Google tag for %q", tt.tag)
			}
		})
	}
}

func TestBrowserPageIncludesConfiguredGoogleAdSense(t *testing.T) {
	const client = "ca-pub-1234567890123456"
	t.Setenv(googleAdSenseEnvironment, client)
	setUpHandlerTest(t, Page{Title: "ad-supported"})

	for _, path := range []string{"/", "/about", "/ad-supported"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("User-Agent", "Mozilla/5.0")
		response := httptest.NewRecorder()

		if err := handle(response, request); err != nil {
			t.Fatalf("handle(%q) error = %v", path, err)
		}

		body := response.Body.String()
		if !strings.Contains(body, "https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client="+client) {
			t.Errorf("browser response for %q does not load AdSense with the configured client", path)
		}
		if !strings.Contains(body, `crossorigin="anonymous"`) {
			t.Errorf("browser response for %q does not set anonymous cross-origin loading for AdSense", path)
		}
	}
}

func TestBrowserPageOmitsUnconfiguredOrInvalidGoogleAdSense(t *testing.T) {
	for _, tt := range []struct {
		name   string
		client string
	}{
		{name: "unconfigured"},
		{name: "missing prefix", client: "1234567890123456"},
		{name: "wrong length", client: "ca-pub-123"},
		{name: "invalid characters", client: `ca-pub-1234567890123456"></script>`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(googleAdSenseEnvironment, tt.client)
			setUpHandlerTest(t, Page{Title: "ad-free"})

			request := httptest.NewRequest(http.MethodGet, "/ad-free", nil)
			request.Header.Set("User-Agent", "Mozilla/5.0")
			response := httptest.NewRecorder()

			if err := handle(response, request); err != nil {
				t.Fatalf("handle() error = %v", err)
			}
			if body := response.Body.String(); strings.Contains(body, "pagead2.googlesyndication.com") {
				t.Errorf("browser response includes AdSense for %q", tt.client)
			}
		})
	}
}

func TestBrowserPageIncludesConfiguredUmamiTracker(t *testing.T) {
	const (
		umamiURL  = "https://umami.schollz.com/"
		websiteID = "94db1cb1-74f4-4a40-ad6c-962362670409"
	)
	t.Setenv(umamiURLEnvironment, umamiURL)
	t.Setenv(umamiWebsiteIDEnvironment, websiteID)
	setUpHandlerTest(t, Page{Title: "tracked"})

	request := httptest.NewRequest(http.MethodGet, "/tracked", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	body := response.Body.String()
	if !strings.Contains(body, `src="https://umami.schollz.com/script.js"`) {
		t.Error("browser response does not load the configured Umami tracker")
	}
	if !strings.Contains(body, `data-website-id="`+websiteID+`"`) {
		t.Error("browser response does not include the configured Umami website ID")
	}
}

func TestBrowserPageOmitsIncompleteOrInvalidUmamiTracker(t *testing.T) {
	const (
		validURL = "https://umami.schollz.com"
		validID  = "94db1cb1-74f4-4a40-ad6c-962362670409"
	)
	for _, tt := range []struct {
		name      string
		umamiURL  string
		websiteID string
	}{
		{name: "unconfigured"},
		{name: "missing website ID", umamiURL: validURL},
		{name: "missing URL", websiteID: validID},
		{name: "URL has path", umamiURL: validURL + `/tracker`, websiteID: validID},
		{name: "URL has query", umamiURL: validURL + `?bad=true`, websiteID: validID},
		{name: "URL has credentials", umamiURL: `https://user:pass@umami.schollz.com`, websiteID: validID},
		{name: "invalid scheme", umamiURL: `javascript:alert(1)`, websiteID: validID},
		{name: "invalid website ID", umamiURL: validURL, websiteID: `"></script><script>alert(1)</script>`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(umamiURLEnvironment, tt.umamiURL)
			t.Setenv(umamiWebsiteIDEnvironment, tt.websiteID)
			setUpHandlerTest(t, Page{Title: "untracked"})

			request := httptest.NewRequest(http.MethodGet, "/untracked", nil)
			request.Header.Set("User-Agent", "Mozilla/5.0")
			response := httptest.NewRecorder()

			if err := handle(response, request); err != nil {
				t.Fatalf("handle() error = %v", err)
			}
			if body := response.Body.String(); strings.Contains(body, "data-website-id") {
				t.Errorf("browser response includes an Umami tracker for %q and %q", tt.umamiURL, tt.websiteID)
			}
		})
	}
}

func TestSelfDestructPageLoadsOnceThenIsDeleted(t *testing.T) {
	const (
		title   = "one-time-browser"
		content = "this is the final load"
	)
	setUpHandlerTest(t, Page{
		Title:        title,
		Text:         content,
		Published:    true,
		SelfDestruct: true,
	})

	headRequest := httptest.NewRequest(http.MethodHead, "/"+title, nil)
	headResponse := httptest.NewRecorder()
	if err := handle(headResponse, headRequest); err != nil {
		t.Fatalf("HEAD handle() error = %v", err)
	}
	if _, err := pageStore.GetPage(context.Background(), title); err != nil {
		t.Fatalf("HEAD consumed self-destruct page: %v", err)
	}

	finalRequest := httptest.NewRequest(http.MethodGet, "/"+title, nil)
	finalRequest.Header.Set("User-Agent", "Mozilla/5.0")
	finalResponse := httptest.NewRecorder()
	if err := handle(finalResponse, finalRequest); err != nil {
		t.Fatalf("final GET handle() error = %v", err)
	}

	finalBody := finalResponse.Body.String()
	if !strings.Contains(finalBody, content) {
		t.Fatalf("final GET does not contain page text: %q", finalBody)
	}
	if !strings.Contains(finalBody, `data-page-self-destruct="true"`) {
		t.Fatal("final GET does not expose the armed self-destruct state")
	}
	if got := finalResponse.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("final GET Cache-Control = %q, want no-store", got)
	}
	if got := finalResponse.Header().Get("X-Robots-Tag"); got != robotsDirective(false) {
		t.Errorf("final GET X-Robots-Tag = %q, want %q", got, robotsDirective(false))
	}
	if _, err := pageStore.GetPage(context.Background(), title); !errors.Is(err, database.ErrPageNotFound) {
		t.Fatalf("page remains after final GET; GetPage() error = %v", err)
	}

	nextRequest := httptest.NewRequest(http.MethodGet, "/"+title, nil)
	nextResponse := httptest.NewRecorder()
	if err := handle(nextResponse, nextRequest); err != nil {
		t.Fatalf("next GET handle() error = %v", err)
	}
	nextBody := nextResponse.Body.String()
	if strings.Contains(nextBody, content) {
		t.Fatal("deleted page text appeared after its final load")
	}
	if !strings.Contains(nextBody, `data-page-self-destruct="false"`) {
		t.Fatal("missing page rendered as armed for self destruct")
	}
}

func TestCurlConsumesSelfDestructPage(t *testing.T) {
	const (
		title   = "one-time-curl"
		content = "curl gets this once"
	)
	setUpHandlerTest(t, Page{
		Title:        title,
		Text:         content,
		SelfDestruct: true,
	})

	finalRequest := httptest.NewRequest(http.MethodGet, "/"+title, nil)
	finalRequest.Header.Set("User-Agent", "curl/8.7.1")
	finalResponse := httptest.NewRecorder()
	if err := handle(finalResponse, finalRequest); err != nil {
		t.Fatalf("final curl GET handle() error = %v", err)
	}
	if got := finalResponse.Body.String(); got != content {
		t.Errorf("final curl GET body = %q, want %q", got, content)
	}

	nextRequest := httptest.NewRequest(http.MethodGet, "/"+title, nil)
	nextRequest.Header.Set("User-Agent", "curl/8.7.1")
	nextResponse := httptest.NewRecorder()
	if err := handle(nextResponse, nextRequest); err != nil {
		t.Fatalf("next curl GET handle() error = %v", err)
	}
	if got := nextResponse.Body.String(); got != "" {
		t.Errorf("curl GET after deletion = %q, want empty", got)
	}
}

func TestPublishedBrowserPageCanBeIndexed(t *testing.T) {
	setUpHandlerTest(t, Page{
		Title:     "published",
		Text:      "discoverable",
		Published: true,
	})

	request := httptest.NewRequest(http.MethodGet, "/published", nil)
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if got := response.Header().Get("X-Robots-Tag"); got != robotsDirective(true) {
		t.Errorf("X-Robots-Tag = %q, want %q", got, robotsDirective(true))
	}
	body := response.Body.String()
	if !strings.Contains(body, `content="`+robotsDirective(true)+`"`) {
		t.Error("published browser response does not contain an index directive")
	}
	if !strings.Contains(body, `data-page-published="true"`) {
		t.Error("published browser response does not expose the publication state")
	}
}

func TestSitemapIncludesLandingAndOnlyPublishedPages(t *testing.T) {
	setUpHandlerTest(t, Page{
		Title:     "alpha-published",
		Text:      "first",
		Published: true,
	})
	ctx := context.Background()
	if err := pageStore.UpsertPage(ctx, database.Page{
		Title: "private-page",
		Text:  "not discoverable",
	}); err != nil {
		t.Fatalf("seed private page: %v", err)
	}
	if err := pageStore.UpsertPage(ctx, database.Page{
		Title:     "zebra-published",
		Text:      "second",
		Published: true,
	}); err != nil {
		t.Fatalf("seed second published page: %v", err)
	}
	if err := pageStore.UpsertPage(ctx, database.Page{
		Title:        "armed-published",
		Text:         "must not be indexed",
		Published:    true,
		SelfDestruct: true,
	}); err != nil {
		t.Fatalf("seed armed published page: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/sitemap.xml", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if got := response.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Errorf("Content-Type = %q, want XML", got)
	}
	var sitemap sitemapURLSet
	if err := xml.Unmarshal(response.Body.Bytes(), &sitemap); err != nil {
		t.Fatalf("parse sitemap: %v\n%s", err, response.Body.String())
	}
	wantLocations := []string{
		"https://example.com/",
		"https://example.com/about",
		"https://example.com/alpha-published",
		"https://example.com/zebra-published",
	}
	if len(sitemap.URLs) != len(wantLocations) {
		t.Fatalf("sitemap URLs = %+v, want %v", sitemap.URLs, wantLocations)
	}
	for i, want := range wantLocations {
		if got := sitemap.URLs[i].Location; got != want {
			t.Errorf("sitemap URL %d = %q, want %q", i, got, want)
		}
	}
	if strings.Contains(response.Body.String(), "private-page") {
		t.Fatal("sitemap contains an unpublished page")
	}
	if strings.Contains(response.Body.String(), "armed-published") {
		t.Fatal("sitemap contains a self-destruct page")
	}
}

func TestRobotsAdvertisesSitemap(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodGet, "http://example.com/robots.txt", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	if got := response.Body.String(); !strings.Contains(
		got,
		"Sitemap: https://example.com/sitemap.xml",
	) {
		t.Errorf("robots.txt = %q, want sitemap URL", got)
	}
}

func TestBrowserGetsLockStateWithoutLockMetadataInPaste(t *testing.T) {
	const (
		title    = "locked-browser"
		content  = "visible paste content"
		password = "correct horse battery staple"
	)
	setUpLockedHandlerTest(t, title, content, password)

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load locked page: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/"+title, nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	body := response.Body.String()
	if !strings.Contains(body, `data-page-locked="true"`) {
		t.Fatal("browser response does not expose the page lock state")
	}
	if !strings.Contains(body, content) {
		t.Fatal("browser response does not contain the paste content")
	}
	if strings.Contains(body, stored.LockSalt) || strings.Contains(body, stored.LockVerifier) {
		t.Fatal("browser response leaked page lock verification metadata")
	}
}

func TestWebsocketPersistsPaste(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "socket"})
	isolateWebsocketConnections(t)

	handlerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		if err := handle(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	socketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?place=socket"
	conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close websocket: %v", err)
		}
		select {
		case <-handlerDone:
		case <-time.After(time.Second):
			t.Error("websocket handler did not stop")
		}
	})

	want := Page{
		Title:       "socket",
		Text:        "saved through websocket",
		CursorStart: 5,
		CursorEnd:   9,
	}
	if err := conn.WriteJSON(websocketMessage{
		Type:        websocketMessageEdit,
		Text:        want.Text,
		CursorStart: want.CursorStart,
		CursorEnd:   want.CursorEnd,
	}); err != nil {
		t.Fatalf("write websocket update: %v", err)
	}

	var acknowledgement websocketMessage
	if err := conn.ReadJSON(&acknowledgement); err != nil {
		t.Fatalf("read websocket acknowledgement: %v", err)
	}
	if acknowledgement.Type != websocketMessageAck {
		t.Fatalf(
			"acknowledgement type = %q, want %q",
			acknowledgement.Type,
			websocketMessageAck,
		)
	}

	got, err := pageStore.GetPage(context.Background(), want.Title)
	if err != nil {
		t.Fatalf("load saved page: %v", err)
	}
	if got != (database.Page{
		Title:       want.Title,
		Text:        want.Text,
		CursorStart: want.CursorStart,
		CursorEnd:   want.CursorEnd,
	}) {
		t.Errorf("saved page = %+v, want %+v", got, want)
	}
}

func TestWebsocketBroadcastsEphemeralCursorPresence(t *testing.T) {
	setUpHandlerTest(t, Page{
		Title:       "presence",
		Text:        "shared note",
		CursorStart: 1,
		CursorEnd:   1,
	})
	isolateWebsocketConnections(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handle(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	socketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?place=presence"
	first, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		t.Fatalf("dial first websocket: %v", err)
	}
	t.Cleanup(func() {
		first.Close()
	})

	second, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		t.Fatalf("dial second websocket: %v", err)
	}
	t.Cleanup(func() {
		second.Close()
	})
	if err := second.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set second websocket deadline: %v", err)
	}

	cursor := websocketMessage{
		Type:        websocketMessageCursor,
		CursorStart: 0,
		CursorEnd:   0,
	}
	if err := first.WriteJSON(cursor); err != nil {
		t.Fatalf("write cursor presence: %v", err)
	}

	var broadcast websocketMessage
	if err := second.ReadJSON(&broadcast); err != nil {
		t.Fatalf("read cursor presence: %v", err)
	}
	if broadcast.Type != websocketMessageCursor {
		t.Fatalf(
			"cursor message type = %q, want %q",
			broadcast.Type,
			websocketMessageCursor,
		)
	}
	if broadcast.ClientID == "" {
		t.Fatal("cursor message does not identify its editor")
	}
	if broadcast.CursorStart != cursor.CursorStart || broadcast.CursorEnd != cursor.CursorEnd {
		t.Errorf(
			"cursor message range = %d:%d, want %d:%d",
			broadcast.CursorStart,
			broadcast.CursorEnd,
			cursor.CursorStart,
			cursor.CursorEnd,
		)
	}

	stored, err := pageStore.GetPage(context.Background(), "presence")
	if err != nil {
		t.Fatalf("load page after cursor presence: %v", err)
	}
	if stored.Text != "shared note" || stored.CursorStart != 1 || stored.CursorEnd != 1 {
		t.Errorf("cursor-only presence changed stored page: %+v", stored)
	}

	third, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		t.Fatalf("dial third websocket: %v", err)
	}
	t.Cleanup(func() {
		third.Close()
	})
	if err := third.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set third websocket deadline: %v", err)
	}

	var snapshot websocketMessage
	if err := third.ReadJSON(&snapshot); err != nil {
		t.Fatalf("read cursor snapshot: %v", err)
	}
	if snapshot.Type != websocketMessageCursor ||
		snapshot.ClientID != broadcast.ClientID ||
		snapshot.CursorStart != cursor.CursorStart ||
		snapshot.CursorEnd != cursor.CursorEnd {
		t.Errorf("cursor snapshot = %+v, want current presence %+v", snapshot, broadcast)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close first websocket: %v", err)
	}
	if err := second.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("reset second websocket deadline: %v", err)
	}

	var departure websocketMessage
	if err := second.ReadJSON(&departure); err != nil {
		t.Fatalf("read cursor departure: %v", err)
	}
	if departure.Type != websocketMessageCursorLeave ||
		departure.ClientID != broadcast.ClientID {
		t.Errorf("cursor departure = %+v, want client %q to leave", departure, broadcast.ClientID)
	}
}

func TestWebsocketCursorPresenceStaysOnItsPage(t *testing.T) {
	isolateWebsocketConnections(t)

	newConnection := func(id, place string) *Connection {
		return &Connection{
			id:    id,
			place: place,
			send:  make(chan websocketMessage, websocketSendQueueSize),
			done:  make(chan struct{}),
		}
	}
	current := newConnection("current", "shared-page")
	collaborator := newConnection("collaborator", "shared-page")
	otherPage := newConnection("other", "different-page")
	registerConnection(current)
	registerConnection(collaborator)
	registerConnection(otherPage)

	updateConnectionCursor(current, 4, 4)

	select {
	case message := <-collaborator.send:
		if message.Type != websocketMessageCursor ||
			message.ClientID != current.id ||
			message.CursorStart != 4 ||
			message.CursorEnd != 4 {
			t.Errorf("same-page cursor message = %+v", message)
		}
	default:
		t.Fatal("same-page collaborator did not receive cursor presence")
	}

	select {
	case message := <-otherPage.send:
		t.Fatalf("different page received cursor message %+v", message)
	default:
	}
}

func TestWebsocketConnectionQueueIsBounded(t *testing.T) {
	connection := &Connection{
		send: make(chan websocketMessage, 1),
		done: make(chan struct{}),
	}
	if !connection.enqueue(websocketMessage{Type: websocketMessageCursor}) {
		t.Fatal("first queued message was rejected")
	}
	if connection.enqueue(websocketMessage{Type: websocketMessageCursor}) {
		t.Fatal("message was accepted after the queue reached capacity")
	}
}

func TestWebsocketValidatesCursorRanges(t *testing.T) {
	for _, test := range []struct {
		name  string
		start int64
		end   int64
		want  bool
	}{
		{name: "caret", start: 3, end: 3, want: true},
		{name: "selection", start: 3, end: 8, want: true},
		{name: "negative start", start: -1, end: 0},
		{name: "reversed", start: 5, end: 4},
		{
			name:  "over message limit",
			start: 0,
			end:   maxWebsocketMessageSize + 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validCursorRange(test.start, test.end); got != test.want {
				t.Errorf(
					"validCursorRange(%d, %d) = %t, want %t",
					test.start,
					test.end,
					got,
					test.want,
				)
			}
		})
	}
}

func TestWebsocketMessageKeepsZeroCursorAndEmptyText(t *testing.T) {
	encoded, err := json.Marshal(websocketMessage{
		Type: websocketMessageUpdate,
	})
	if err != nil {
		t.Fatalf("marshal WebSocket message: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal WebSocket message: %v", err)
	}
	for _, field := range []string{"text", "cursor_start", "cursor_end"} {
		if _, ok := fields[field]; !ok {
			t.Errorf("zero-value WebSocket message omitted %q: %s", field, encoded)
		}
	}
}

func TestWebsocketCreatesPreviouslyMissingPaste(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	created, err := applyWebsocketUpdate(context.Background(), "new-page", pageUpdate{
		Text: "first browser save",
	})
	if err != nil {
		t.Fatalf("first WebSocket update error = %v", err)
	}
	if created.Title != "new-page" || created.Text != "first browser save" {
		t.Errorf("created page = %+v", created)
	}
}

func TestWebsocketPreservesServerManagedStateOnOrdinaryEdit(t *testing.T) {
	setUpHandlerTest(t, Page{
		Title:     "preserve-state",
		Text:      "original",
		Published: true,
	})

	saved, err := applyWebsocketUpdate(context.Background(), "preserve-state", pageUpdate{
		Text: "replacement",
	})
	if err != nil {
		t.Fatalf("ordinary WebSocket update error = %v", err)
	}
	if saved.Locked {
		t.Fatal("ordinary WebSocket update changed the database lock state")
	}
	if !saved.Published {
		t.Fatal("ordinary WebSocket update cleared the database publication state")
	}
	if saved.SelfDestruct {
		t.Fatal("ordinary WebSocket update changed the database self-destruct state")
	}
	if saved.Text != "replacement" {
		t.Errorf("saved text = %q, want replacement", saved.Text)
	}
}

func TestWebsocketSelfDestructLifecycleAndLockedRejection(t *testing.T) {
	const (
		title    = "self-destruct-lifecycle"
		content  = "keep until the next load"
		password = "correct horse battery staple"
	)
	setUpHandlerTest(t, Page{
		Title:     title,
		Text:      content,
		Published: true,
	})
	ctx := context.Background()

	armed, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      "attempted replacement",
		Operation: operationSelfDestruct,
	})
	if err != nil {
		t.Fatalf("arm self destruct error = %v", err)
	}
	if !armed.SelfDestruct {
		t.Fatal("self destruct operation did not arm the page")
	}
	if armed.Published {
		t.Fatal("arming self destruct did not unpublish the page")
	}
	if armed.Text != content {
		t.Errorf("arming self destruct changed text to %q, want %q", armed.Text, content)
	}

	edited, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text: "updated one-time text",
	})
	if err != nil {
		t.Fatalf("edit armed page error = %v", err)
	}
	if !edited.SelfDestruct {
		t.Fatal("ordinary edit cleared self destruct")
	}

	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationPublish,
	}); !errors.Is(err, errSelfDestructArmed) {
		t.Fatalf("publish armed page error = %v, want %v", err, errSelfDestructArmed)
	}

	cancelled, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationCancelSelfDestruct,
	})
	if err != nil {
		t.Fatalf("cancel self destruct error = %v", err)
	}
	if cancelled.SelfDestruct {
		t.Fatal("cancel operation left self destruct armed")
	}

	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationSelfDestruct,
	}); err != nil {
		t.Fatalf("re-arm self destruct error = %v", err)
	}
	locked, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      edited.Text,
		Operation: operationLock,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("lock armed page error = %v", err)
	}
	if !locked.Locked {
		t.Fatal("lock operation did not lock page")
	}
	if locked.SelfDestruct {
		t.Fatal("locking a page did not cancel self destruct")
	}

	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationSelfDestruct,
	}); !errors.Is(err, errPageLocked) {
		t.Fatalf("locked self destruct error = %v, want %v", err, errPageLocked)
	}
	stored, err := pageStore.GetPage(ctx, title)
	if err != nil {
		t.Fatalf("load locked page after rejected self destruct: %v", err)
	}
	if stored.SelfDestruct {
		t.Fatal("rejected locked operation armed self destruct")
	}
}

func TestWebsocketPagePublishingLifecycle(t *testing.T) {
	const title = "publish-lifecycle"
	setUpHandlerTest(t, Page{Title: title, Text: "draft"})
	ctx := context.Background()

	published, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      "attempted replacement",
		Operation: operationPublish,
	})
	if err != nil {
		t.Fatalf("publish update error = %v", err)
	}
	if !published.Published || published.Text != "draft" {
		t.Errorf("published page = %+v", published)
	}

	edited, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text: "edited while published",
	})
	if err != nil {
		t.Fatalf("published edit error = %v", err)
	}
	if !edited.Published {
		t.Fatal("ordinary edit cleared publication state")
	}

	locked, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      edited.Text,
		Operation: operationLock,
		Password:  "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("lock update error = %v", err)
	}
	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      locked.Text,
		Operation: operationUnpublish,
	}); !errors.Is(err, errPageLocked) {
		t.Fatalf("locked unpublish error = %v, want %v", err, errPageLocked)
	}

	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationUnlock,
		Password:  "correct horse battery staple",
	}); err != nil {
		t.Fatalf("unlock update error = %v", err)
	}
	unpublished, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      edited.Text,
		Operation: operationUnpublish,
	})
	if err != nil {
		t.Fatalf("unpublish update error = %v", err)
	}
	if unpublished.Published {
		t.Fatal("unpublish update left the page published")
	}

	titles, err := pageStore.ListPublishedPageTitles(ctx)
	if err != nil {
		t.Fatalf("list published pages: %v", err)
	}
	if len(titles) != 0 {
		t.Errorf("published titles after unpublish = %v, want none", titles)
	}
}

func TestWebsocketPageLockLifecycle(t *testing.T) {
	const (
		title    = "locked-socket"
		content  = "text that must not change"
		password = "correct horse battery staple"
	)
	setUpHandlerTest(t, Page{Title: title, Text: content})
	ctx := context.Background()

	locked, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      content,
		Operation: operationLock,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("lock update error = %v", err)
	}
	if !locked.Locked {
		t.Fatal("lock update did not set the database lock state")
	}
	if locked.Text != content {
		t.Errorf("locking changed page text to %q, want %q", locked.Text, content)
	}
	if locked.LockSalt == "" || locked.LockVerifier == "" {
		t.Fatal("lock update did not store verification credentials")
	}

	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{Text: "unauthorized edit"}); !errors.Is(err, errPageLocked) {
		t.Fatalf("ordinary locked update error = %v, want %v", err, errPageLocked)
	}
	stored, err := pageStore.GetPage(ctx, title)
	if err != nil {
		t.Fatalf("load locked page: %v", err)
	}
	if stored.Text != locked.Text {
		t.Fatal("ordinary update changed a locked page")
	}

	encryptedContent := "-----BEGIN COWYO ENCRYPTED BLOCK V1-----\n{\"data\":\"ciphertext\"}\n-----END COWYO ENCRYPTED BLOCK V1-----"
	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Text:      encryptedContent,
		Operation: operationEncrypt,
	}); !errors.Is(err, errPageLocked) {
		t.Fatalf("locked encryption error = %v, want %v", err, errPageLocked)
	}
	stored, err = pageStore.GetPage(ctx, title)
	if err != nil {
		t.Fatalf("load page after locked encryption: %v", err)
	}
	if !stored.Locked || stored.Text != content {
		t.Fatalf("locked encryption changed page = %+v", stored)
	}

	if _, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationUnlock,
		Password:  "wrong page password",
	}); !errors.Is(err, errWrongLockPassword) {
		t.Fatalf("wrong-password unlock error = %v, want %v", err, errWrongLockPassword)
	}

	unlocked, err := applyWebsocketUpdate(ctx, title, pageUpdate{
		Operation: operationUnlock,
		Password:  password,
	})
	if err != nil {
		t.Fatalf("unlock update error = %v", err)
	}
	if unlocked.Text != content {
		t.Errorf("unlocked text = %q, want %q", unlocked.Text, content)
	}
	if unlocked.Locked || unlocked.LockSalt != "" || unlocked.LockVerifier != "" {
		t.Fatal("unlock update did not clear database lock metadata")
	}

	edited, err := applyWebsocketUpdate(ctx, title, pageUpdate{Text: "editable again"})
	if err != nil {
		t.Fatalf("post-unlock edit error = %v", err)
	}
	if edited.Text != "editable again" {
		t.Errorf("post-unlock text = %q", edited.Text)
	}
}

func TestLockedDecryptionPreservesUnencryptedText(t *testing.T) {
	const encrypted = "-----BEGIN COWYO ENCRYPTED BLOCK V1-----\n{\"data\":\"ciphertext\"}\n-----END COWYO ENCRYPTED BLOCK V1-----"
	const footer = "\npublic footer"

	current := encrypted + footer
	valid := "decrypted secret" + footer
	if err := validateCryptoUpdate(current, valid, operationDecrypt, true); err != nil {
		t.Fatalf("valid locked decryption error = %v", err)
	}

	changedFooter := "decrypted secret\nchanged footer"
	if err := validateCryptoUpdate(current, changedFooter, operationDecrypt, true); err == nil {
		t.Fatal("locked decryption changed unencrypted text")
	}
}

func isolateWebsocketConnections(t *testing.T) {
	t.Helper()

	connectionsMu.Lock()
	previousConnections := connections
	connections = make(map[string]*Connection)
	connectionsMu.Unlock()

	t.Cleanup(func() {
		connectionsMu.Lock()
		connections = previousConnections
		connectionsMu.Unlock()
	})
}

func setUpHandlerTest(t *testing.T, page Page) {
	t.Helper()
	usePermissivePageOperationLimiters(t)

	previousStore := pageStore
	previousPolicy := policy
	previousTemplate := indexTemplate

	ctx := context.Background()
	var err error
	pageStore, err = database.Open(ctx, database.Config{
		SQLitePath: filepath.Join(t.TempDir(), "cowyo2.sqlite3"),
	})
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	policy = bluemonday.UGCPolicy()
	indexTemplate = template.Must(template.ParseFS(siteContent, "index.html"))

	if err := pageStore.UpsertPage(ctx, database.Page{
		Title:             page.Title,
		Text:              page.Text,
		CursorStart:       page.CursorStart,
		CursorEnd:         page.CursorEnd,
		Published:         page.Published,
		SelfDestruct:      page.SelfDestruct,
		Locked:            page.Locked,
		EndToEndEncrypted: page.EndToEndEncrypted,
	}); err != nil {
		t.Fatalf("seed test store: %v", err)
	}

	t.Cleanup(func() {
		if err := pageStore.Close(); err != nil {
			t.Errorf("close test store: %v", err)
		}
		pageStore = previousStore
		policy = previousPolicy
		indexTemplate = previousTemplate
	})
}

func setUpLockedHandlerTest(t *testing.T, title, text, password string) {
	t.Helper()
	setUpHandlerTest(t, Page{Title: title, Text: text})

	credentials, err := createPageLock(password)
	if err != nil {
		t.Fatalf("create page lock credentials: %v", err)
	}
	if err := pageStore.UpsertPage(context.Background(), database.Page{
		Title:        title,
		Text:         text,
		Locked:       true,
		LockSalt:     credentials.salt,
		LockVerifier: credentials.verifier,
	}); err != nil {
		t.Fatalf("seed locked page: %v", err)
	}
}
