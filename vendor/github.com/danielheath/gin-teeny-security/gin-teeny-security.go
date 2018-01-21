// A GIN middleware providing low-fi security for personal stuff.
package gin_teeny_security

import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/sessions"
import "net/http"

// Forces you to a login page until you provide a secret code.
// No CSRF protection, so any script on any page can log you
// out (or in, if they know the password).
// The rest of your site needs XSS protection on forms or any site on the
// net can inject stuff. If you're sending open CORS headers this
// would be particularly bad.
func RequiresSecretAccessCode(secretAccessCode, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		if c.Request.URL.Path == path {
			if c.Request.Method == "POST" {
				c.Request.ParseForm()

				if c.Request.PostForm.Get("secretAccessCode") == secretAccessCode {
					c.Header("Location", "/")
					session.Set("secretAccessCode", secretAccessCode)
					session.Save()
					c.AbortWithStatus(http.StatusFound)
					return
				} else {
					session.Set("secretAccessCode", "")
					session.Save()
					c.Data(http.StatusForbidden, "text/html", []byte(`
            <h1>Login</h1>
            <h2>Wrong password</h2>
            <form action="`+path+`" method="POST">
              <input name="secretAccessCode" />
              <input type="submit" value="Login" />
            </form>
          `))
					c.Abort()
					return
				}
			} else if c.Request.Method == "GET" {
				c.Data(http.StatusOK, "text/html", []byte(`
          <h1>Login</h1>
          <form action="`+path+`" method="POST">
          <input name="secretAccessCode" />
          <input type="submit" value="Login" />
          </form>
        `))
				c.Abort()
				return
			} else {
				c.Next()
				return
			}
		}

		v := session.Get("secretAccessCode")
		if v != secretAccessCode {
			c.Header("Location", path)
			c.AbortWithStatus(http.StatusTemporaryRedirect)
		} else {
			c.Next()
		}
	}
}
