package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
)

// AllowedIPs is a white/black list of
// IP addresses allowed to access cowyo
var AllowedIPs = map[string]bool{
	"192.168.1.13": true,
	"192.168.1.12": true,
	"192.168.1.2":  true,
}

// RuntimeArgs contains all runtime
// arguments available
var RuntimeArgs struct {
	WikiName         string
	ExternalIP       string
	Port             string
	DatabaseLocation string
	ServerCRT        string
	ServerKey        string
	SourcePath       string
	AdminKey         string
	Socket           string
	ForceWss         bool
}
var VersionNum string

const _24K = (1 << 20) * 24

func main() {
	VersionNum = "0.94"
	// _, executableFile, _, _ := runtime.Caller(0) // get full path of this file
	cwd, _ := os.Getwd()
	databaseFile := path.Join(cwd, "data.db")
	flag.StringVar(&RuntimeArgs.Port, "p", ":8003", "port to bind")
	flag.StringVar(&RuntimeArgs.DatabaseLocation, "db", databaseFile, "location of database file")
	flag.StringVar(&RuntimeArgs.AdminKey, "a", RandStringBytesMaskImprSrc(50), "key to access admin priveleges")
	flag.StringVar(&RuntimeArgs.ServerCRT, "crt", "", "location of ssl crt")
	flag.StringVar(&RuntimeArgs.ServerKey, "key", "", "location of ssl key")
	flag.StringVar(&RuntimeArgs.WikiName, "w", "cowyo", "custom name for wiki")
	flag.BoolVar(&RuntimeArgs.ForceWss, "e", false, "force encrypted sockets")
	dumpDataset := flag.Bool("dump", false, "flag to dump all data to 'dump' directory")
	flag.CommandLine.Usage = func() {
		fmt.Println(`cowyo (version ` + VersionNum + `): A Websocket Wiki and Kind Of A List Application
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'cowyo yourserver.com'
Example: 'cowyo -p :8080 localhost:8080'
Example: 'cowyo -db /var/lib/cowyo/db.bolt localhost:8003'
Example: 'cowyo -p :8080 -crt ssl/server.crt -key ssl/server.key localhost:8080'
Options:`)
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()

	if *dumpDataset {
		fmt.Println("Dumping data to 'dump' folder...")
		dumpEverything()
		os.Exit(1)
	}

	RuntimeArgs.ExternalIP = flag.Arg(0)
	if RuntimeArgs.ExternalIP == "" {
		RuntimeArgs.ExternalIP = GetLocalIP() + RuntimeArgs.Port
	}
	RuntimeArgs.SourcePath = cwd

	// create programdata bucket
	Open(RuntimeArgs.DatabaseLocation)

	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("programdata"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return err
	})
	if err != nil {
		panic(err)
	}
	Close()

	// Default page
	aboutFile, _ := ioutil.ReadFile(path.Join(RuntimeArgs.SourcePath, "templates/aboutpage.md"))
	p := WikiData{"help", "", []string{}, []string{}, false, "zzz"}
	p.save(string(aboutFile))

	// var q WikiData
	// q.load("about")
	// fmt.Println(getImportantVersions(q))

	r := gin.Default()
	r.LoadHTMLGlob(path.Join(RuntimeArgs.SourcePath, "templates/*"))
	r.GET("/", newNote)
	r.HEAD("/", func(c *gin.Context) { c.Status(200) })
	r.GET("/:title", editNote)
	r.PUT("/:title", func(c *gin.Context) {
		filename := c.Param("title")
		fmt.Println(filename)
		fmt.Println(c.Request.Body)
		fmt.Println(c.Request.ContentLength)
		fmt.Println(c.Request.Header)
		contentLength := c.Request.ContentLength
		var reader io.Reader
		reader = c.Request.Body
		if contentLength == -1 {
			// queue file to disk, because s3 needs content length
			var err error
			var f io.Reader

			f = reader

			var b bytes.Buffer

			n, err := io.CopyN(&b, f, _24K+1)
			if err != nil && err != io.EOF {
				log.Printf("%s", err.Error())
			}

			if n > _24K {
				file, err := ioutil.TempFile("./", "transfer-")
				if err != nil {
					log.Printf("%s", err.Error())
				}

				defer file.Close()

				n, err = io.Copy(file, io.MultiReader(&b, f))
				if err != nil {
					os.Remove(file.Name())
					log.Printf("%s", err.Error())
				}

				reader, err = os.Open(file.Name())
			} else {
				reader = bytes.NewReader(b.Bytes())
			}

			contentLength = n
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		fmt.Println("---------------")
		fmt.Println(buf.String())
		fmt.Println("---------------")
		fmt.Println(c.ContentType())
		fmt.Println(c.Request.Header)
		fmt.Println("---------------")
		p := WikiData{filename, "", []string{}, []string{}, false, ""}
		p.save(buf.String())
	})
	r.GET("/:title/*option", everythingElse)
	r.POST("/:title/*option", encryptionRoute)
	r.DELETE("/listitem", deleteListItem)
	r.DELETE("/deletepage", deletePage)
	if RuntimeArgs.ServerCRT != "" && RuntimeArgs.ServerKey != "" {
		RuntimeArgs.Socket = "wss"
		fmt.Println("--------------------------")
		fmt.Println("cowyo (version " + VersionNum + ") is up and running on https://" + RuntimeArgs.ExternalIP)
		fmt.Println("Admin key: " + RuntimeArgs.AdminKey)
		fmt.Println("--------------------------")
		r.RunTLS(RuntimeArgs.Port, RuntimeArgs.ServerCRT, RuntimeArgs.ServerKey)
	} else {
		RuntimeArgs.Socket = "ws"
		if RuntimeArgs.ForceWss {
			RuntimeArgs.Socket = "wss"
		}
		fmt.Println("--------------------------")
		fmt.Println("cowyo (version " + VersionNum + ") is up and running on http://" + RuntimeArgs.ExternalIP)
		fmt.Println("Admin key: " + RuntimeArgs.AdminKey)
		fmt.Println("--------------------------")
		r.Run(RuntimeArgs.Port)
	}
}
