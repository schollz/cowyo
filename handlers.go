package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	// "github.com/gin-contrib/static"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/schollz/cowyo/encrypt"
)

var customCSS []byte
var defaultLock string

func serve(host, port, crt_path, key_path string, TLS bool, cssFile string, defaultPage string, defaultPassword string) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))
	router.HTMLRender = loadTemplates("index.tmpl")
	// router.Use(static.Serve("/static/", static.LocalFile("./static", true)))
	router.GET("/", func(c *gin.Context) {
		if defaultPage != "" {
			c.Redirect(302, "/"+defaultPage+"/read")
		} else {
			c.Redirect(302, "/"+randomAlliterateCombo())
		}
	})
	router.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		c.Redirect(302, "/"+page+"/")
	})
	router.GET("/:page/*command", handlePageRequest)
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

	// collect custom CSS
	if len(cssFile) > 0 {
		var errRead error
		customCSS, errRead = ioutil.ReadFile(cssFile)
		if errRead != nil {
			fmt.Println(errRead.Error())
			return
		}
		fmt.Printf("Loaded CSS file, %d bytes\n", len(customCSS))
	}

	if defaultPassword != "" {
		fmt.Println("running with locked pages")
		defaultLock = HashPassword(defaultPassword)
	}

	if TLS {
		http.ListenAndServeTLS(host+":"+port, crt_path, key_path, router)
	} else {
		router.Run(host + ":" + port)
	}
}

func loadTemplates(list ...string) multitemplate.Render {
	r := multitemplate.New()

	for _, x := range list {
		templateString, err := Asset("templates/" + x)
		if err != nil {
			panic(err)
		}

		tmplMessage, err := template.New(x).Parse(string(templateString))
		if err != nil {
			panic(err)
		}

		r.Add(x, tmplMessage)
	}

	return r
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
	isLocked := p.IsEncrypted
	isEncrypted := p.IsEncrypted
	destroyed := p.IsPrimedForSelfDestruct
	if !p.IsLocked && p.IsPrimedForSelfDestruct {
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

func thread_SiteMap() {
	for {
		log.Info("Generating sitemap...")
		ioutil.WriteFile(path.Join(pathToData, "sitemap.xml"), []byte(generateSiteMap()), 0644)
		log.Info("..finished generating sitemap")
		time.Sleep(24 * time.Hour)
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
	sitemap = `<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
	for i := range names {
		sitemap += fmt.Sprintf(`
	<url>
	<loc>https://cowyo.com/%s/view</loc>
	<lastmod>%s</lastmod>
	<changefreq>monthly</changefreq>
	<priority>0.8</priority>
	</url>
	`, names[i], lastEdited[i])
	}
	sitemap += "</urlset>"
	return
}

func handlePageRequest(c *gin.Context) {
	page := c.Param("page")
	command := c.Param("command")

	if page == "sitemap.xml" {
		siteMap, err := ioutil.ReadFile(path.Join(pathToData, "sitemap.xml"))
		if err != nil {
			c.Data(http.StatusInternalServerError, contentType("sitemap.xml"), []byte(""))
		} else {
			c.Data(http.StatusOK, contentType("sitemap.xml"), siteMap)
		}
		return
	} else if page == "favicon.ico" {
		data, _ := Asset("/static/img/cowyo/favicon.ico")
		c.Data(http.StatusOK, contentType("/static/img/cowyo/favicon.ico"), data)
		return
	} else if page == "static" {
		filename := page + command
		var data []byte
		fmt.Println(filename)
		if filename == "static/css/custom.css" {
			data = customCSS
		} else {
			var errAssset error
			data, errAssset = Asset(filename)
			if errAssset != nil {
				c.String(http.StatusInternalServerError, "Could not find data")
			}
		}
		c.Data(http.StatusOK, contentType(filename), data)
		return
	}
	p := Open(page)
	fmt.Println(command)
	if len(command) < 2 {
		fmt.Println(p.IsPublished)
		if p.IsPublished {
			c.Redirect(302, "/"+page+"/read")
		} else {
			c.Redirect(302, "/"+page+"/edit")
		}
		return
	}

	version := c.DefaultQuery("version", "ajksldfjl")

	// use the default lock
	if defaultLock != "" && p.IsNew() {
		p.IsLocked = true
		p.PassphraseToUnlock = defaultLock
	}

	// Disallow anything but viewing locked/encrypted pages
	if (p.IsEncrypted || p.IsLocked) &&
		(command[0:2] != "/v" && command[0:2] != "/r") {
		fmt.Println("IS LOCKED")
		c.Redirect(302, "/"+page+"/view")
		return
	}

	// Destroy page if it is opened and primed
	if p.IsPrimedForSelfDestruct && !p.IsLocked && !p.IsEncrypted {
		p.Update("*This page has self-destructed. You can not return to it.*\n\n" + p.Text.GetCurrent())
		p.Erase()
		command = "/view"
	}
	if command == "/erase" {
		if !p.IsLocked && !p.IsEncrypted {
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

	if command[0:3] == "/ra" {
		c.Writer.Header().Set("Content-Type", contentType(p.Name))
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Data(200, contentType(p.Name), []byte(rawText))
		return
	}
	log.Debug(command)
	log.Debug("%v", command[0:2] != "/e" &&
		command[0:2] != "/v" &&
		command[0:2] != "/l" &&
		command[0:2] != "/h" &&
		command[0:2] != "/r")

	var FileNames, FileLastEdited []string
	var FileSizes, FileNumChanges []int
	if page == "ls" {
		command = "/view"
		FileNames, FileSizes, FileNumChanges, FileLastEdited = DirectoryList()
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
			command[0:2] != "/h",
		"DirectoryPage":      page == "ls",
		"FileNames":          FileNames,
		"FileSizes":          FileSizes,
		"FileNumChanges":     FileNumChanges,
		"FileLastEdited":     FileLastEdited,
		"Page":               page,
		"RenderedPage":       template.HTML([]byte(rawHTML)),
		"RawPage":            rawText,
		"Versions":           versionsInt64,
		"VersionsText":       versionsText,
		"VersionsChangeSums": versionsChangeSums,
		"IsLocked":           p.IsLocked,
		"IsEncrypted":        p.IsEncrypted,
		"ListItems":          renderList(rawText),
		"Route":              "/" + page + command,
		"HasDotInName":       strings.Contains(page, "."),
		"RecentlyEdited":     getRecentlyEdited(page, c),
		"IsPublished":        p.IsPublished,
		"CustomCSS":          len(customCSS) > 0,
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
	var message string
	success := false
	if p.IsLocked {
		message = "Locked, must unlock first"
	} else if p.IsEncrypted {
		message = "Encrypted, must decrypt first"
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
		success = true
	}
	c.JSON(http.StatusOK, gin.H{"success": success, "message": message})
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
	if p.IsLocked {
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
		p.IsLocked = true
		p.PassphraseToUnlock = defaultLock
	}

	if p.IsEncrypted {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Encrypted"})
		return
	}
	var message string
	if p.IsLocked {
		err2 := CheckPasswordHash(json.Passphrase, p.PassphraseToUnlock)
		if err2 != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Can't unlock"})
			return
		}
		p.IsLocked = false
		message = "Unlocked"
	} else {
		p.IsLocked = true
		p.PassphraseToUnlock = HashPassword(json.Passphrase)
		message = "Locked"
	}
	fmt.Println(p)
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
	if p.IsLocked {
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
	if p.IsLocked {
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
