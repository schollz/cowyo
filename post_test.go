package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/schollz/cowyo2/internal/database"
)

func TestPostToNamedDocument(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "named", Text: "old", Published: true})
	usePostingLimiter(t, 100, time.Minute, 100)

	const content = "posted with curl\nexactly as written"
	request := newPostRequest("http://example.com/named", content)
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}

	const wantURL = "https://example.com/named"
	if response.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if got := response.Header().Get("Location"); got != wantURL {
		t.Errorf("Location = %q, want %q", got, wantURL)
	}
	if got := response.Body.String(); got != wantURL+"\n" {
		t.Errorf("body = %q, want %q", got, wantURL+"\n")
	}

	page, err := pageStore.GetPage(context.Background(), "named")
	if err != nil {
		t.Fatalf("load posted page: %v", err)
	}
	if page.Text != content {
		t.Errorf("stored text = %q, want %q", page.Text, content)
	}
	if !page.Published {
		t.Error("named POST cleared the existing publication state")
	}

	getRequest := httptest.NewRequest(http.MethodGet, wantURL, nil)
	getRequest.Header.Set("User-Agent", "curl/8.7.1")
	getResponse := httptest.NewRecorder()
	if err := handle(getResponse, getRequest); err != nil {
		t.Fatalf("curl GET handle() error = %v", err)
	}
	if got := getResponse.Body.String(); got != content {
		t.Errorf("curl GET body = %q, want %q", got, content)
	}
}

func TestPostPreservesSelfDestructState(t *testing.T) {
	const title = "armed-post"
	setUpHandlerTest(t, Page{
		Title:        title,
		Text:         "old",
		SelfDestruct: true,
	})
	usePostingLimiter(t, 100, time.Minute, 100)

	request := newPostRequest("http://example.com/"+title, "replacement")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("POST handle() error = %v", err)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("POST status = %d, want %d", response.Code, http.StatusCreated)
	}

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load posted self-destruct page: %v", err)
	}
	if stored.Text != "replacement" {
		t.Errorf("stored text = %q, want replacement", stored.Text)
	}
	if !stored.SelfDestruct {
		t.Fatal("named POST cleared the existing self-destruct state")
	}
}

func TestPostCannotReplaceLockedDocumentAndCurlDoesNotAlterIt(t *testing.T) {
	const (
		title    = "locked"
		content  = "keep this text"
		password = "correct horse battery staple"
	)
	setUpLockedHandlerTest(t, title, content, password)
	usePostingLimiter(t, 100, time.Minute, 100)

	request := newPostRequest("http://example.com/"+title, "replacement")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("POST handle() error = %v", err)
	}
	if response.Code != http.StatusLocked {
		t.Errorf("POST status = %d, want %d", response.Code, http.StatusLocked)
	}

	curlRequest := httptest.NewRequest(http.MethodGet, "http://example.com/"+title, nil)
	curlRequest.Header.Set("User-Agent", "curl/8.7.1")
	curlResponse := httptest.NewRecorder()
	if err := handle(curlResponse, curlRequest); err != nil {
		t.Fatalf("curl GET handle() error = %v", err)
	}
	if got := curlResponse.Body.String(); got != content {
		t.Errorf("curl body = %q, want only the paste content", got)
	}

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load locked page: %v", err)
	}
	if stored.Text != content || !stored.Locked {
		t.Fatal("POST or curl changed the locked page")
	}
}

func TestPostTreatsFormerPageLockSignaturesAsOrdinaryText(t *testing.T) {
	const text = "replacement\n-----BEGIN COWYO PAGE LOCK V1-----\nlegacy data\n-----END COWYO PAGE LOCK V1-----"
	setUpHandlerTest(t, Page{Title: "unlocked", Text: "original"})
	usePostingLimiter(t, 100, time.Minute, 100)

	request := newPostRequest("http://example.com/unlocked", text)
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("POST handle() error = %v", err)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("POST status = %d, want %d", response.Code, http.StatusCreated)
	}

	stored, err := pageStore.GetPage(context.Background(), "unlocked")
	if err != nil {
		t.Fatalf("load page: %v", err)
	}
	if stored.Text != text || stored.Locked {
		t.Fatal("signature-like text was not stored as ordinary unlocked content")
	}
}

func TestPostToRootCreatesRandomDocument(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})
	usePostingLimiter(t, 100, time.Minute, 100)

	const content = "a random paste"
	request := newPostRequest("http://example.com/", content)
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	location := response.Header().Get("Location")
	if !regexp.MustCompile(`^http://example\.com/[a-z]+-[a-z]+$`).MatchString(location) {
		t.Fatalf("Location = %q, want an alliterative document URL", location)
	}
	if got := response.Body.String(); got != location+"\n" {
		t.Errorf("body = %q, want %q", got, location+"\n")
	}

	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	title := strings.TrimPrefix(parsedLocation.Path, "/")
	if !isAlliterativeDocumentName(title) {
		t.Fatalf("random document name = %q, want matching initials", title)
	}
	page, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load random page: %v", err)
	}
	if page.Text != content {
		t.Errorf("stored text = %q, want %q", page.Text, content)
	}
}

func TestPostBodyLimit(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})
	usePostingLimiter(t, 100, time.Minute, 100)

	allowedContent := strings.Repeat("a", int(maxPostBodyBytes))
	allowedRequest := newPostRequest("http://example.com/allowed", allowedContent)
	allowedRequest.RemoteAddr = "192.0.2.10:1234"
	allowedResponse := httptest.NewRecorder()
	if err := handle(allowedResponse, allowedRequest); err != nil {
		t.Fatalf("allowed handle() error = %v", err)
	}
	if allowedResponse.Code != http.StatusCreated {
		t.Errorf("allowed status = %d, want %d", allowedResponse.Code, http.StatusCreated)
	}

	tooLargeRequest := newPostRequest(
		"http://example.com/too-large",
		strings.Repeat("b", int(maxPostBodyBytes)+1),
	)
	tooLargeRequest.RemoteAddr = "192.0.2.11:1234"
	tooLargeResponse := httptest.NewRecorder()
	if err := handle(tooLargeResponse, tooLargeRequest); err != nil {
		t.Fatalf("too-large handle() error = %v", err)
	}
	if tooLargeResponse.Code != http.StatusRequestEntityTooLarge {
		t.Errorf(
			"too-large status = %d, want %d",
			tooLargeResponse.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
	if _, err := pageStore.GetPage(context.Background(), "too-large"); !errors.Is(err, database.ErrPageNotFound) {
		t.Errorf("oversized paste was stored; GetPage() error = %v", err)
	}

	streamingRequest := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/streaming-too-large",
		io.LimitReader(strings.NewReader(strings.Repeat("c", int(maxPostBodyBytes)+1)), maxPostBodyBytes+1),
	)
	streamingRequest.ContentLength = -1
	streamingRequest.Header.Set("User-Agent", "curl/8.7.1")
	streamingRequest.RemoteAddr = "192.0.2.12:1234"
	streamingResponse := httptest.NewRecorder()
	if err := handle(streamingResponse, streamingRequest); err != nil {
		t.Fatalf("streaming handle() error = %v", err)
	}
	if streamingResponse.Code != http.StatusRequestEntityTooLarge {
		t.Errorf(
			"streaming status = %d, want %d",
			streamingResponse.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
}

func TestAdminPostBypassesBodyLimit(t *testing.T) {
	const adminKey = "test-admin-key"
	t.Setenv(adminPostKeyEnvironment, adminKey)
	setUpHandlerTest(t, Page{Title: "seed"})
	usePostingLimiter(t, 100, time.Minute, 100)

	content := strings.Repeat("a", int(maxPostBodyBytes)+1)
	request := newPostRequest("http://example.com/admin-large", content)
	request.Header.Set(adminPostKeyHeader, adminKey)
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	page, err := pageStore.GetPage(context.Background(), "admin-large")
	if err != nil {
		t.Fatalf("load admin page: %v", err)
	}
	if page.Text != content {
		t.Errorf("stored text length = %d, want %d", len(page.Text), len(content))
	}
}

func TestAdminPostCanPublishDocument(t *testing.T) {
	const adminKey = "test-admin-key"
	t.Setenv(adminPostKeyEnvironment, adminKey)

	tests := []struct {
		name string
		seed Page
	}{
		{name: "new page", seed: Page{Title: "seed"}},
		{name: "existing unpublished page", seed: Page{Title: "public-import", Text: "old"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setUpHandlerTest(t, test.seed)
			usePostingLimiter(t, 100, time.Minute, 100)

			request := newPostRequest("http://example.com/public-import", "published text")
			request.Header.Set(adminPostKeyHeader, adminKey)
			request.Header.Set(adminPostPublishedHeader, "true")
			response := httptest.NewRecorder()

			if err := handle(response, request); err != nil {
				t.Fatalf("handle() error = %v", err)
			}
			if response.Code != http.StatusCreated {
				t.Errorf("status = %d, want %d", response.Code, http.StatusCreated)
			}
			if got := response.Header().Get(adminPostPublishedHeader); got != "true" {
				t.Errorf(
					"%s response header = %q, want true",
					adminPostPublishedHeader,
					got,
				)
			}

			page, err := pageStore.GetPage(context.Background(), "public-import")
			if err != nil {
				t.Fatalf("load admin-published page: %v", err)
			}
			if !page.Published {
				t.Error("admin POST with publication header stored an unpublished page")
			}
		})
	}
}

func TestPostPublicationHeaderRequiresAdminKey(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "seed"})
	usePostingLimiter(t, 100, time.Minute, 100)

	request := newPostRequest("http://example.com/private-import", "private text")
	request.Header.Set(adminPostPublishedHeader, "true")
	response := httptest.NewRecorder()

	if err := handle(response, request); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	page, err := pageStore.GetPage(context.Background(), "private-import")
	if err != nil {
		t.Fatalf("load ordinary posted page: %v", err)
	}
	if page.Published {
		t.Error("ordinary POST used the admin-only publication header")
	}
}

func TestAdminPostBypassesRateLimitWithoutUsingTokens(t *testing.T) {
	const adminKey = "test-admin-key"
	t.Setenv(adminPostKeyEnvironment, adminKey)
	setUpHandlerTest(t, Page{Title: "admin-rate"})
	usePostingLimiter(t, 1, time.Minute, 1)

	for attempt := 1; attempt <= 2; attempt++ {
		request := newPostRequest(
			"http://example.com/admin-rate",
			"admin attempt "+strconv.Itoa(attempt),
		)
		request.Header.Set(adminPostKeyHeader, adminKey)
		request.RemoteAddr = "192.0.2.30:1234"
		response := httptest.NewRecorder()
		if err := handle(response, request); err != nil {
			t.Fatalf("admin attempt %d handle() error = %v", attempt, err)
		}
		if response.Code != http.StatusCreated {
			t.Errorf(
				"admin attempt %d status = %d, want %d",
				attempt,
				response.Code,
				http.StatusCreated,
			)
		}
	}

	for attempt := 1; attempt <= 2; attempt++ {
		request := newPostRequest(
			"http://example.com/admin-rate",
			"ordinary attempt "+strconv.Itoa(attempt),
		)
		request.RemoteAddr = "192.0.2.30:1234"
		response := httptest.NewRecorder()
		if err := handle(response, request); err != nil {
			t.Fatalf("ordinary attempt %d handle() error = %v", attempt, err)
		}

		wantStatus := http.StatusCreated
		if attempt == 2 {
			wantStatus = http.StatusTooManyRequests
		}
		if response.Code != wantStatus {
			t.Errorf(
				"ordinary attempt %d status = %d, want %d",
				attempt,
				response.Code,
				wantStatus,
			)
		}
	}
}

func TestAdminPostReplacesLockedPageWithoutUnlockingIt(t *testing.T) {
	const (
		adminKey = "test-admin-key"
		title    = "admin-locked"
		password = "correct horse battery staple"
	)
	t.Setenv(adminPostKeyEnvironment, adminKey)
	setUpLockedHandlerTest(t, title, "original", password)
	usePostingLimiter(t, 100, time.Minute, 100)

	before, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load locked page before admin POST: %v", err)
	}

	request := newPostRequest("http://example.com/"+title, "admin replacement")
	request.Header.Set(adminPostKeyHeader, adminKey)
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("admin POST handle() error = %v", err)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("admin POST status = %d, want %d", response.Code, http.StatusCreated)
	}

	after, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load locked page after admin POST: %v", err)
	}
	if after.Text != "admin replacement" {
		t.Errorf("stored text = %q, want admin replacement", after.Text)
	}
	if !after.Locked ||
		after.LockSalt != before.LockSalt ||
		after.LockVerifier != before.LockVerifier {
		t.Fatal("admin POST did not preserve the page lock metadata")
	}

	ordinaryRequest := newPostRequest("http://example.com/"+title, "ordinary replacement")
	ordinaryResponse := httptest.NewRecorder()
	if err := handle(ordinaryResponse, ordinaryRequest); err != nil {
		t.Fatalf("ordinary POST handle() error = %v", err)
	}
	if ordinaryResponse.Code != http.StatusLocked {
		t.Errorf(
			"ordinary POST status = %d, want %d",
			ordinaryResponse.Code,
			http.StatusLocked,
		)
	}
}

func TestAdminPostRequiresConfiguredMatchingKey(t *testing.T) {
	const title = "admin-key-check"
	setUpLockedHandlerTest(t, title, "original", "page-lock-password")
	usePostingLimiter(t, 100, time.Minute, 100)

	tests := []struct {
		name          string
		configuredKey string
		providedKey   string
	}{
		{name: "not configured", providedKey: "provided"},
		{name: "not provided", configuredKey: "configured"},
		{name: "does not match", configuredKey: "configured", providedKey: "wrong"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(adminPostKeyEnvironment, test.configuredKey)
			request := newPostRequest("http://example.com/"+title, "replacement")
			if test.providedKey != "" {
				request.Header.Set(adminPostKeyHeader, test.providedKey)
			}
			response := httptest.NewRecorder()

			if err := handle(response, request); err != nil {
				t.Fatalf("handle() error = %v", err)
			}
			if response.Code != http.StatusLocked {
				t.Errorf("status = %d, want %d", response.Code, http.StatusLocked)
			}
		})
	}
}

func TestPostRateLimit(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "limited"})
	usePostingLimiter(t, 1, time.Minute, 2)

	for attempt := 1; attempt <= 3; attempt++ {
		request := newPostRequest(
			"http://example.com/limited",
			"attempt "+strconv.Itoa(attempt),
		)
		request.RemoteAddr = "192.0.2.20:1234"
		response := httptest.NewRecorder()

		if err := handle(response, request); err != nil {
			t.Fatalf("attempt %d handle() error = %v", attempt, err)
		}

		if attempt <= 2 && response.Code != http.StatusCreated {
			t.Errorf("attempt %d status = %d, want %d", attempt, response.Code, http.StatusCreated)
		}
		if attempt == 3 {
			if response.Code != http.StatusTooManyRequests {
				t.Errorf("attempt 3 status = %d, want %d", response.Code, http.StatusTooManyRequests)
			}
			if response.Header().Get("Retry-After") == "" {
				t.Error("rate-limited response has no Retry-After header")
			}
		}
	}

	page, err := pageStore.GetPage(context.Background(), "limited")
	if err != nil {
		t.Fatalf("load rate-limited page: %v", err)
	}
	if page.Text != "attempt 2" {
		t.Errorf("stored text = %q, want the last accepted post", page.Text)
	}
}

func TestPostRateLimiterRefillsTokens(t *testing.T) {
	limiter := newPostRateLimiter(2, time.Minute, 2)
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time {
		return now
	}

	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("first request was denied")
	}
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("second request was denied")
	}
	if allowed, retryAfter := limiter.Allow("client"); allowed || retryAfter != 30*time.Second {
		t.Fatalf("third request = (%v, %s), want (false, 30s)", allowed, retryAfter)
	}

	now = now.Add(30 * time.Second)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("request was denied after a token refilled")
	}
}

func newPostRequest(target, content string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(content))
	request.Header.Set("User-Agent", "curl/8.7.1")
	return request
}

func usePostingLimiter(t *testing.T, requests int, per time.Duration, burst int) {
	t.Helper()

	previousLimiter := postingLimiter
	postingLimiter = newPostRateLimiter(requests, per, burst)
	t.Cleanup(func() {
		postingLimiter = previousLimiter
	})
}
