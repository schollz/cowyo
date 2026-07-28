package main

import (
	"context"
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
		`id="publishAction"`,
		`id="pageLockAction"`,
		`id="selfDestructAction"`,
		`id="themeAction"`,
		`href="/about"`,
	}
	previousIndex := -1
	for _, marker := range actionOrder {
		currentIndex := strings.Index(builtIndex, marker)
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

func TestRootRedirectsToAlliterativeDocument(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
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

func TestSitemapIncludesOnlyPublishedPages(t *testing.T) {
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

	mu.Lock()
	previousConnections := connections
	connections = make(map[string]Connection)
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		connections = previousConnections
		mu.Unlock()
	})

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
	if err := conn.WriteJSON(want); err != nil {
		t.Fatalf("write websocket update: %v", err)
	}

	var acknowledgement Page
	if err := conn.ReadJSON(&acknowledgement); err != nil {
		t.Fatalf("read websocket acknowledgement: %v", err)
	}
	if acknowledgement.Title != "ok" {
		t.Fatalf("acknowledgement title = %q, want %q", acknowledgement.Title, "ok")
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

func TestWebsocketCreatesPreviouslyMissingPaste(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})

	created, err := applyWebsocketUpdate(context.Background(), "new-page", Page{
		Text: "first browser save",
	})
	if err != nil {
		t.Fatalf("first WebSocket update error = %v", err)
	}
	if created.Title != "new-page" || created.Text != "first browser save" {
		t.Errorf("created page = %+v", created)
	}
}

func TestWebsocketIgnoresClientLockStateOnOrdinaryEdit(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "forge-lock", Text: "original"})

	saved, err := applyWebsocketUpdate(context.Background(), "forge-lock", Page{
		Text:         "replacement",
		Published:    true,
		SelfDestruct: true,
		Locked:       true,
	})
	if err != nil {
		t.Fatalf("ordinary WebSocket update error = %v", err)
	}
	if saved.Locked {
		t.Fatal("ordinary WebSocket update changed the database lock state")
	}
	if saved.Published {
		t.Fatal("ordinary WebSocket update changed the database publication state")
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

	armed, err := applyWebsocketUpdate(ctx, title, Page{
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

	edited, err := applyWebsocketUpdate(ctx, title, Page{
		Text:         "updated one-time text",
		SelfDestruct: false,
	})
	if err != nil {
		t.Fatalf("edit armed page error = %v", err)
	}
	if !edited.SelfDestruct {
		t.Fatal("ordinary edit cleared self destruct")
	}

	if _, err := applyWebsocketUpdate(ctx, title, Page{
		Operation: operationPublish,
	}); !errors.Is(err, errSelfDestructArmed) {
		t.Fatalf("publish armed page error = %v, want %v", err, errSelfDestructArmed)
	}

	cancelled, err := applyWebsocketUpdate(ctx, title, Page{
		Operation: operationCancelSelfDestruct,
	})
	if err != nil {
		t.Fatalf("cancel self destruct error = %v", err)
	}
	if cancelled.SelfDestruct {
		t.Fatal("cancel operation left self destruct armed")
	}

	if _, err := applyWebsocketUpdate(ctx, title, Page{
		Operation: operationSelfDestruct,
	}); err != nil {
		t.Fatalf("re-arm self destruct error = %v", err)
	}
	locked, err := applyWebsocketUpdate(ctx, title, Page{
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

	if _, err := applyWebsocketUpdate(ctx, title, Page{
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

	published, err := applyWebsocketUpdate(ctx, title, Page{
		Text:      "attempted replacement",
		Operation: operationPublish,
	})
	if err != nil {
		t.Fatalf("publish update error = %v", err)
	}
	if !published.Published || published.Text != "draft" {
		t.Errorf("published page = %+v", published)
	}

	edited, err := applyWebsocketUpdate(ctx, title, Page{
		Text:      "edited while published",
		Published: false,
	})
	if err != nil {
		t.Fatalf("published edit error = %v", err)
	}
	if !edited.Published {
		t.Fatal("ordinary edit cleared publication state")
	}

	locked, err := applyWebsocketUpdate(ctx, title, Page{
		Text:      edited.Text,
		Operation: operationLock,
		Password:  "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("lock update error = %v", err)
	}
	if _, err := applyWebsocketUpdate(ctx, title, Page{
		Text:      locked.Text,
		Operation: operationUnpublish,
	}); !errors.Is(err, errPageLocked) {
		t.Fatalf("locked unpublish error = %v, want %v", err, errPageLocked)
	}

	if _, err := applyWebsocketUpdate(ctx, title, Page{
		Operation: operationUnlock,
		Password:  "correct horse battery staple",
	}); err != nil {
		t.Fatalf("unlock update error = %v", err)
	}
	unpublished, err := applyWebsocketUpdate(ctx, title, Page{
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

	locked, err := applyWebsocketUpdate(ctx, title, Page{
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

	if _, err := applyWebsocketUpdate(ctx, title, Page{Text: "unauthorized edit"}); !errors.Is(err, errPageLocked) {
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
	if _, err := applyWebsocketUpdate(ctx, title, Page{
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

	if _, err := applyWebsocketUpdate(ctx, title, Page{
		Operation: operationUnlock,
		Password:  "wrong page password",
	}); !errors.Is(err, errWrongLockPassword) {
		t.Fatalf("wrong-password unlock error = %v, want %v", err, errWrongLockPassword)
	}

	unlocked, err := applyWebsocketUpdate(ctx, title, Page{
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

	edited, err := applyWebsocketUpdate(ctx, title, Page{Text: "editable again"})
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

func setUpHandlerTest(t *testing.T, page Page) {
	t.Helper()

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
		Title:        page.Title,
		Text:         page.Text,
		CursorStart:  page.CursorStart,
		CursorEnd:    page.CursorEnd,
		Published:    page.Published,
		SelfDestruct: page.SelfDestruct,
		Locked:       page.Locked,
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
