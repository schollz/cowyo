package cowyo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/microcosm-cc/bluemonday"
	"github.com/schollz/cowyo2/internal/database"
	"github.com/schollz/cowyo2/internal/site"
	log "github.com/schollz/logger"
)

// flag for port
var flagPort string
var flagLog string
var pageStore *database.Store
var connections map[string]*Connection
var connectionsMu sync.Mutex
var pageMutationMu sync.Mutex

const (
	googleTagEnvironment      = "GOOGLE_TAG"
	umamiURLEnvironment       = "UMAMI_URL"
	umamiWebsiteIDEnvironment = "UMAMI_WEBSITE_ID"
)

var siteContent = site.Content()
var policy *bluemonday.Policy

func init() {
	flag.StringVar(&flagPort, "port", "8001", "port to run the server on")
	flag.StringVar(&flagLog, "log", "info", "log level")
}

// Main runs the cowyo server command.
func Main() {
	flag.Parse()
	log.SetLevel(flagLog)

	if err := loadEnvironment(); err != nil {
		log.Errorf("loading .env: %s", err)
		os.Exit(1)
	}

	connections = make(map[string]*Connection)

	policy = bluemonday.UGCPolicy()

	var err error
	pageStore, err = database.Open(context.Background(), database.ConfigFromEnv())
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	defer pageStore.Close()
	log.Infof("using %s database", pageStore.Backend())

	// start server
	Serve()
}

func loadEnvironment(filenames ...string) error {
	err := godotenv.Load(filenames...)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

var indexTemplate *template.Template

func Serve() {
	// load go template from index.html
	// parse template from embed
	indexTemplate = template.Must(template.ParseFS(siteContent, "index.html"))

	log.Infof("listening on :%s", flagPort)
	http.HandleFunc("/", handler)
	http.ListenAndServe(fmt.Sprintf(":%s", flagPort), nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	t := time.Now().UTC()
	// Redirect URLs with trailing slashes (except for the root "/")
	if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/"), http.StatusPermanentRedirect)
		return
	}
	err := handle(w, r)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Infof("%v %v %v %s\n", r.RemoteAddr, r.Method, r.URL.Path, time.Since(t))
}

type Page struct {
	Title        string `json:"title"`
	Text         string `json:"text"`
	CursorStart  int64  `json:"cursor_start"`
	CursorEnd    int64  `json:"cursor_end"`
	Published    bool   `json:"published"`
	SelfDestruct bool   `json:"self_destruct"`
	Locked       bool   `json:"locked"`
}

type pageTemplateData struct {
	Page
	GoogleTag      string
	UmamiURL       string
	UmamiWebsiteID string
	Landing        bool
	About          bool
	SEO            seoTemplateData
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}

func handle(w http.ResponseWriter, r *http.Request) (err error) {
	if r.URL.Path == "/ws" {
		return handleWebsocket(w, r)
	} else if r.URL.Path == apiRoot || strings.HasPrefix(r.URL.Path, apiPrefix) {
		return handleAPI(w, r)
	} else if r.URL.Path == "/sitemap.xml" {
		return handleSitemap(w, r)
	} else if r.URL.Path == "/robots.txt" {
		return handleRobots(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/static") {
		// serve static file from embed
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Cache-Control", "must-revalidate")
		w.Header().Set("Cache-Control", "proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.FileServer(http.FS(siteContent)).ServeHTTP(w, r)
		return
	} else if r.URL.Path == "/about" {
		return handleAbout(w, r)
	} else if r.Method == http.MethodPost {
		return handlePost(w, r)
	} else if r.URL.Path == "/" {
		if isCurlRequest(r) || r.URL.Query().Has("new") {
			return redirectToRandomDocument(w, r)
		}
		return handleLanding(w, r)
	}
	key := r.URL.Path[1:]
	p := Page{Title: key}
	var storedPage database.Page
	consumeOnLoad := r.Method == http.MethodGet
	if consumeOnLoad {
		pageMutationMu.Lock()
		storedPage, err = pageStore.ConsumePage(r.Context(), key)
		pageMutationMu.Unlock()
	} else {
		storedPage, err = pageStore.GetPage(r.Context(), key)
	}
	if err != nil && !errors.Is(err, database.ErrPageNotFound) {
		return err
	}
	if err == nil {
		p = Page{
			Title:        storedPage.Title,
			Text:         storedPage.Text,
			CursorStart:  storedPage.CursorStart,
			CursorEnd:    storedPage.CursorEnd,
			Published:    storedPage.Published,
			SelfDestruct: storedPage.SelfDestruct,
			Locked:       storedPage.Locked,
		}
		if p.SelfDestruct {
			p.Published = false
			if consumeOnLoad {
				logPageEdit(key, "http-get", operationSelfDestruct, len(p.Text))
			}
		}
	}
	log.Tracef("loading: %+v", p)
	w.Header().Set("Vary", "User-Agent")
	if p.SelfDestruct {
		w.Header().Set("Cache-Control", "no-store")
	}
	if p.Published {
		w.Header().Set("X-Robots-Tag", robotsDirective(true))
	} else {
		w.Header().Set("X-Robots-Tag", robotsDirective(false))
	}
	seo, err := buildPageSEO(r, p)
	if err != nil {
		return fmt.Errorf("build page metadata: %w", err)
	}
	w.Header().Set("Content-Language", "en-US")
	w.Header().Set(
		"Link",
		fmt.Sprintf("<%s>; rel=\"canonical\"", postedDocumentURL(r, p.Title)),
	)
	if isCurlRequest(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, err = fmt.Fprint(w, p.Text)
		return
	}
	p.Text = html.EscapeString(p.Text)
	p.Text = policy.Sanitize(p.Text)
	data := pageTemplateData{
		Page:      p,
		GoogleTag: configuredGoogleTag(),
		SEO:       seo,
	}
	data.UmamiURL, data.UmamiWebsiteID = configuredUmamiTracker()
	return indexTemplate.Execute(w, data)
}

func handleLanding(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	seo, err := buildLandingSEO(r)
	if err != nil {
		return fmt.Errorf("build landing page metadata: %w", err)
	}

	w.Header().Set("Content-Language", "en-US")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", postedDocumentURL(r, "")))
	w.Header().Set("Vary", "User-Agent")
	w.Header().Set("X-Robots-Tag", robotsDirective(true))
	data := pageTemplateData{
		GoogleTag: configuredGoogleTag(),
		Landing:   true,
		SEO:       seo,
	}
	data.UmamiURL, data.UmamiWebsiteID = configuredUmamiTracker()
	return indexTemplate.Execute(w, data)
}

func handleAbout(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	seo, err := buildAboutSEO(r)
	if err != nil {
		return fmt.Errorf("build about page metadata: %w", err)
	}

	w.Header().Set("Content-Language", "en-US")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", postedDocumentURL(r, "about")))
	w.Header().Set("X-Robots-Tag", robotsDirective(true))
	data := pageTemplateData{
		GoogleTag: configuredGoogleTag(),
		About:     true,
		SEO:       seo,
	}
	data.UmamiURL, data.UmamiWebsiteID = configuredUmamiTracker()
	return indexTemplate.Execute(w, data)
}

func redirectToRandomDocument(w http.ResponseWriter, r *http.Request) error {
	name, err := randomDocumentName()
	if err != nil {
		return fmt.Errorf("generate random paste name: %w", err)
	}
	http.Redirect(w, r, "/"+name, http.StatusFound)
	return nil
}

var googleTagPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

func configuredGoogleTag() string {
	tag := strings.TrimSpace(os.Getenv(googleTagEnvironment))
	if !googleTagPattern.MatchString(tag) {
		return ""
	}
	return tag
}

var umamiWebsiteIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

func configuredUmamiTracker() (string, string) {
	configuredURL := strings.TrimSpace(os.Getenv(umamiURLEnvironment))
	websiteID := strings.TrimSpace(os.Getenv(umamiWebsiteIDEnvironment))
	if configuredURL == "" || !umamiWebsiteIDPattern.MatchString(websiteID) {
		return "", ""
	}

	parsed, err := url.Parse(configuredURL)
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", ""
	}

	return html.EscapeString(parsed.Scheme + "://" + parsed.Host), websiteID
}

func handleSitemap(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	titles, err := pageStore.ListPublishedPageTitles(r.Context())
	if err != nil {
		return fmt.Errorf("list published pages: %w", err)
	}

	urls := make([]sitemapURL, 0, len(titles)+2)
	urls = append(
		urls,
		sitemapURL{Location: postedDocumentURL(r, "")},
		sitemapURL{Location: postedDocumentURL(r, "about")},
	)
	for _, title := range titles {
		if title == "about" {
			continue
		}
		urls = append(urls, sitemapURL{
			Location: postedDocumentURL(r, title),
		})
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		return nil
	}

	if _, err := fmt.Fprint(w, xml.Header); err != nil {
		return err
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	return encoder.Encode(sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	})
}

func handleRobots(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		return nil
	}
	_, err := fmt.Fprintf(
		w,
		"User-agent: *\nAllow: /\nSitemap: %s\n",
		postedDocumentURL(r, "sitemap.xml"),
	)
	return err
}

func isCurlRequest(r *http.Request) bool {
	return strings.HasPrefix(strings.ToLower(r.UserAgent()), "curl/")
}

var (
	errPageLocked              = errors.New("page is locked")
	errSelfDestructArmed       = errors.New("page self destruct is armed")
	errDecryptChangedPlaintext = errors.New(
		"decryption must preserve text outside complete encrypted blocks",
	)
)

type pageUpdate struct {
	Text        string
	CursorStart int64
	CursorEnd   int64
	Operation   string
	Password    string
}

type pageUpdateOptions struct {
	requireExisting  bool
	preserveLockText bool
	source           string
}

func applyWebsocketUpdate(
	ctx context.Context,
	place string,
	update pageUpdate,
) (database.Page, error) {
	return applyPageUpdate(ctx, place, update, pageUpdateOptions{
		source: "websocket",
	})
}

func applyAPIUpdate(
	ctx context.Context,
	place string,
	update pageUpdate,
) (database.Page, error) {
	return applyPageUpdate(ctx, place, update, pageUpdateOptions{
		requireExisting:  true,
		preserveLockText: true,
		source:           "http-api",
	})
}

func applyPageUpdate(
	ctx context.Context,
	place string,
	update pageUpdate,
	options pageUpdateOptions,
) (database.Page, error) {
	pageMutationMu.Lock()
	defer pageMutationMu.Unlock()

	stored, err := pageStore.GetPage(ctx, place)
	if errors.Is(err, database.ErrPageNotFound) {
		if options.requireExisting {
			return database.Page{}, database.ErrPageNotFound
		}
		stored = database.Page{Title: place}
		err = nil
	} else if err != nil {
		return database.Page{}, fmt.Errorf("load page: %w", err)
	}

	next := database.Page{
		Title:        place,
		Text:         update.Text,
		CursorStart:  update.CursorStart,
		CursorEnd:    update.CursorEnd,
		Published:    stored.Published,
		SelfDestruct: stored.SelfDestruct,
		Locked:       stored.Locked,
		LockSalt:     stored.LockSalt,
		LockVerifier: stored.LockVerifier,
	}
	switch update.Operation {
	case "":
		if stored.Locked {
			return stored, errPageLocked
		}
	case operationLock:
		if stored.Locked {
			err = errPageAlreadyLocked
		} else {
			if options.preserveLockText {
				next.Text = stored.Text
				next.CursorStart = stored.CursorStart
				next.CursorEnd = stored.CursorEnd
			}
			var credentials pageLockCredentials
			credentials, err = createPageLock(update.Password)
			if err == nil {
				next.Locked = true
				next.SelfDestruct = false
				next.LockSalt = credentials.salt
				next.LockVerifier = credentials.verifier
			}
		}
	case operationUnlock:
		if !stored.Locked {
			err = errPageNotLocked
			break
		}
		err = verifyPageLock(pageLockCredentials{
			salt:     stored.LockSalt,
			verifier: stored.LockVerifier,
		}, update.Password)
		if err == nil {
			next.Text = stored.Text
			if options.preserveLockText {
				next.CursorStart = stored.CursorStart
				next.CursorEnd = stored.CursorEnd
			}
			next.Locked = false
			next.LockSalt = ""
			next.LockVerifier = ""
		}
	case operationEncrypt, operationDecrypt:
		err = validateCryptoUpdate(
			stored.Text,
			update.Text,
			update.Operation,
			stored.Locked,
		)
	case operationPublish, operationUnpublish:
		if stored.Locked {
			err = errPageLocked
		} else if update.Operation == operationPublish && stored.SelfDestruct {
			err = errSelfDestructArmed
		} else {
			next.Text = stored.Text
			next.CursorStart = stored.CursorStart
			next.CursorEnd = stored.CursorEnd
			next.Published = update.Operation == operationPublish
		}
	case operationSelfDestruct, operationCancelSelfDestruct:
		if stored.Locked {
			err = errPageLocked
		} else {
			next.Text = stored.Text
			next.CursorStart = stored.CursorStart
			next.CursorEnd = stored.CursorEnd
			next.SelfDestruct = update.Operation == operationSelfDestruct
			if next.SelfDestruct {
				next.Published = false
			}
		}
	default:
		err = errors.New("unsupported page operation")
	}
	if err != nil {
		return stored, err
	}

	if err := pageStore.UpsertPage(ctx, next); err != nil {
		return database.Page{}, fmt.Errorf("save page: %w", err)
	}
	logPageEdit(place, options.source, pageOperation(update.Operation), len(next.Text))
	return next, nil
}

func validateCryptoUpdate(current, next, operation string, locked bool) error {
	if !locked {
		return nil
	}

	switch operation {
	case operationEncrypt:
		return errPageLocked
	case operationDecrypt:
		if !preservesTextOutsideEncryptedBlocks(current, next) {
			return errDecryptChangedPlaintext
		}
	}
	return nil
}

const encryptedBlockStart = "-----BEGIN COWYO ENCRYPTED BLOCK V1-----"
const encryptedBlockEnd = "-----END COWYO ENCRYPTED BLOCK V1-----"

var encryptedBlockPattern = regexp.MustCompile(
	regexp.QuoteMeta(encryptedBlockStart) +
		`\r?\n[^\r\n]+\r?\n` +
		regexp.QuoteMeta(encryptedBlockEnd),
)

type encryptedBlockEnvelope struct {
	Version    int    `json:"v"`
	KDF        string `json:"kdf"`
	N          int    `json:"N"`
	R          int    `json:"r"`
	P          int    `json:"p"`
	Salt       string `json:"salt"`
	Cipher     string `json:"cipher"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"data"`
}

func validEncryptedDocument(text string) bool {
	match := encryptedBlockPattern.FindStringIndex(text)
	return match != nil &&
		match[0] == 0 &&
		match[1] == len(text) &&
		validEncryptedBlock(text)
}

func hasValidEncryptedBlock(text string) bool {
	for _, block := range encryptedBlockPattern.FindAllString(text, -1) {
		if validEncryptedBlock(block) {
			return true
		}
	}
	return false
}

func validEncryptedBlock(block string) bool {
	payload := strings.TrimPrefix(block, encryptedBlockStart)
	payload = strings.TrimSuffix(payload, encryptedBlockEnd)
	payload = strings.TrimSpace(payload)

	var envelope encryptedBlockEnvelope
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	if envelope.Version != 1 ||
		envelope.KDF != "scrypt" ||
		envelope.N != 1<<16 ||
		envelope.R != 8 ||
		envelope.P != 1 ||
		envelope.Cipher != "xchacha20-poly1305" {
		return false
	}

	salt, err := base64.RawURLEncoding.DecodeString(envelope.Salt)
	if err != nil || len(salt) != 16 {
		return false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != 24 {
		return false
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Ciphertext)
	return err == nil && len(ciphertext) >= 16
}

func preservesTextOutsideEncryptedBlocks(current, next string) bool {
	ranges := encryptedBlockPattern.FindAllStringIndex(current, -1)
	if len(ranges) == 0 {
		return false
	}

	prefix := current[:ranges[0][0]]
	suffix := current[ranges[len(ranges)-1][1]:]
	if !strings.HasPrefix(next, prefix) || !strings.HasSuffix(next, suffix) {
		return false
	}

	searchEnd := len(next) - len(suffix)
	position := len(prefix)
	for index := 0; index < len(ranges)-1; index++ {
		unchanged := current[ranges[index][1]:ranges[index+1][0]]
		found := strings.Index(next[position:searchEnd], unchanged)
		if found < 0 {
			return false
		}
		position += found + len(unchanged)
	}
	return true
}

func websocketErrorMessage(err error) string {
	switch {
	case errors.Is(err, errPageLocked):
		return "This page is locked."
	case errors.Is(err, errPageAlreadyLocked):
		return "This page is already locked."
	case errors.Is(err, errPageNotLocked):
		return "This page is not locked."
	case errors.Is(err, errWrongLockPassword):
		return "Wrong password for this page lock."
	case errors.Is(err, errSelfDestructArmed):
		return "Cancel self destruct before publishing."
	default:
		return err.Error()
	}
}
