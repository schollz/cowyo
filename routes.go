package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
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
	} else if strings.ToLower(title) == "about" { //}&& strings.Contains(AllowedIPs, c.ClientIP()) != true {
		c.Redirect(302, "/about/view")
	} else {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"Title":      title,
			"ExternalIP": RuntimeArgs.ExternalIP,
		})
	}
}

func everythingElse(c *gin.Context) {
	option := c.Param("option")
	title := c.Param("title")
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
	var p CowyoData
	err := p.load(strings.ToLower(title))
	if err != nil {
		panic(err)
	}
	unsafe := blackfriday.MarkdownCommon([]byte(p.CurrentText))
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	html2 := string(html)
	r, _ := regexp.Compile("\\$\\$(.*?)\\$\\$")
	for _, s := range r.FindAllString(html2, -1) {
		html2 = strings.Replace(html2, s, "<span class='texp' data-expr='"+s[2:len(s)-2]+"'></span>", 1)
	}
	r, _ = regexp.Compile("\\$(.*?)\\$")
	for _, s := range r.FindAllString(html2, -1) {
		html2 = strings.Replace(html2, s, "<span class='texi' data-expr='"+s[1:len(s)-1]+"'></span>", 1)
	}

	html2 = strings.Replace(html2, "&amp;#36;", "&#36;", -1)
	c.HTML(http.StatusOK, "view.tmpl", gin.H{
		"Title": title,
		"Body":  template.HTML([]byte(html2)),
	})
}

func reorderList(text string) ([]template.HTML, []string) {
	listItemsString := ""
	for _, lineString := range strings.Split(text, "\n") {
		if len(lineString) > 1 {
			if string(lineString[0]) != "-" {
				listItemsString += "- " + lineString + "\n"
			} else {
				listItemsString += lineString + "\n"
			}
		}
	}

	// get ordering of template.HTML for rendering
	renderedListString := string(blackfriday.MarkdownCommon([]byte(listItemsString)))
	listItems := []template.HTML{}
	endItems := []template.HTML{}
	for _, lineString := range strings.Split(renderedListString, "\n") {
		if len(lineString) > 1 {
			if strings.Contains(lineString, "<del>") || strings.Contains(lineString, "</ul>") {
				endItems = append(endItems, template.HTML(lineString))
			} else {
				listItems = append(listItems, template.HTML(lineString))
			}
		}
	}

	// get ordering of strings for deleting
	listItemsStringArray := []string{}
	endItemsStringArray := []string{}
	for _, lineString := range strings.Split(listItemsString, "\n") {
		if len(lineString) > 1 {
			if strings.Contains(lineString, "~~") {
				endItemsStringArray = append(endItemsStringArray, lineString)
			} else {
				listItemsStringArray = append(listItemsStringArray, lineString)
			}
		}
	}
	return append(listItems, endItems...), append(listItemsStringArray, endItemsStringArray...)
}

func renderList(c *gin.Context, title string) {
	if strings.ToLower(title) == "about" { //}&& strings.Contains(AllowedIPs, c.ClientIP()) != true {
		c.Redirect(302, "/about/view")
	}
	var p CowyoData
	err := p.load(strings.ToLower(title))
	if err != nil {
		panic(err)
	}

	listItems, _ := reorderList(p.CurrentText)

	c.HTML(http.StatusOK, "list.tmpl", gin.H{
		"Title":     title,
		"ListItems": listItems,
	})
}

func deleteListItem(c *gin.Context) {
	lineNum, err := strconv.Atoi(c.DefaultQuery("lineNum", "None"))
	title := c.Query("title") // shortcut for c.Request.URL.Query().Get("lastname")
	if err == nil {
		var p CowyoData
		err := p.load(strings.ToLower(title))
		if err != nil {
			panic(err)
		}

		_, listItems := reorderList(p.CurrentText)
		newText := p.CurrentText
		for i, lineString := range listItems {
			// fmt.Println(i, lineString, lineNum)
			if i+1 == lineNum {
				// fmt.Println("MATCHED")
				if strings.Contains(lineString, "~~") == false {
					// fmt.Println(p.Text, "("+lineString[2:]+"\n"+")", "~~"+lineString[2:]+"~~"+"\n")
					newText = strings.Replace(newText+"\n", lineString[2:]+"\n", "~~"+lineString[2:]+"~~"+"\n", 1)
				} else {
					newText = strings.Replace(newText+"\n", lineString[2:]+"\n", lineString[4:len(lineString)-2]+"\n", 1)
				}
				p.save(newText)
				break
			}
		}

		c.JSON(200, gin.H{
			"message": "Done.",
		})
	} else {
		c.JSON(404, gin.H{
			"message": "?",
		})
	}

}
