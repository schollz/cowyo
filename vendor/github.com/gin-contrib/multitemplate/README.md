# Multitemplate

[![Build Status](https://travis-ci.org/gin-contrib/multitemplate.svg)](https://travis-ci.org/gin-contrib/multitemplate)
[![codecov](https://codecov.io/gh/gin-contrib/multitemplate/branch/master/graph/badge.svg)](https://codecov.io/gh/gin-contrib/multitemplate)
[![Go Report Card](https://goreportcard.com/badge/github.com/gin-contrib/multitemplate)](https://goreportcard.com/report/github.com/gin-contrib/multitemplate)
[![GoDoc](https://godoc.org/github.com/gin-contrib/multitemplate?status.svg)](https://godoc.org/github.com/gin-contrib/multitemplate)

This is a custom HTML render to support multi templates, ie. more than one `*template.Template`.

## Usage

### Start using it

Download and install it:

```sh
$ go get github.com/gin-contrib/multitemplate
```

Import it in your code:

```go
import "github.com/gin-contrib/multitemplate"
```

### Simple example

See [example/simple/example.go](example/simple/example.go)

[embedmd]:# (example/simple/example.go go)
```go
package main

import (
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

func createMyRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	r.AddFromFiles("index", "templates/base.html", "templates/index.html")
	r.AddFromFiles("article", "templates/base.html", "templates/index.html", "templates/article.html")
	return r
}

func main() {
	router := gin.Default()
	router.HTMLRender = createMyRender()
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index", gin.H{
			"title": "Html5 Template Engine",
		})
	})
	router.GET("/article", func(c *gin.Context) {
		c.HTML(200, "article", gin.H{
			"title": "Html5 Article Engine",
		})
	})
	router.Run(":8080")
}
```

### Advanced example

[Approximating html/template Inheritance](https://elithrar.github.io/article/approximating-html-template-inheritance/)

See [example/advanced/example.go](example/advanced/example.go)

[embedmd]:# (example/advanced/example.go go)
```go
package main

import (
	"path/filepath"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.HTMLRender = loadTemplates("./templates")
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "Welcome!",
		})
	})
	router.GET("/article", func(c *gin.Context) {
		c.HTML(200, "article.html", gin.H{
			"title": "Html5 Article Engine",
		})
	})

	router.Run(":8080")
}

func loadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	layouts, err := filepath.Glob(templatesDir + "/layouts/*.html")
	if err != nil {
		panic(err.Error())
	}

	includes, err := filepath.Glob(templatesDir + "/includes/*.html")
	if err != nil {
		panic(err.Error())
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, include := range includes {
		files := append(layouts, include)
		r.AddFromFiles(filepath.Base(include), files...)
	}
	return r
}
```
