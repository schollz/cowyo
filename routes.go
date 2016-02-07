package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

func newNote(c *gin.Context) {
	title := randomAlliterateCombo()
	c.Redirect(302, "/"+title)
}

func editNote(c *gin.Context) {
	title := c.Param("title")
	if title == "ws" {
		wshandler(c.Writer, c.Request)
	} else if strings.ToLower(title) == "about" && strings.Contains(AllowedIPs, c.ClientIP()) != true {
		c.Redirect(302, "/about/view")
	} else {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"Title":      title,
			"ExternalIP": ExternalIP,
		})
	}
}

func everythingElse(c *gin.Context) {
	option := c.Param("option")
	title := c.Param("title")
	fmt.Println(title, "["+option+"]")
	if option == "/view" {
		renderMarkdown(c, title)
	} else if option == "/list" {
		renderList(c, title)
	} else if title == "static" {
		serveStaticFile(c, option)
	} else {
		c.Redirect(302, "/"+title)
	}
}

func serveStaticFile(c *gin.Context, option string) {
	staticFile, err := ioutil.ReadFile("./static" + option)
	if err != nil {
		c.AbortWithStatus(404)
	} else {
		c.Data(200, contentType(option), []byte(staticFile))
	}
}

func renderMarkdown(c *gin.Context, title string) {
	p := CowyoData{strings.ToLower(title), ""}
	err := p.load()
	if err != nil {
		panic(err)
	}
	unsafe := blackfriday.MarkdownCommon([]byte(p.Text))
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	c.HTML(http.StatusOK, "view.tmpl", gin.H{
		"Title": title,
		"Body":  template.HTML(html),
	})
}

func renderList(c *gin.Context, title string) {
	p := CowyoData{strings.ToLower(title), ""}
	err := p.load()
	if err != nil {
		panic(err)
	}
	listItems := []string{}
	for _, line := range strings.Split(p.Text, "\n") {
		if len(line) > 1 {
			listItems = append(listItems, line)
		}
	}
	fmt.Println(listItems)
	c.HTML(http.StatusOK, "list.tmpl", gin.H{
		"Title":     title,
		"ListItems": listItems,
	})
}
