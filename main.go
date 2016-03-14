package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
)

// AllowedIPs is a white/black list of
// IP addresses allowed to access awwkoala
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
}
var VersionNum string

func main() {
	VersionNum = "0.93"
	// _, executableFile, _, _ := runtime.Caller(0) // get full path of this file
	cwd, _ := os.Getwd()
	databaseFile := path.Join(cwd, "data.db")
	flag.StringVar(&RuntimeArgs.Port, "p", ":8003", "port to bind")
	flag.StringVar(&RuntimeArgs.DatabaseLocation, "db", databaseFile, "location of database file")
	flag.StringVar(&RuntimeArgs.AdminKey, "a", RandStringBytesMaskImprSrc(50), "key to access admin priveleges")
	flag.StringVar(&RuntimeArgs.ServerCRT, "crt", "", "location of ssl crt")
	flag.StringVar(&RuntimeArgs.ServerKey, "key", "", "location of ssl key")
	flag.StringVar(&RuntimeArgs.WikiName, "w", "AwwKoala", "custom name for wiki")
	dumpDataset := flag.Bool("dump", false, "flag to dump all data to 'dump' directory")
	flag.CommandLine.Usage = func() {
		fmt.Println(`AwwKoala (version ` + VersionNum + `): A Websocket Wiki and Kind Of A List Application
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'awwkoala yourserver.com'
Example: 'awwkoala -p :8080 localhost:8080'
Example: 'awwkoala -db /var/lib/awwkoala/db.bolt localhost:8003'
Example: 'awwkoala -p :8080 -crt ssl/server.crt -key ssl/server.key localhost:8080'
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
	p := WikiData{"help", "", []string{}, []string{}, false}
	p.save(string(aboutFile))

	// var q WikiData
	// q.load("about")
	// fmt.Println(getImportantVersions(q))

	r := gin.Default()
	r.LoadHTMLGlob(path.Join(RuntimeArgs.SourcePath, "templates/*"))
	r.GET("/", newNote)
	r.HEAD("/", func(c *gin.Context) { c.Status(200) })
	r.GET("/:title", editNote)
	r.GET("/:title/*option", everythingElse)
	r.POST("/:title/*option", encryptionRoute)
	r.DELETE("/listitem", deleteListItem)
	r.DELETE("/deletepage", deletePage)
	if RuntimeArgs.ServerCRT != "" && RuntimeArgs.ServerKey != "" {
		RuntimeArgs.Socket = "wss"
		fmt.Println("--------------------------")
		fmt.Println("AwwKoala (version " + VersionNum + ") is up and running on https://" + RuntimeArgs.ExternalIP)
		fmt.Println("Admin key: " + RuntimeArgs.AdminKey)
		fmt.Println("--------------------------")
		r.RunTLS(RuntimeArgs.Port, RuntimeArgs.ServerCRT, RuntimeArgs.ServerKey)
	} else {
		RuntimeArgs.Socket = "ws"
		fmt.Println("--------------------------")
		fmt.Println("AwwKoala (version " + VersionNum + ") is up and running on http://" + RuntimeArgs.ExternalIP)
		fmt.Println("Admin key: " + RuntimeArgs.AdminKey)
		fmt.Println("--------------------------")
		r.Run(RuntimeArgs.Port)
	}
}
