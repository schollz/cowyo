package config

import (
  "fmt"
  "log"
"encoding/base64"
  "net/http"
  "github.com/gin-contrib/sessions"
  "github.com/jcelliott/lumber"
  "github.com/schollz/cowyo/server"
  "strings"
)
func (c Config) ListenAndServe() error {
  insecurePorts := map[int]bool{}
  securePorts := map[int]bool{}
  err := make(chan error)
  for _, s := range c.Sites {
    if !insecurePorts[*s.Port] {
      insecurePorts[*s.Port] = true
      go func(s SiteConfig) {
        err <- http.ListenAndServe(fmt.Sprintf("localhost:%d", *s.Port), c)
      }(s)
    }
    if s.TLS != nil && !securePorts[s.TLS.Port] {
      securePorts[s.TLS.Port] = true
      go func(s SiteConfig) {
        err <- http.ListenAndServeTLS(
          fmt.Sprintf("localhost:%d", s.TLS.Port),
          s.TLS.CertPath,
          s.TLS.KeyPath,
          c,
          )
      }(s)
    }
  }
  for {
    return <- err
  }
  return nil
}

func (c Config) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
  for i := range c.Sites {
    if c.Sites[i].MatchesRequest(r) {
      c.Sites[i].Handle(rw, r)
      return
    }
  }
  http.NotFound(rw, r)
}

func (s SiteConfig) MatchesRequest(r *http.Request) bool {
  sh := *s.Host
  if strings.HasPrefix(sh, "*") {
    return strings.HasSuffix(r.Host, sh[1:])
  }
  return sh == r.Host
}

func (s SiteConfig) sessionStore() sessions.Store {
  keys := [][]byte{}
  for _, k := range s.CookieKeys {
    key, err := base64.StdEncoding.DecodeString(k.AuthenticateBase64)
    if err != nil {
      panic(err)
    }
    if len(key) != 32 {
      log.Panicf("AuthenticateBase64 key %s must be 32 bytes; suggest %s", k.AuthenticateBase64, makeAuthKey())
    }

      keys = append(keys, key)
    key, err = base64.StdEncoding.DecodeString(k.EncryptBase64)
    if err != nil {
      panic(err)
    }

    if len(key) != 32 {
      log.Panicf("EncryptBase64 key %s must be 32 bytes, suggest %s", k.EncryptBase64, makeAuthKey())
    }
      keys = append(keys, key)
  }
  return sessions.NewStore(keys...)
}

func (s SiteConfig) Handle(rw http.ResponseWriter, r *http.Request) {
  dataDir := strings.Replace(*s.DataDir, "${HOST}", r.Host, -1)

  router := server.Site{
    PathToData:      dataDir,
    Css:             []byte{},
    DefaultPage:     *s.DefaultPage,
    DefaultPassword: *s.Lock,
    Debounce:        *s.DebounceSave,
    Diary:           *s.Diary,
    SessionStore:    s.sessionStore(),
    SecretCode:      *s.AccessCode,
    AllowInsecure:   *s.AllowInsecureMarkup,
    Fileuploads:     *s.MaxFileUploadMb > 0,
    MaxUploadSize:   *s.MaxFileUploadMb,
    Logger:          lumber.NewConsoleLogger(server.LogLevel),
    MaxDocumentSize: *s.MaxDocumentLength,
  }.Router()

  router.ServeHTTP(rw, r)
}
