package gin_teeny_security

import "net/http/cookiejar"
import "strings"
import "net/http/httptest"
import "net/http"
import "net/url"
import "log"
import "io"
import "fmt"
import "time"
import "io/ioutil"
import "testing"
import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/sessions"

func init() {
	http.DefaultClient.Jar, _ = cookiejar.New(nil)
}

func SampleGinApp() *gin.Engine {
	router := gin.Default()
	store := sessions.NewCookieStore([]byte("tis a secret"))
	router.Use(sessions.Sessions("mysession", store))
	cfg := &Config{
		Path:   "/enter-password/",
		Secret: "garden",
		RequireAuth: func(c *gin.Context) bool {
			return !strings.HasPrefix(c.Request.URL.Path, "/public")
		},
	}
	router.Use(cfg.Middleware)

	router.GET("/private", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/html", []byte("private stuff"))
	})

	router.GET("/public", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/html", []byte("public stuff"))
	})

	return router
}

func TestAuth(t *testing.T) {
	ts := httptest.NewServer(SampleGinApp())

	// Check public stuff can be accessed
	res, err := http.Get(ts.URL + "/public/")
	die(err)
	mustBe("public stuff", readString(res.Body))

	// Check private stuff can't be accessed
	res, err = http.Get(ts.URL + "/private/")
	die(err)

	// Check entering the password as an HTTP header instead of a cookie works
	r, err := http.NewRequest("GET", ts.URL+"/private/", nil)
	die(err)
	r.Header.Set("Authorization", "garden")
	res, err = http.DefaultClient.Do(r)
	die(err)
	mustBe("private stuff", readString(res.Body))

	// Check entering the wrong password as an HTTP header instead of a cookie works
	r, err = http.NewRequest("GET", ts.URL+"/private/", nil)
	die(err)
	r.Header.Set("Authorization", "wrong")
	res, err = http.DefaultClient.Do(r)
	die(err)
	mustStartWith("<h1>Login</h1>\n\n<form action=\"/enter-password/?return=%2Fprivate\"", readString(res.Body))

	res, err = http.Get(ts.URL + "/private/")
	die(err)
	mustStartWith("<h1>Login</h1>\n\n<form action=\"/enter-password/?return=%2Fprivate\"", readString(res.Body))

	// Check entering a bad password gives you a message
	allowedFinishTime := time.Now().Add(time.Second)
	res, err = http.PostForm(ts.URL+"/enter-password/", url.Values{"secretAccessCode": []string{"wrong"}})
	die(err)
	mustStartWith("<h1>Login</h1>\n<h2>Wrong Password</h2>", readString(res.Body))
	finishTime := time.Now()
	if finishTime.Before(allowedFinishTime) {
		die(fmt.Errorf("Expected failed login to take at least 1 second"))
	}

	// Check entering a good password lets you access things
	res, err = http.PostForm(ts.URL+"/enter-password/?return=/private/", url.Values{"secretAccessCode": []string{"garden"}})
	die(err)
	mustBe("private stuff", readString(res.Body))
}

func mustStartWith(expected, actual string) {
	if !strings.HasPrefix(strings.TrimSpace(actual), expected) {
		log.Panicf("Should have gotten content starting with '%s' but got '%s'", expected, actual)
	}
}

func mustBe(expected, actual string) {
	if actual != expected {
		log.Panicf("Should have gotten '%s' but got '%s'", expected, actual)
	}
}

func readString(r io.ReadCloser) string {
	b, e := ioutil.ReadAll(r)
	defer r.Close()
	die(e)
	return string(b)
}

func die(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
