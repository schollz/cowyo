// A GIN middleware providing low-fi security for personal stuff.

package gin_teeny_security

import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/sessions"
import "net/http"
import "net/url"
import "io"
import "time"
import "sync"
import "html/template"

// Forces you to a login page until you provide a secret code.
// No CSRF protection, so any script on any page can log you
// out (or in, if they know the password).
// The rest of your site needs XSS protection on forms or any site on the
// net can inject stuff. If you're sending open CORS headers this
// would be particularly bad.
func RequiresSecretAccessCode(secretAccessCode, path string) gin.HandlerFunc {
	cfg := &Config{
		Path:   path,
		Secret: secretAccessCode,
	}

	return cfg.Middleware
}

type Config struct {
	Path              string // defaults to login
	Secret            string
	RequireAuth       func(*gin.Context) bool // defaults to always requiring auth if unset
	Template          *template.Template
	SaveKeyToSession  func(*gin.Context, string)
	GetKeyFromSession func(*gin.Context) string

	LoginAttemptSlowdown time.Duration
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

func DefaultSetSession(c *gin.Context, secret string) {
	session := sessions.Default(c)
	session.Set("secretAccessCode", secret)
	session.Save()
}

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
