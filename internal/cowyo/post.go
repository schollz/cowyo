package cowyo

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/cowyo2/internal/database"
)

const (
	maxPostBodyBytes         int64 = 16 * 1024
	randomDocumentAttempts         = 100
	postRateRequests               = 10
	postRateBurst                  = 5
	adminPostKeyEnvironment        = "ADMIN_POST_KEY"
	adminPostKeyHeader             = "X-Cowyo-Admin-Key"
	adminPostPublishedHeader       = "X-Cowyo-Published"
)

var postingLimiter = newPostRateLimiter(postRateRequests, time.Minute, postRateBurst)

func handlePost(w http.ResponseWriter, r *http.Request) error {
	admin := isAdminPost(r)
	adminPublishes := admin &&
		strings.EqualFold(strings.TrimSpace(r.Header.Get(adminPostPublishedHeader)), "true")
	if !admin {
		allowed, retryAfter := postingLimiter.Allow(postClientKey(r))
		if !allowed {
			retryAfterSeconds := max(1, int(math.Ceil(retryAfter.Seconds())))
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
			http.Error(w, "posting rate limit exceeded", http.StatusTooManyRequests)
			return nil
		}
	}

	if !admin && r.ContentLength > maxPostBodyBytes {
		http.Error(w, "paste exceeds the 16 KiB limit", http.StatusRequestEntityTooLarge)
		return nil
	}

	bodyReader := io.Reader(r.Body)
	if !admin {
		bodyReader = http.MaxBytesReader(w, r.Body, maxPostBodyBytes)
	}
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			http.Error(w, "paste exceeds the 16 KiB limit", http.StatusRequestEntityTooLarge)
			return nil
		}
		http.Error(w, "could not read paste", http.StatusBadRequest)
		return nil
	}

	page := database.Page{
		Text:      string(body),
		Published: adminPublishes,
	}
	operation := "edit"
	if r.URL.Path == "/" {
		page.Title, err = createRandomPage(r, page)
		if err != nil {
			return err
		}
		operation = "create"
	} else {
		page.Title = strings.TrimPrefix(r.URL.Path, "/")
		pageMutationMu.Lock()
		stored, loadErr := pageStore.GetPage(r.Context(), page.Title)
		if loadErr != nil && !errors.Is(loadErr, database.ErrPageNotFound) {
			pageMutationMu.Unlock()
			return fmt.Errorf("load paste %q: %w", page.Title, loadErr)
		}
		if loadErr == nil && stored.Locked && !admin {
			pageMutationMu.Unlock()
			http.Error(w, "page is locked", http.StatusLocked)
			return nil
		}
		if loadErr == nil {
			page.Published = stored.Published
			page.SelfDestruct = stored.SelfDestruct
			page.Locked = stored.Locked
			page.LockSalt = stored.LockSalt
			page.LockVerifier = stored.LockVerifier
			if adminPublishes {
				page.Published = true
			}
		} else {
			operation = "create"
		}
		err := pageStore.UpsertPage(r.Context(), page)
		pageMutationMu.Unlock()
		if err != nil {
			return fmt.Errorf("save paste %q: %w", page.Title, err)
		}
	}
	logPageEdit(page.Title, "http-post", operation, len(page.Text))

	location := postedDocumentURL(r, page.Title)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Location", location)
	if adminPublishes {
		w.Header().Set(adminPostPublishedHeader, "true")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusCreated)
	_, err = fmt.Fprintln(w, location)
	return err
}

func isAdminPost(r *http.Request) bool {
	configuredKey := os.Getenv(adminPostKeyEnvironment)
	providedKey := r.Header.Get(adminPostKeyHeader)
	if configuredKey == "" || providedKey == "" {
		return false
	}

	configuredHash := sha256.Sum256([]byte(configuredKey))
	providedHash := sha256.Sum256([]byte(providedKey))
	return subtle.ConstantTimeCompare(configuredHash[:], providedHash[:]) == 1
}

func createRandomPage(r *http.Request, page database.Page) (string, error) {
	for attempt := 0; attempt < randomDocumentAttempts; attempt++ {
		title, err := randomDocumentName()
		if err != nil {
			return "", fmt.Errorf("generate random paste name: %w", err)
		}
		page.Title = title

		created, err := pageStore.CreatePage(r.Context(), page)
		if err != nil {
			return "", fmt.Errorf("create random paste: %w", err)
		}
		if created {
			return title, nil
		}
	}
	return "", errors.New("could not allocate a random paste name")
}

func postedDocumentURL(r *http.Request, title string) string {
	if configuredURL, ok := configuredSiteURL(); ok {
		configuredURL.Path = "/" + title
		return configuredURL.String()
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if forwarded := strings.ToLower(strings.TrimSpace(
		strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0],
	)); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}

	return (&url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   "/" + title,
	}).String()
}

const siteURLEnvironment = "SITE_URL"

func configuredSiteURL() (*url.URL, bool) {
	configured := strings.TrimSpace(os.Getenv(siteURLEnvironment))
	if configured == "" {
		return nil, false
	}

	parsed, err := url.Parse(configured)
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return nil, false
	}
	parsed.Path = ""
	return parsed, true
}

func postClientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	if r.RemoteAddr == "" {
		return "unknown"
	}
	return r.RemoteAddr
}

type postRateLimiter struct {
	mu              sync.Mutex
	tokensPerSecond float64
	burst           float64
	clients         map[string]*postRateBucket
	now             func() time.Time
	lastCleanup     time.Time
	cleanupInterval time.Duration
	clientTTL       time.Duration
}

type postRateBucket struct {
	tokens   float64
	lastSeen time.Time
}

func newPostRateLimiter(requests int, per time.Duration, burst int) *postRateLimiter {
	return &postRateLimiter{
		tokensPerSecond: float64(requests) / per.Seconds(),
		burst:           float64(burst),
		clients:         make(map[string]*postRateBucket),
		now:             time.Now,
		cleanupInterval: per,
		clientTTL:       10 * per,
	}
}

func (l *postRateLimiter) Allow(client string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanup(now)

	bucket, ok := l.clients[client]
	if !ok {
		bucket = &postRateBucket{
			tokens:   l.burst,
			lastSeen: now,
		}
		l.clients[client] = bucket
	}

	elapsed := now.Sub(bucket.lastSeen).Seconds()
	if elapsed > 0 {
		bucket.tokens = min(l.burst, bucket.tokens+elapsed*l.tokensPerSecond)
	}
	bucket.lastSeen = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}

	missingTokens := 1 - bucket.tokens
	retryAfter := time.Duration(
		math.Ceil(missingTokens / l.tokensPerSecond * float64(time.Second)),
	)
	return false, retryAfter
}

func (l *postRateLimiter) cleanup(now time.Time) {
	if l.lastCleanup.IsZero() {
		l.lastCleanup = now
		return
	}
	if now.Sub(l.lastCleanup) < l.cleanupInterval {
		return
	}

	for client, bucket := range l.clients {
		if now.Sub(bucket.lastSeen) >= l.clientTTL {
			delete(l.clients, client)
		}
	}
	l.lastCleanup = now
}
