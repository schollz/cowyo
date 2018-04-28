package server

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	secretRequired "github.com/danielheath/gin-teeny-security"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jcelliott/lumber"
	"github.com/schollz/cowyo/encrypt"
)

const minutesToUnlock = 10.0

var defaultLock string
var debounceTime int
var diaryMode bool
var allowFileUploads bool
var maxUploadMB uint
var needSitemapUpdate = true
var pathToData string
var log *lumber.ConsoleLogger

type Site struct {
	PathToData           string
	Css                  []byte
	DefaultPage          string
	DefaultPassword      string
	Debounce             int
	Diary                bool
	SessionStore         sessions.Store
	SecretCode           string
	AllowInsecure        bool
	HotTemplateReloading bool
	Fileuploads          bool
	MaxUploadSize        uint
	Logger               *lumber.ConsoleLogger
}

func Serve(
	filepathToData,
	host,
	port,
	crt_path,
	key_path string,
	TLS bool,
	cssFile string,
	defaultPage string,
	defaultPassword string,
	debounce int,
	diary bool,
	secret string,
	secretCode string,
	allowInsecure bool,
	hotTemplateReloading bool,
	fileuploads bool,
	maxUploadSize uint,
	logger *lumber.ConsoleLogger,
) {
	var customCSS []byte
	// collect custom CSS
	if len(cssFile) > 0 {
		var errRead error
		customCSS, errRead = ioutil.ReadFile(cssFile)
		if errRead != nil {
			fmt.Println(errRead)
			return
		}
		fmt.Printf("Loaded CSS file, %d bytes\n", len(customCSS))
	}

	router := Site{
		filepathToData,
		customCSS,
		defaultPage,
		defaultPassword,
		debounce,
		diary,
		sessions.NewCookieStore([]byte(secret)),
		secretCode,
		allowInsecure,
		hotTemplateReloading,
		fileuploads,
		maxUploadSize,
		logger,
	}.Router()

	if TLS {
		http.ListenAndServeTLS(host+":"+port, crt_path, key_path, router)
	} else {
		panic(router.Run(host + ":" + port))
	}
}

func (s Site) Router() *gin.Engine {
	pathToData = s.PathToData
	allowFileUploads = s.Fileuploads
	maxUploadMB = s.MaxUploadSize
	log = s.Logger
	if log == nil {
		log = lumber.NewConsoleLogger(lumber.TRACE)
	}

	if s.HotTemplateReloading {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
		"sniffContentType": sniffContentType,
	})

	if s.HotTemplateReloading {
		router.LoadHTMLGlob("templates/*.tmpl")
	} else {
		router.HTMLRender = loadTemplates("index.tmpl")
	}

	router.Use(sessions.Sessions(s.PathToData, s.SessionStore))
	if s.SecretCode != "" {
		cfg := &secretRequired.Config{
			Secret: s.SecretCode,
			Path:   "/login/",
			RequireAuth: func(c *gin.Context) bool {
				page := c.Param("page")
				cmd := c.Param("command")

				if page == "sitemap.xml" || page == "favicon.ico" || page == "static" || page == "uploads" {
					return false // no auth for these
				}

				if page != "" && cmd == "/read" {
					p := Open(page)
					fmt.Printf("p: '%+v'\n", p)
					if p != nil && p.IsPublished {
						return false // Published pages don't require auth.
					}
				}
				return true
			},
		}
		router.Use(cfg.Middleware)
	}

	// router.Use(static.Serve("/static/", static.LocalFile("./static", true)))
	router.GET("/", func(c *gin.Context) {
		if s.DefaultPage != "" {
			c.Redirect(302, "/"+s.DefaultPage+"/read")
		} else {
			c.Redirect(302, "/"+randomAlliterateCombo())
		}
	})

	router.POST("/uploads", handleUpload)

	router.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		c.Redirect(302, "/"+page+"/")
	})
	router.GET("/:page/*command", s.handlePageRequest)
	router.POST("/update", handlePageUpdate)
	router.POST("/relinquish", handlePageRelinquish) // relinquish returns the page no matter what (and destroys if nessecary)
	router.POST("/exists", handlePageExists)
	router.POST("/prime", handlePrime)
	router.POST("/lock", handleLock)
	router.POST("/publish", handlePublish)
	router.POST("/encrypt", handleEncrypt)
	router.DELETE("/oldlist", handleClearOldListItems)
	router.DELETE("/listitem", deleteListItem)

	// start long-processes as threads
	go thread_SiteMap()

	// lock all pages automatically
	if s.DefaultPassword != "" {
		fmt.Println("running with locked pages")
		defaultLock = HashPassword(s.DefaultPassword)
	}

	// set the debounce time
	debounceTime = s.Debounce

	// set diary mode
	diaryMode = s.Diary

	// Allow iframe/scripts in markup?
	allowInsecureHtml = s.AllowInsecure
	return router
}

func loadTemplates(list ...string) multitemplate.Render {
	r := multitemplate.New()

	for _, x := range list {
		templateString, err := Asset("templates/" + x)
		if err != nil {
			panic(err)
		}

		tmplMessage, err := template.New(x).Funcs(template.FuncMap{
			"sniffContentType": sniffContentType,
		}).Parse(string(templateString))
		if err != nil {
			panic(err)
		}

		r.Add(x, tmplMessage)
	}

	return r
}

func pageIsLocked(p *Page, c *gin.Context) bool {
	// it is easier to reason about when the page is actually unlocked
	var unlocked = !p.IsLocked ||
		(p.IsLocked && p.UnlockedFor == getSetSessionID(c))
	return !unlocked
}

func handlePageRelinquish(c *gin.Context) {
	type QueryJSON struct {
		Page string `json:"page"`
	}
	var json QueryJSON
	err := c.BindJSON(&json)
	if err != nil {
		log.Trace(err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Wrong JSON"})
		return
	}
	if len(json.Page) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Must specify `page`"})
		return
	}
	message := "Relinquished"
	p := Open(json.Page)
	name := p.Meta
	if name == "" {
		name = json.Page
	}
	text := p.Text.GetCurrent()
	isLocked := pageIsLocked(p, c)
	isEncrypted := p.IsEncrypted
	destroyed := p.IsPrimedForSelfDestruct
	if !isLocked && p.IsPrimedForSelfDestruct {
		p.Erase()
		message = "Relinquished and erased"
	}
	c.JSON(http.StatusOK, gin.H{"success": true,
		"name":      name,
		"message":   message,
		"text":      text,
		"locked":    isLocked,
		"encrypted": isEncrypted,
		"destroyed": destroyed})
}

func getSetSessionID(c *gin.Context) (sid string) {
	var (
		session = sessions.Default(c)
		v       = session.Get("sid")
	)
	if v != nil {
		sid = v.(string)
	}
	if v == nil || sid == "" {
		sid = RandStringBytesMaskImprSrc(8)
		session.Set("sid", sid)
		session.Save()
	}
	return sid
}

func thread_SiteMap() {
	for {
		if needSitemapUpdate {
			log.Info("Generating sitemap...")
			needSitemapUpdate = false
			ioutil.WriteFile(path.Join(pathToData, "sitemap.xml"), []byte(generateSiteMap()), 0644)
			log.Info("..finished generating sitemap")
		}
		time.Sleep(time.Second)
	}
}

func generateSiteMap() (sitemap string) {
	files, _ := ioutil.ReadDir(pathToData)
	lastEdited := make([]string, len(files))
	names := make([]string, len(files))
	i := 0
	for _, f := range files {
		names[i] = DecodeFileName(f.Name())
		p := Open(names[i])
		if p.IsPublished {
			lastEdited[i] = time.Unix(p.Text.LastEditTime()/1000000000, 0).Format("2006-01-02")
			i++
		}
	}
	names = names[:i]
	lastEdited = lastEdited[:i]
	sitemap = ""
	for i := range names {
		sitemap += fmt.Sprintf(`
	<url>
	<loc>{{ .Request.Host }}/%s/read</loc>
	<lastmod>%s</lastmod>
	<changefreq>monthly</changefreq>
	<priority>0.8</priority>
	</url>
	`, names[i], lastEdited[i])
	}
	sitemap += "</urlset>"
	return
}

func (s Site) handlePageRequest(c *gin.Context) {
	page := c.Param("page")
	command := c.Param("command")

	if page == "sitemap.xml" {
		siteMap, err := ioutil.ReadFile(path.Join(pathToData, "sitemap.xml"))
		if err != nil {
			c.Data(http.StatusInternalServerError, contentType("sitemap.xml"), []byte(""))
		} else {
			fmt.Fprintln(c.Writer, `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
			template.Must(template.New("sitemap").Parse(string(siteMap))).Execute(c.Writer, c)
		}
		return
	} else if page == "favicon.ico" {
		data, _ := Asset("/static/img/cowyo/favicon.ico")
		c.Data(http.StatusOK, contentType("/static/img/cowyo/favicon.ico"), data)
		return
	} else if page == "static" {
		filename := page + command
		var data []byte
		if filename == "static/css/custom.css" {
			data = s.Css
		} else {
			var errAssset error
			data, errAssset = Asset(filename)
			if errAssset != nil {
				c.String(http.StatusInternalServerError, "Could not find data")
			}
		}
		c.Data(http.StatusOK, contentType(filename), data)
		return
	} else if page == "uploads" {
		if len(command) == 0 || command == "/" || command == "/edit" {
			if !allowFileUploads {
				c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Uploads are disabled on this server"))
				return
			}
		} else {
			command = command[1:]
			if !strings.HasSuffix(command, ".upload") {
				command = command + ".upload"
			}
			pathname := path.Join(pathToData, command)

			if allowInsecureHtml {
				c.Header(
					"Content-Disposition",
					`inline; filename="`+c.DefaultQuery("filename", "upload")+`"`,
				)
			} else {
				// Prevent malicious html uploads by forcing type to plaintext and 'download-instead-of-view'
				c.Header("Content-Type", "text/plain")
				c.Header(
					"Content-Disposition",
					`attachment; filename="`+c.DefaultQuery("filename", "upload")+`"`,
				)
			}
			c.File(pathname)
			return
		}
	}

	p := Open(page)
	if len(command) < 2 {
		if p.IsPublished {
			c.Redirect(302, "/"+page+"/read")
		} else {
			c.Redirect(302, "/"+page+"/edit")
		}
		return
	}

	// use the default lock
	if defaultLock != "" && p.IsNew() {
		p.IsLocked = true
		p.PassphraseToUnlock = defaultLock
	}

	version := c.DefaultQuery("version", "ajksldfjl")
	isLocked := pageIsLocked(p, c)

	// Disallow anything but viewing locked/encrypted pages
	if (p.IsEncrypted || isLocked) &&
		(command[0:2] != "/v" && command[0:2] != "/r") {
		c.Redirect(302, "/"+page+"/view")
		return
	}

	// Destroy page if it is opened and primed
	if p.IsPrimedForSelfDestruct && !isLocked && !p.IsEncrypted {
		p.Update("<center><em>This page has self-destructed. You cannot return to it.</em></center>\n\n" + p.Text.GetCurrent())
		p.Erase()
		if p.IsPublished {
			command = "/read"
		} else {
			command = "/view"
		}
	}
	if command == "/erase" {
		if !isLocked && !p.IsEncrypted {
			p.Erase()
			c.Redirect(302, "/"+page+"/edit")
		} else {
			c.Redirect(302, "/"+page+"/view")
		}
		return
	}
	rawText := p.Text.GetCurrent()
	rawHTML := p.RenderedPage

	// Check to see if an old version is requested
	versionInt, versionErr := strconv.Atoi(version)
	if versionErr == nil && versionInt > 0 {
		versionText, err := p.Text.GetPreviousByTimestamp(int64(versionInt))
		if err == nil {
			rawText = versionText
			rawHTML = GithubMarkdownToHTML(rawText)
		}
	}

	// Get history
	var versionsInt64 []int64
	var versionsChangeSums []int
	var versionsText []string
	if command[0:2] == "/h" {
		versionsInt64, versionsChangeSums = p.Text.GetMajorSnapshotsAndChangeSums(60) // get snapshots 60 seconds apart
		versionsText = make([]string, len(versionsInt64))
		for i, v := range versionsInt64 {
			versionsText[i] = time.Unix(v/1000000000, 0).Format("Mon Jan 2 15:04:05 MST 2006")
		}
		versionsText = reverseSliceString(versionsText)
		versionsInt64 = reverseSliceInt64(versionsInt64)
		versionsChangeSums = reverseSliceInt(versionsChangeSums)
	}

	if len(command) > 3 && command[0:3] == "/ra" {
		c.Writer.Header().Set("Content-Type", "text/plain")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Data(200, contentType(p.Name), []byte(rawText))
		return
	}

	var DirectoryEntries []os.FileInfo
	if page == "ls" {
		command = "/view"
		DirectoryEntries = DirectoryList()
	}
	if page == "uploads" {
		command = "/view"
		var err error
		DirectoryEntries, err = UploadList()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	// swap out /view for /read if it is published
	if p.IsPublished {
		rawHTML = strings.Replace(rawHTML, "/view", "/read", -1)
	}

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"EditPage":    command[0:2] == "/e", // /edit
		"ViewPage":    command[0:2] == "/v", // /view
		"ListPage":    command[0:2] == "/l", // /list
		"HistoryPage": command[0:2] == "/h", // /history
		"ReadPage":    command[0:2] == "/r", // /history
		"DontKnowPage": command[0:2] != "/e" &&
			command[0:2] != "/v" &&
			command[0:2] != "/l" &&
			command[0:2] != "/r" &&
			command[0:2] != "/h",
		"DirectoryPage":      page == "ls" || page == "uploads",
		"UploadPage":         page == "uploads",
		"DirectoryEntries":   DirectoryEntries,
		"Page":               page,
		"RenderedPage":       template.HTML([]byte(rawHTML)),
		"RawPage":            rawText,
		"Versions":           versionsInt64,
		"VersionsText":       versionsText,
		"VersionsChangeSums": versionsChangeSums,
		"IsLocked":           isLocked,
		"IsEncrypted":        p.IsEncrypted,
		"ListItems":          renderList(rawText),
		"Route":              "/" + page + command,
		"HasDotInName":       strings.Contains(page, "."),
		"RecentlyEdited":     getRecentlyEdited(page, c),
		"IsPublished":        p.IsPublished,
		"CustomCSS":          len(s.Css) > 0,
		"Debounce":           debounceTime,
		"DiaryMode":          diaryMode,
		"Date":               time.Now().Format("2006-01-02"),
		"UnixTime":           time.Now().Unix(),
		"ChildPageNames":     p.ChildPageNames(),
		"AllowFileUploads":   allowFileUploads,
		"MaxUploadMB":        maxUploadMB,
	})
}

func getRecentlyEdited(title string, c *gin.Context) []string {
	session := sessions.Default(c)
	var recentlyEdited string
	v := session.Get("recentlyEdited")
	editedThings := []string{}
	if v == nil {
		recentlyEdited = title
	} else {
		editedThings = strings.Split(v.(string), "|||")
		if !stringInSlice(title, editedThings) {
			recentlyEdited = v.(string) + "|||" + title
		} else {
			recentlyEdited = v.(string)
		}
	}
	session.Set("recentlyEdited", recentlyEdited)
	session.Save()
	editedThingsWithoutCurrent := make([]string, len(editedThings))
	i := 0
	for _, thing := range editedThings {
		if thing == title {
			continue
		}
		if strings.Contains(thing, "icon-") {
			continue
		}
		editedThingsWithoutCurrent[i] = thing
		i++
	}
	return editedThingsWithoutCurrent[:i]
}

func handlePageExists(c *gin.Context) {
	type QueryJSON struct {
		Page string `json:"page"`
	}
	var json QueryJSON
	err := c.BindJSON(&json)
	if err != nil {
		log.Trace(err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Wrong JSON", "exists": false})
		return
	}
	p := Open(json.Page)
	if len(p.Text.GetCurrent()) > 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": json.Page + " found", "exists": true})
	} else {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": json.Page + " not found", "exists": false})
	}

}

func handlePageUpdate(c *gin.Context) {
	type QueryJSON struct {
		Page        string `json:"page"`
		NewText     string `json:"new_text"`
		FetchedAt   int64  `json:"fetched_at"`
		IsEncrypted bool   `json:"is_encrypted"`
		IsPrimed    bool   `json:"is_primed"`
		Meta        string `json:"meta"`
	}
	var json QueryJSON
	err := c.BindJSON(&json)
	if err != nil {
		log.Trace(err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Wrong JSON"})
		return
	}
	if len(json.NewText) > 100000000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Too much"})
		return
	}
	if len(json.Page) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Must specify `page`"})
		return
	}
	log.Trace("Update: %v", json)
	p := Open(json.Page)
	var (
		message       string
		sinceLastEdit = time.Since(p.LastEditTime())
	)
	success := false
	if pageIsLocked(p, c) {
		if sinceLastEdit < minutesToUnlock {
			message = "This page is being edited by someone else"
		} else {
			// here what might have happened is that two people unlock without
			// editing thus they both suceeds but only one is able to edit
			message = "Locked, must unlock first"
		}
	} else if p.IsEncrypted {
		message = "Encrypted, must decrypt first"
	} else if json.FetchedAt > 0 && p.LastEditUnixTime() > json.FetchedAt {
		message = "Refusing to overwrite others work"
	} else {
		p.Meta = json.Meta
		p.Update(json.NewText)
		if json.IsEncrypted {
			p.IsEncrypted = true
		}
		if json.IsPrimed {
			p.IsPrimedForSelfDestruct = true
		}
		p.Save()
		message = "Saved"
		if p.IsPublished {
			needSitemapUpdate = true
		}
		success = true
	}
	c.JSON(http.StatusOK, gin.H{"success": success, "message": message, "unix_time": time.Now().Unix()})
}

func handlePrime(c *gin.Context) {
	type QueryJSON struct {
		Page string `json:"page"`
	}
	var json QueryJSON
	if c.BindJSON(&json) != nil {
		c.String(http.StatusBadRequest, "Problem binding keys")
		return
	}
	log.Trace("Update: %v", json)
	p := Open(json.Page)
	if pageIsLocked(p, c) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Locked"})
		return
	} else if p.IsEncrypted {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Encrypted"})
		return
	}
	p.IsPrimedForSelfDestruct = true
	p.Save()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Primed"})
}

func handleLock(c *gin.Context) {
	type QueryJSON struct {
		Page       string `json:"page"`
		Passphrase string `json:"passphrase"`
	}

	var json QueryJSON
	if c.BindJSON(&json) != nil {
		c.String(http.StatusBadRequest, "Problem binding keys")
		return
	}
	p := Open(json.Page)
	if defaultLock != "" && p.IsNew() {
		p.IsLocked = true // IsLocked was replaced by variable wrt Context
		p.PassphraseToUnlock = defaultLock
	}

	if p.IsEncrypted {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Encrypted"})
		return
	}
	var (
		message       string
		sessionID     = getSetSessionID(c)
		sinceLastEdit = time.Since(p.LastEditTime())
	)

	// both lock/unlock ends here on locked&timeout combination
	if p.IsLocked &&
		p.UnlockedFor != sessionID &&
		p.UnlockedFor != "" &&
		sinceLastEdit.Minutes() < minutesToUnlock {

		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("This page is being edited by someone else! Will unlock automatically %2.0f minutes after the last change.", minutesToUnlock-sinceLastEdit.Minutes()),
		})
		return
	}
	if !pageIsLocked(p, c) {
		p.IsLocked = true
		p.PassphraseToUnlock = HashPassword(json.Passphrase)
		p.UnlockedFor = ""
		message = "Locked"
	} else {
		err2 := CheckPasswordHash(json.Passphrase, p.PassphraseToUnlock)
		if err2 != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Can't unlock"})
			return
		}
		p.UnlockedFor = sessionID
		message = "Unlocked only for you"
	}
	p.Save()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func handlePublish(c *gin.Context) {
	type QueryJSON struct {
		Page    string `json:"page"`
		Publish bool   `json:"publish"`
	}

	var json QueryJSON
	if c.BindJSON(&json) != nil {
		c.String(http.StatusBadRequest, "Problem binding keys")
		return
	}
	p := Open(json.Page)
	p.IsPublished = json.Publish
	p.Save()
	message := "Published"
	if !p.IsPublished {
		message = "Unpublished"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func handleUpload(c *gin.Context) {
	if !allowFileUploads {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Uploads are disabled on this server"))
		return
	}

	file, info, err := c.Request.FormFile("file")
	defer file.Close()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	newName := "sha256-" + encodeBytesToBase32(h.Sum(nil))

	// Replaces any existing version, but sha256 collisions are rare as anything.
	outfile, err := os.Create(path.Join(pathToData, newName+".upload"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	file.Seek(0, io.SeekStart)
	_, err = io.Copy(outfile, file)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Header("Location", "/uploads/"+newName+"?filename="+url.QueryEscape(info.Filename))
	return
}

func handleEncrypt(c *gin.Context) {
	type QueryJSON struct {
		Page       string `json:"page"`
		Passphrase string `json:"passphrase"`
	}

	var json QueryJSON
	if c.BindJSON(&json) != nil {
		c.String(http.StatusBadRequest, "Problem binding keys")
		return
	}
	p := Open(json.Page)
	if pageIsLocked(p, c) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Locked"})
		return
	}
	q := Open(json.Page)
	var message string
	if p.IsEncrypted {
		decrypted, err2 := encrypt.DecryptString(p.Text.GetCurrent(), json.Passphrase)
		if err2 != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Wrong password"})
			return
		}
		q.Erase()
		q = Open(json.Page)
		q.Update(decrypted)
		q.IsEncrypted = false
		q.IsLocked = p.IsLocked
		q.IsPrimedForSelfDestruct = p.IsPrimedForSelfDestruct
		message = "Decrypted"
	} else {
		currentText := p.Text.GetCurrent()
		encrypted, _ := encrypt.EncryptString(currentText, json.Passphrase)
		q.Erase()
		q = Open(json.Page)
		q.Update(encrypted)
		q.IsEncrypted = true
		q.IsLocked = p.IsLocked
		q.IsPrimedForSelfDestruct = p.IsPrimedForSelfDestruct
		message = "Encrypted"
	}
	q.Save()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func deleteListItem(c *gin.Context) {
	lineNum, err := strconv.Atoi(c.DefaultQuery("lineNum", "None"))
	page := c.Query("page") // shortcut for c.Request.URL.Query().Get("lastname")
	if err == nil {
		p := Open(page)

		_, listItems := reorderList(p.Text.GetCurrent())
		newText := p.Text.GetCurrent()
		for i, lineString := range listItems {
			// fmt.Println(i, lineString, lineNum)
			if i+1 == lineNum {
				// fmt.Println("MATCHED")
				if strings.Contains(lineString, "~~") == false {
					// fmt.Println(p.Text, "("+lineString[2:]+"\n"+")", "~~"+lineString[2:]+"~~"+"\n")
					newText = strings.Replace(newText+"\n", lineString[2:]+"\n", "~~"+strings.TrimSpace(lineString[2:])+"~~"+"\n", 1)
				} else {
					newText = strings.Replace(newText+"\n", lineString[2:]+"\n", lineString[4:len(lineString)-2]+"\n", 1)
				}
				p.Update(newText)
				break
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Done.",
		})
	} else {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
	}
}

func handleClearOldListItems(c *gin.Context) {
	type QueryJSON struct {
		Page string `json:"page"`
	}

	var json QueryJSON
	if c.BindJSON(&json) != nil {
		c.String(http.StatusBadRequest, "Problem binding keys")
		return
	}
	p := Open(json.Page)
	if p.IsEncrypted {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Encrypted"})
		return
	}
	if pageIsLocked(p, c) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Locked"})
		return
	}
	lines := strings.Split(p.Text.GetCurrent(), "\n")
	newLines := make([]string, len(lines))
	newLinesI := 0
	for _, line := range lines {
		if strings.Count(line, "~~") != 2 {
			newLines[newLinesI] = line
			newLinesI++
		}
	}
	p.Update(strings.Join(newLines[0:newLinesI], "\n"))
	p.Save()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Cleared"})
}