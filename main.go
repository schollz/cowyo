package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

var db *bolt.DB
var open bool
var ExternalIP string
var AllowedIPs string

func init() {
	AllowedIPs = "192.168.1.13,192.168.1.12"
}

func Open() error {
	var err error
	_, filename, _, _ := runtime.Caller(0) // get full path of this file
	dbfile := path.Join(path.Dir(filename), "data.db")
	config := &bolt.Options{Timeout: 30 * time.Second}
	db, err = bolt.Open(dbfile, 0600, config)
	if err != nil {
		fmt.Println("Opening BoltDB timed out")
		log.Fatal(err)
	}
	open = true
	return nil
}

func Close() {
	open = false
	db.Close()
}

type CowyoData struct {
	Title string
	Text  string
}

// Database functions

func (p *CowyoData) load() error {
	if !open {
		return fmt.Errorf("db must be opened before saving!")
	}
	err := db.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("datas"))
		if b == nil {
			return nil
		}
		k := []byte(p.Title)
		val := b.Get(k)
		if val == nil {
			return nil
		}
		err = p.decode(val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Could not get CowyoData: %s", err)
		return err
	}
	return nil
}

func (p *CowyoData) save() error {
	if !open {
		return fmt.Errorf("db must be opened before saving!")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("datas"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		enc, err := p.encode()
		if err != nil {
			return fmt.Errorf("could not encode CowyoData: %s", err)
		}
		err = bucket.Put([]byte(p.Title), enc)
		return err
	})
	return err
}

func (p *CowyoData) encode() ([]byte, error) {
	enc, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func (p *CowyoData) decode(data []byte) error {
	err := json.Unmarshal(data, &p)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal("You need to specify the external IP address")
	}
	ExternalIP = os.Args[1]
	Open()
	defer Close()
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		title := randomAlliterateCombo()
		c.Redirect(302, "/"+title)

	})
	r.GET("/:title", func(c *gin.Context) {
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
	})
	r.GET("/:title/*option", func(c *gin.Context) {
		option := c.Param("option")
		title := c.Param("title")
		fmt.Println(title, "["+option+"]")
		if option == "/view" {
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

		} else {
			c.Redirect(302, "/"+title)
		}
	})

	r.Run(":12312")
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		type Message struct {
			TextData     string
			Title        string
			UpdateServer bool
			UpdateClient bool
		}

		var m Message
		err = json.Unmarshal(msg, &m)
		if err != nil {
			panic(err)
		}

		if m.UpdateServer {
			p := CowyoData{strings.ToLower(m.Title), m.TextData}
			err := p.save()
			if err != nil {
				panic(err)
			}
		}
		if m.UpdateClient {
			p := CowyoData{strings.ToLower(m.Title), ""}
			err := p.load()
			if err != nil {
				panic(err)
			}
			m.TextData = p.Text
		}
		newMsg, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		conn.WriteMessage(t, newMsg)
	}
}
