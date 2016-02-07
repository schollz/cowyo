package main

import (
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
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/", newNote)
	r.GET("/:title", editNote)
	r.GET("/:title/*option", everythingElse)
	r.Run(":12312")
}
