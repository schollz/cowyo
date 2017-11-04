package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/urfave/cli.v1"
)

var version string
var pathToData string

func main() {
	app := cli.NewApp()
	app.Name = "cowyo"
	app.Usage = "a simple wiki"
	app.Version = version
	app.Compiled = time.Now()
	app.Action = func(c *cli.Context) error {
		if !c.GlobalBool("debug") {
			turnOffDebugger()
		}
		pathToData = c.GlobalString("data")
		os.MkdirAll(pathToData, 0755)
		host := c.GlobalString("host")
		crt_f := c.GlobalString("cert") // crt flag
		key_f := c.GlobalString("key")  // key flag
		if host == "" {
			host = GetLocalIP()
		}
		TLS := false
		if crt_f != "" && key_f != "" {
			TLS = true
		}
		if TLS {
			fmt.Printf("\nRunning cowyo server (version %s) at https://%s:%s\n\n", version, host, c.GlobalString("port"))
		} else {
			fmt.Printf("\nRunning cowyo server (version %s) at http://%s:%s\n\n", version, host, c.GlobalString("port"))
		}
		serve(c.GlobalString("host"), c.GlobalString("port"), c.GlobalString("cert"), c.GlobalString("key"), TLS, c.GlobalString("css"), c.GlobalString("default-page"))
		return nil
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "data",
			Value: "data",
			Usage: "data folder to use",
		},
		cli.StringFlag{
			Name:  "olddata",
			Value: "",
			Usage: "data folder for migrating",
		},
		cli.StringFlag{
			Name:  "host",
			Value: "",
			Usage: "host to use",
		},
		cli.StringFlag{
			Name:  "port,p",
			Value: "8050",
			Usage: "port to use",
		},
		cli.StringFlag{
			Name:  "cert",
			Value: "",
			Usage: "absolute path to SSL public sertificate",
		},
		cli.StringFlag{
			Name:  "key",
			Value: "",
			Usage: "absolute path to SSL private key",
		},
		cli.StringFlag{
			Name:  "css",
			Value: "",
			Usage: "use a custom CSS file",
		},
		cli.StringFlag{
			Name:  "default-page",
			Value: "",
			Usage: "show default-page/read instead of editing",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "turn on debugging",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "migrate",
			Aliases: []string{"m"},
			Usage:   "migrate from the old cowyo",
			Action: func(c *cli.Context) error {
				if !c.GlobalBool("debug") {
					turnOffDebugger()
				}
				pathToData = c.GlobalString("data")
				pathToOldData := c.GlobalString("olddata")
				if len(pathToOldData) == 0 {
					fmt.Printf("You need to specify folder with -olddata")
					return nil
				}
				os.MkdirAll(pathToData, 0755)
				if !exists(pathToOldData) {
					fmt.Printf("Can not find '%s', does it exist?", pathToOldData)
					return nil
				}
				migrate(pathToOldData, pathToData)
				return nil
			},
		},
	}

	app.Run(os.Args)

}
