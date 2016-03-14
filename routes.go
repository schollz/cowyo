package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type EncryptionPost struct {
	Text     string `form:"text" json:"text" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func encryptionRoute(c *gin.Context) {
	title := c.Param("title")
	option := c.Param("option")
	fmt.Println(option, title)
	var jsonLoad EncryptionPost
	if option == "/decrypt" {
		if c.BindJSON(&jsonLoad) == nil {
			var err error
			currentText, _, _, _, encrypted := getCurrentText(title, -1)
			if encrypted == true {
				currentText, err = decryptString(currentText, jsonLoad.Password)
				if err != nil {
					c.JSON(200, gin.H{
						"status":  "Inorrect passphrase.",
						"title":   title,
						"option":  option,
						"success": false,
					})
				} else {
					p := WikiData{strings.ToLower(title), "", []string{}, []string{}, false}
					p.save(currentText)
					c.JSON(200, gin.H{
						"status":  "posted",
						"title":   title,
						"option":  option,
						"success": true,
					})
				}
			}
		} else {
			c.JSON(200, gin.H{
				"status":  "Could not bind",
				"title":   title,
				"option":  option,
				"success": false,
			})
		}
	}
	if option == "/encrypt" {
		if c.BindJSON(&jsonLoad) == nil {
			p := WikiData{strings.ToLower(title), "", []string{}, []string{}, true}
			p.save(encryptString(jsonLoad.Text, jsonLoad.Password))
			c.JSON(200, gin.H{
				"status":  "posted",
				"title":   title,
				"option":  option,
				"success": true,
			})
		} else {
			c.JSON(200, gin.H{
				"status":  "posted",
				"title":   title,
				"option":  option,
				"success": false,
			})
		}
	}

}

func newNote(c *gin.Context) {
	title := randomAlliterateCombo()
	c.Redirect(302, "/"+title)
}

func editNote(c *gin.Context) {
	title := c.Param("title")
	if title == "ws" {
		wshandler(c.Writer, c.Request)
	} else if title == "robots.txt" {
		c.Data(200, "text/plain", []byte(robotsTxt))
	} else if strings.ToLower(title) == "help" { //}&& strings.Contains(AllowedIPs, c.ClientIP()) != true {
		c.Redirect(302, "/Help/view")
	} else {
		locked, _ := hasPassword(title)
		if locked {
			c.Redirect(302, "/"+title+"/view")
		} else {
			version := c.DefaultQuery("version", "-1")
			versionNum, _ := strconv.Atoi(version)
			currentText, versions, currentVersion, totalTime, encrypted := getCurrentText(title, versionNum)
			if encrypted {
				c.Redirect(302, "/"+title+"/view")
			}
			if strings.Contains(currentText, "self-destruct\n") || strings.Contains(currentText, "\nself-destruct") {
				c.Redirect(302, "/"+title+"/view")
			}
			numRows := len(strings.Split(currentText, "\n")) + 10
			totalTimeString := totalTime.String()
			if totalTime.Seconds() < 1 {
				totalTimeString = "< 1 s"
			}
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"Title":       title,
				"WikiName":    RuntimeArgs.WikiName,
				"ExternalIP":  RuntimeArgs.ExternalIP,
				"CurrentText": currentText,
				"NumRows":     numRows,
				"Versions":    versions,
				"TotalTime":   totalTimeString,
				"SocketType":  RuntimeArgs.Socket,
				"NoEdit":      !currentVersion,
			})

		}
	}
}

func everythingElse(c *gin.Context) {
	option := c.Param("option")
	title := c.Param("title")
	if option == "/view" {
		version := c.DefaultQuery("version", "-1")
		noprompt := c.DefaultQuery("noprompt", "-1")
		versionNum, _ := strconv.Atoi(version)
		if strings.ToLower(title) == "help" {
			versionNum = -1
		}
		currentText, versions, _, totalTime, encrypted := getCurrentText(title, versionNum)
		if (strings.Contains(currentText, "self-destruct\n") || strings.Contains(currentText, "\nself-destruct")) && strings.ToLower(title) != "help" {
			currentText = strings.Replace(currentText, "self-destruct\n", `> *This page has been deleted, you cannot return after closing.*`+"\n", 1)
			currentText = strings.Replace(currentText, "\nself-destruct", "\n"+`> *This page has been deleted, you cannot return after closing.*`, 1)
			p := WikiData{strings.ToLower(title), "", []string{}, []string{}, false}
			p.save("")
		}
		renderMarkdown(c, currentText, title, versions, "", totalTime, encrypted, noprompt == "-1")
	} else if title == "ls" && option == "/"+RuntimeArgs.AdminKey && len(RuntimeArgs.AdminKey) > 1 {
		renderMarkdown(c, listEverything(), "ls", nil, RuntimeArgs.AdminKey, time.Now().Sub(time.Now()), false, false)
	} else if option == "/list" {
		renderList(c, title)
	} else if title == "static" {
		serveStaticFile(c, option)
	} else {
		c.Redirect(302, "/"+title)
	}
}

func serveStaticFile(c *gin.Context, option string) {
	staticFile, err := ioutil.ReadFile(path.Join(RuntimeArgs.SourcePath, "static") + option)
	if err != nil {
		c.AbortWithStatus(404)
	} else {
		c.Data(200, contentType(option), []byte(staticFile))
	}
}

func renderMarkdown(c *gin.Context, currentText string, title string, versions []versionsInfo, AdminKey string, totalTime time.Duration, encrypted bool, noprompt bool) {
	r, _ := regexp.Compile("\\[\\[(.*?)\\]\\]")
	for _, s := range r.FindAllString(currentText, -1) {
		currentText = strings.Replace(currentText, s, "["+s[2:len(s)-2]+"](/"+s[2:len(s)-2]+"/view)", 1)
	}
	unsafe := blackfriday.MarkdownCommon([]byte(currentText))
	pClean := bluemonday.UGCPolicy()
	pClean.AllowElements("img")
	pClean.AllowAttrs("alt").OnElements("img")
	pClean.AllowAttrs("src").OnElements("img")
	pClean.AllowAttrs("class").OnElements("a")
	pClean.AllowAttrs("href").OnElements("a")
	pClean.AllowAttrs("id").OnElements("a")
	pClean.AllowDataURIImages()
	html := pClean.SanitizeBytes(unsafe)
	html2 := string(html)
	r, _ = regexp.Compile("\\$\\$(.*?)\\$\\$")
	for _, s := range r.FindAllString(html2, -1) {
		html2 = strings.Replace(html2, s, "<span class='texp' data-expr='"+s[2:len(s)-2]+"'></span>", 1)
	}
	r, _ = regexp.Compile("\\$(.*?)\\$")
	for _, s := range r.FindAllString(html2, -1) {
		html2 = strings.Replace(html2, s, "<span class='texi' data-expr='"+s[1:len(s)-1]+"'></span>", 1)
	}

	html2 = strings.Replace(html2, "&amp;#36;", "&#36;", -1)
	html2 = strings.Replace(html2, "&amp;#91;", "&#91;", -1)
	html2 = strings.Replace(html2, "&amp;#93;", "&#93;", -1)
	html2 = strings.Replace(html2, "&amp35;", "&#35;", -1)
	totalTimeString := totalTime.String()
	if totalTime.Seconds() < 1 {
		totalTimeString = "< 1 s"
	}
	c.HTML(http.StatusOK, "view.tmpl", gin.H{
		"Title":     title,
		"WikiName":  RuntimeArgs.WikiName,
		"Body":      template.HTML([]byte(html2)),
		"Versions":  versions,
		"TotalTime": totalTimeString,
		"AdminKey":  AdminKey,
		"Encrypted": encrypted,
		"Prompt":    noprompt,
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
	if strings.ToLower(title) == "help" { //}&& strings.Contains(AllowedIPs, c.ClientIP()) != true {
		c.Redirect(302, "/Help/view")
	}
	var p WikiData
	err := p.load(strings.ToLower(title))
	if err != nil {
		panic(err)
	}

	currentText := p.CurrentText
	if strings.Contains(currentText, "self-destruct\n") || strings.Contains(currentText, "\nself-destruct") {
		c.Redirect(302, "/"+title+"/view")
	}
	if p.Encrypted {
		c.Redirect(302, "/"+title+"/view")
	}

	pClean := bluemonday.UGCPolicy()
	pClean.AllowElements("img")
	pClean.AllowAttrs("alt").OnElements("img")
	pClean.AllowAttrs("src").OnElements("img")
	pClean.AllowAttrs("class").OnElements("a")
	pClean.AllowAttrs("href").OnElements("a")
	pClean.AllowAttrs("id").OnElements("a")
	pClean.AllowDataURIImages()
	text := pClean.SanitizeBytes([]byte(p.CurrentText))
	listItems, _ := reorderList(string(text))
	fmt.Println(string(text))
	c.HTML(http.StatusOK, "list.tmpl", gin.H{
		"Title":     title,
		"WikiName":  RuntimeArgs.WikiName,
		"ListItems": listItems,
	})
}

func deleteListItem(c *gin.Context) {
	lineNum, err := strconv.Atoi(c.DefaultQuery("lineNum", "None"))
	title := c.Query("title") // shortcut for c.Request.URL.Query().Get("lastname")
	if err == nil {
		var p WikiData
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
					newText = strings.Replace(newText+"\n", lineString[2:]+"\n", "~~"+strings.TrimSpace(lineString[2:])+"~~"+"\n", 1)
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

func deletePage(c *gin.Context) {
	deleteName := c.DefaultQuery("DeleteName", "None")
	adminKey := c.DefaultQuery("AdminKey", "None")
	fmt.Println(adminKey)
	fmt.Println(deleteName)
	// if adminKey == RuntimeArgs.AdminKey || true == true {
	if strings.ToLower(deleteName) != "help" {
		p := WikiData{strings.ToLower(deleteName), "", []string{}, []string{}, false}
		p.save("")
	}
	// // remove from program data
	// var deleteKey []byte
	// foundKey := false
	// err := db.View(func(tx *bolt.Tx) error {
	// 	b := tx.Bucket([]byte("programdata"))
	// 	c := b.Cursor()
	// 	for k, v := c.First(); k != nil; k, v = c.Next() {
	// 		if strings.ToLower(string(v)) == strings.ToLower(deleteName) {
	// 			fmt.Println("FOUND " + string(v))
	// 			deleteKey = k
	// 			foundKey = true
	// 			break
	// 		}
	// 	}
	// 	return nil
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// if foundKey == true {
	// 	fmt.Println(len([]string{}))
	// 	fmt.Println(deleteKey)
	// 	db.View(func(tx *bolt.Tx) error {
	// 		b := tx.Bucket([]byte("programdata"))
	// 		err := b.Delete(deleteKey)
	// 		return err
	// 	})
	// }

	// return OKAY
	c.JSON(200, gin.H{
		"message": "Done.",
	})
	// } else {
	// 	c.JSON(404, gin.H{
	// 		"message": "?",
	// 	})
	// }
}

func listEverything() string {
	Open(RuntimeArgs.DatabaseLocation)
	defer Close()
	everything := `| Title | Current size    | Changes  | Total Size |  |
| --------- |-------------| -----| ------------- | ------------- |
`
	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte("datas"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var p WikiData
			p.load(string(k))
			if len(p.CurrentText) > 1 {
				contentSize := strconv.Itoa(len(p.CurrentText))
				numChanges := strconv.Itoa(len(p.Diffs))
				totalSize := strconv.Itoa(len(v))
				everything += "| [" + p.Title + "](/" + p.Title + "/view) | " + contentSize + " | " + numChanges + " | " + totalSize + ` | <a class="deleteable" id="` + p.Title + `">Delete</a> | ` + "\n"
			}
		}
		return nil
	})
	return everything
}

func dumpEverything() {
	Open(RuntimeArgs.DatabaseLocation)
	defer Close()
	err := os.MkdirAll("dump", 0777)
	if err != nil {
		fmt.Println("Already exists")
	}
	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte("datas"))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			var p WikiData
			p.load(string(k))
			fmt.Println(string(k), len(p.CurrentText))
			ioutil.WriteFile(path.Join("dump", string(k)+".md"), []byte(p.CurrentText), 0644)
		}
		return nil
	})
}
