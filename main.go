package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

var ExternalIP string
var AllowedIPs string

func init() {
	AllowedIPs = "192.168.1.13,192.168.1.12,192.168.1.2"
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal("You need to specify the external IP address")
	}
	ExternalIP = os.Args[1]
	Open()
	defer Close()

	// Default page
	p := CowyoData{"about", about_page}
	p.save()
	fmt.Println(about_page)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/", newNote)
	r.GET("/:title", editNote)
	r.GET("/:title/*option", everythingElse)
	r.DELETE("/listitem", deleteListItem)
	r.Run(":12312")
}
