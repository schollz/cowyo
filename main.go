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
// IP addresses allowed to access cowyo
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
}

func main() {
	_, executableFile, _, _ := runtime.Caller(0) // get full path of this file
	databaseFile := path.Join(path.Dir(executableFile), "data.db")
	flag.StringVar(&RuntimeArgs.Port, "p", ":12312", "port to bind")
	flag.StringVar(&RuntimeArgs.DatabaseLocation, "db", databaseFile, "location of database file")
	flag.StringVar(&RuntimeArgs.ServerCRT, "crt", "", "location of ssl crt")
	flag.StringVar(&RuntimeArgs.ServerKey, "key", "", "location of ssl key")
	flag.CommandLine.Usage = func() {
		fmt.Println(`cowyo: a websocket notepad
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'cowyo localhost'
Example: 'cowyo -p :8080 localhost'
Example: 'cowyo -db /var/lib/cowyo/db.bolt localhost'
Example: 'cowyo -p :8080 -crt ssl/server.crt -key ssl/server.key localhost'
Options:`)
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()
	RuntimeArgs.ExternalIP = flag.Arg(0)
	if RuntimeArgs.ExternalIP == "" {
		log.Fatal("You need to specify the external IP address")
	}
	Open(RuntimeArgs.DatabaseLocation)
	defer Close()

	// Default page
	p := CowyoData{"about", about_page, []string{}, []string{}}
	p.save(about_page)
	fmt.Println(about_page)

	var q CowyoData
	q.load("SpikySeaSlug")
	rebuildTexts(q)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
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
