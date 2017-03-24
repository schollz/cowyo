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
		fmt.Printf("\nRunning cowyo server (version %s) at http://%s:%s\n\n", version, GetLocalIP(), c.GlobalString("port"))
		serve(c.GlobalString("port"))
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
			Name:  "port,p",
			Value: "8050",
			Usage: "port to use",
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
