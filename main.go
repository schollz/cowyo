package main

import (
	"flag"
	"fmt"
	"log"
	"path"
	"runtime"

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
	ExternalIP       string
	Port             string
	DatabaseLocation string
	ServerCRT        string
	ServerKey        string
	SourcePath       string
	AdminKey         string
}

func main() {
	_, executableFile, _, _ := runtime.Caller(0) // get full path of this file
	databaseFile := path.Join(path.Dir(executableFile), "data.db")
	flag.StringVar(&RuntimeArgs.Port, "p", ":12312", "port to bind")
	flag.StringVar(&RuntimeArgs.DatabaseLocation, "db", databaseFile, "location of database file")
	flag.StringVar(&RuntimeArgs.AdminKey, "a", "", "key to access admin priveleges")
	flag.StringVar(&RuntimeArgs.ServerCRT, "crt", "", "location of ssl crt")
	flag.StringVar(&RuntimeArgs.ServerKey, "key", "", "location of ssl key")
	flag.CommandLine.Usage = func() {
		fmt.Println(`AwwKoala: A Websocket Wiki and Kind Of A List Application
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'awwkoala localhost'
Example: 'awwkoala -p :8080 localhost:8080'
Example: 'awwkoala -db /var/lib/awwkoala/db.bolt localhost:12312'
Example: 'awwkoala -p :8080 -crt ssl/server.crt -key ssl/server.key localhost:8080'
Options:`)
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()
	RuntimeArgs.ExternalIP = flag.Arg(0)
	if RuntimeArgs.ExternalIP == "" {
		log.Fatal("You need to specify the external IP address")
	}
	RuntimeArgs.SourcePath = path.Dir(executableFile)
	Open(RuntimeArgs.DatabaseLocation)
	defer Close()

	// Default page
	p := WikiData{"about", aboutPageText, []string{}, []string{}}
	p.save(aboutPageText)

	// var q WikiData
	// q.load("about")
	// fmt.Println(getImportantVersions(q))

	r := gin.Default()
	r.LoadHTMLGlob(path.Join(RuntimeArgs.SourcePath, "templates/*"))
	r.GET("/", newNote)
	r.GET("/:title", editNote)
	r.GET("/:title/*option", everythingElse)
	r.DELETE("/listitem", deleteListItem)
	if RuntimeArgs.ServerCRT != "" && RuntimeArgs.ServerKey != "" {
		r.RunTLS(RuntimeArgs.Port, RuntimeArgs.ServerCRT, RuntimeArgs.ServerKey)
	} else {
		log.Println("No crt/key found, running non-https")
		r.Run(RuntimeArgs.Port)
	}
}
