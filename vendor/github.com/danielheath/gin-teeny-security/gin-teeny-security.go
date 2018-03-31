// A GIN middleware providing low-fi security for sites with simple needs.
//
// Redirects users to a login page until they provide a secret code.
// No CSRF protection, so any js on the web can log you
// out (or in, if they know the password).
//
// Protects you from brute-force attacks by making all login attempts
// take 1 second (configurable) and serializing them through a mutex.
//
// Scripts can send `Authorization: <secret code>` instead of
// having to keep a cookie jar.
//
package gin_teeny_security

import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/sessions"
import "net/http"
import "net/url"
import "io"
import "time"
import "sync"
import "html/template"

// Convenient entry-point for those using gin-sessions and
// not wanting to override anything.
func RequiresSecretAccessCode(secretAccessCode, path string) gin.HandlerFunc {
	cfg := &Config{
		Path:   path,
		Secret: secretAccessCode,
	}

	return cfg.Middleware
}

// Main entry point
type Config struct {
	Path              string                     // defaults to 'login'
	Secret            string                     // the password
	RequireAuth       func(*gin.Context) bool    // defaults to always requiring auth if unset; override to allow some public access.
	Template          *template.Template         // Markup for the login page
	SaveKeyToSession  func(*gin.Context, string) // Override to use something other than gin-sessions
	GetKeyFromSession func(*gin.Context) string  // Override to use something other than gin-sessions

	LoginAttemptSlowdown time.Duration // Increase to slow-down attempts to brute force your password.
	mutex                sync.Mutex
}

func (c Config) saveKey(ctx *gin.Context, k string) {
	if c.SaveKeyToSession == nil {
		c.SaveKeyToSession = DefaultSetSession
	}
	c.SaveKeyToSession(ctx, k)
}

func (c Config) getKey(ctx *gin.Context) string {
	if c.GetKeyFromSession == nil {
		c.GetKeyFromSession = DefaultGetSession
	}
	return c.GetKeyFromSession(ctx)
}

// Saves your login status using gin-sessions
func DefaultSetSession(c *gin.Context, secret string) {
	session := sessions.Default(c)
	session.Set("secretAccessCode", secret)
	session.Save()
}

// Gets your login status from gin-sessions
func DefaultGetSession(c *gin.Context) string {
	session := sessions.Default(c)
	str, ok := session.Get("secretAccessCode").(string)
	if !ok {
		return ""
	}
	return str
}

func (c Config) path() string {
	if c.Path == "" {
		return "/login/"
	}
	return c.Path
}

func (c Config) requireAuth(ctx *gin.Context) bool {
	if ctx.Request.Header.Get("Authorization") != "" {
		// Slow down brute-force attempts.
		c.mutex.Lock()
		defer c.mutex.Unlock()
		time.Sleep(c.loginSlowdown())
	}
	if ctx.Request.Header.Get("Authorization") == c.Secret {
		return false
	}
	return c.RequireAuth == nil || c.RequireAuth(ctx)
}

func (c Config) template() *template.Template {
	if c.Template == nil {
		return DEFAULT_LOGIN_PAGE
	}
	return c.Template
}

func (c Config) loginSlowdown() time.Duration {
	if c.LoginAttemptSlowdown == 0 {
		return time.Second
	}
	return c.LoginAttemptSlowdown
}

func (c Config) ExecTemplate(w io.Writer, message, returnUrl string) error {
	return c.template().Execute(w, LoginPageParams{
		Message: message,
		Path:    c.path() + "?" + url.Values{"return": []string{returnUrl}}.Encode(),
	})
}

type LoginPageParams struct {
	Message string
	Path    string
}

var DEFAULT_LOGIN_PAGE = template.Must(template.New("login").Parse(`
<h1>Login</h1>
{{ if .Message }}<h2>{{ .Message }}</h2>{{ end }}
<form action="{{.Path}}" method="POST">
  <input type="password" name="secretAccessCode" />
  <input type="submit" value="Login" />
</form>

<div style="display: none">
CURL users: try setting -H 'Authorization: <your secret>'
</div>
`))

func (cfg *Config) Middleware(c *gin.Context) {
	if c.Request.URL.Path == cfg.path() {
		returnTo := c.Request.URL.Query().Get("return")
		if returnTo == "" {
			returnTo = "/"
		}

		if c.Request.Method == "POST" {
			// slow down brute-force attacks
			cfg.mutex.Lock()
			defer cfg.mutex.Unlock()
			time.Sleep(cfg.loginSlowdown())

			c.Request.ParseForm()

			if c.Request.PostForm.Get("secretAccessCode") == cfg.Secret {
				c.Header("Location", returnTo)
				cfg.saveKey(c, cfg.Secret)

				c.AbortWithStatus(http.StatusFound)
				return
			} else {
				cfg.saveKey(c, "")
				c.Writer.WriteHeader(http.StatusForbidden)
				cfg.ExecTemplate(c.Writer, "Wrong Password", returnTo)
				c.Abort()
				return
			}
		} else if c.Request.Method == "GET" {
			cfg.ExecTemplate(c.Writer, "", returnTo)
			c.Abort()
			return
		} else {
			c.Next()
			return
		}
	}

	v := cfg.getKey(c)
	if cfg.requireAuth(c) && (v != cfg.Secret) {
		c.Header("Location", cfg.Path+"?"+url.Values{"return": []string{c.Request.URL.RequestURI()}}.Encode())
		c.AbortWithStatus(http.StatusTemporaryRedirect)
	} else {
		c.Next()
	}
}
