package main

import (
	"github.com/schollz/cowyo/config"
)

func main() {
	c, err := config.ParseFile("multisite_sample.toml")
	if err != nil {
		panic(err)
	}

	panic(c.ListenAndServe())
}
