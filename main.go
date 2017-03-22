package main

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/"+randomAlliterateCombo())
	})
	router.GET("/:page", func(c *gin.Context) {
		page := c.Param("page")
		c.Redirect(302, "/"+page+"/edit")
	})
	router.GET("/:page/:command", handlePageRequest)
	router.POST("/update", handlePageUpdate)
	router.POST("/prime", handlePrime)
	router.POST("/lock", handleLock)

	router.Run(":8050")
}

func handlePageRequest(c *gin.Context) {
	page := c.Param("page")
	command := c.Param("command")
	version := c.DefaultQuery("version", "ajksldfjl")
	p := Open(page)
	if p.IsPrimedForSelfDestruct && !p.IsLocked {
		p.Update("*This page has now self-destructed.*\n\n" + p.Text.GetCurrent())
		p.Erase()
	}
	if command == "erase" && !p.IsLocked {
		p.Erase()
		c.Redirect(302, "/"+page+"/edit")
	}
	rawText := p.Text.GetCurrent()
	rawHTML := p.RenderedPage

	// Check to see if an old version is requested
	versionInt, versionErr := strconv.Atoi(version)
	if versionErr == nil && versionInt > 0 {
		versionText, err := p.Text.GetPreviousByTimestamp(int64(versionInt))
		if err == nil {
			rawText = versionText
			rawHTML = MarkdownToHtml(rawText)
		}
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"EditPage":     command == "edit",
		"ViewPage":     command == "view",
		"ListPage":     command == "list",
		"HistoryPage":  command == "history",
		"Page":         p.Name,
		"RenderedPage": template.HTML([]byte(rawHTML)),
		"RawPage":      rawText,
		"Versions":     p.Text.GetSnapshots(),
		"IsLocked":     p.IsLocked,
		"IsEncrypted":  p.IsEncrypted,
	})
}

func handlePageUpdate(c *gin.Context) {
	type QueryJSON struct {
		Page    string `json:"page"`
		NewText string `json:"new_text"`
	}
	var json QueryJSON
	if c.BindJSON(&json) != nil {
		c.String(http.StatusBadRequest, "Problem binding keys")
		return
	}
	log.Trace("Update: %v", json)
	p := Open(json.Page)
	if !p.IsLocked {
		p.Update(json.NewText)
		p.Save()
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Saved"})
	} else {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Locked"})
	}
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
	p.IsPrimedForSelfDestruct = true
	p.Save()
	c.JSON(http.StatusOK, gin.H{"success": true})
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
	p.Save()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}
