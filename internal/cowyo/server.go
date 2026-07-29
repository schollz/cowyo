package cowyo

import (
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
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
var connections map[string]Connection
var mu sync.Mutex
var pageMutationMu sync.Mutex

const googleTagEnvironment = "GOOGLE_TAG"

var siteContent = site.Content()
var policy *bluemonday.Policy

type Connection struct {
	conn  *websocket.Conn
	place string
}

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

	connections = make(map[string]Connection)

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
	Operation    string `json:"operation,omitempty"`
	Password     string `json:"password,omitempty"`
	Error        string `json:"error,omitempty"`
	Current      bool   `json:"current,omitempty"`
}

type pageTemplateData struct {
	Page
	GoogleTag string
	Landing   bool
	About     bool
	SEO       seoTemplateData
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
	return indexTemplate.Execute(w, pageTemplateData{
		Page:      p,
		GoogleTag: configuredGoogleTag(),
		SEO:       seo,
	})
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
	return indexTemplate.Execute(w, pageTemplateData{
		GoogleTag: configuredGoogleTag(),
		Landing:   true,
		SEO:       seo,
	})
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
	return indexTemplate.Execute(w, pageTemplateData{
		GoogleTag: configuredGoogleTag(),
		About:     true,
		SEO:       seo,
	})
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

var upgrader = websocket.Upgrader{} // use default options

func handleWebsocket(w http.ResponseWriter, r *http.Request) (err error) {
	// get the place from the query parameter
	query := r.URL.Query()
	log.Tracef("query: %+v", query)
	if _, ok := query["place"]; !ok {
		err = fmt.Errorf("no place")
		log.Error(err)
		return
	}
	place := query["place"][0]

	// use gorilla to open websocket
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// generate random string
	// this is used to identify the websocket connection
	// so that we can send updates to the correct websocket
	// connection
	idCurrent := RandStringBytesMaskImprSrc(32)
	mu.Lock()
	connections[idCurrent] = Connection{conn: c, place: place}
	mu.Unlock()

	defer func() {
		mu.Lock()
		// delete connection
		delete(connections, idCurrent)
		mu.Unlock()
	}()

	for {
		var p Page
		err := c.ReadJSON(&p)
		if err != nil {
			break
		}
		log.Tracef("updating '%s' with operation %q and %d bytes", place, p.Operation, len(p.Text))
		saved, updateErr := applyWebsocketUpdate(r.Context(), place, p)
		p.Password = ""
		err = updateErr
		if err != nil {
			log.Warnf("rejected update to %q: %s", place, err)
			writeWebsocketPage(c, Page{
				Title:        "error",
				Text:         saved.Text,
				Published:    saved.Published,
				SelfDestruct: saved.SelfDestruct,
				Locked:       saved.Locked,
				Operation:    p.Operation,
				Error:        websocketErrorMessage(err),
				Current:      saved.Title != "",
			})
		} else {
			mu.Lock()
			for id := range connections {
				if id == idCurrent || connections[id].place != place {
					continue
				}
				if writeErr := connections[id].conn.WriteJSON(Page{
					Title:        "update",
					Text:         saved.Text,
					Published:    saved.Published,
					SelfDestruct: saved.SelfDestruct,
					Locked:       saved.Locked,
					Operation:    p.Operation,
				}); writeErr != nil {
					log.Error(writeErr)
				}
			}
			mu.Unlock()
			writeWebsocketPage(c, Page{
				Title:        "ok",
				Text:         saved.Text,
				Published:    saved.Published,
				SelfDestruct: saved.SelfDestruct,
				Locked:       saved.Locked,
				Operation:    p.Operation,
			})
		}
	}
	return
}

const (
	operationLock               = "lock"
	operationUnlock             = "unlock"
	operationEncrypt            = "encrypt"
	operationDecrypt            = "decrypt"
	operationPublish            = "publish"
	operationUnpublish          = "unpublish"
	operationSelfDestruct       = "self-destruct"
	operationCancelSelfDestruct = "cancel-self-destruct"
)

var (
	errPageLocked        = errors.New("page is locked")
	errSelfDestructArmed = errors.New("page self destruct is armed")
)

func applyWebsocketUpdate(ctx context.Context, place string, update Page) (database.Page, error) {
	pageMutationMu.Lock()
	defer pageMutationMu.Unlock()

	stored, err := pageStore.GetPage(ctx, place)
	if errors.Is(err, database.ErrPageNotFound) {
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
	logPageEdit(place, "websocket", pageOperation(update.Operation), len(next.Text))
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
			return errors.New("decryption must preserve text outside complete encrypted blocks")
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

func writeWebsocketPage(conn *websocket.Conn, page Page) {
	mu.Lock()
	defer mu.Unlock()
	if err := conn.WriteJSON(page); err != nil {
		log.Error(err)
	}
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
