package config

import (
"encoding/base64"
  "crypto/rand"
)
var DefaultSiteConfig SiteConfig


func makeAuthKey() string {
  secret := make([]byte, 32)
  _, err := rand.Read(secret)
  if err != nil {
    panic(err)
  }
  return base64.StdEncoding.EncodeToString(secret)
}

func init() {
  host := "*"
  port := 8050
  debounce := 500
  dataDir := "data"
  empty := ""
  zer := uint(0)
  lots := uint(100000000)
  fal := false

  ck := CookieKey{
    AuthenticateBase64: "",
    EncryptBase64: "",
  }

  DefaultSiteConfig = SiteConfig{
    Host:&host,
Port:&port,
DataDir:&dataDir,
DebounceSave:&debounce,
CookieKeys: []CookieKey{ck},
DefaultPage:&empty,
AllowInsecureMarkup:&fal,
Lock:&empty,
Diary:&fal,
AccessCode:&empty,
FileUploadsAllowed:&fal,
MaxFileUploadMb:&zer,
MaxDocumentLength:&lots,
  }
}


func copyDefaults(base, defaults *SiteConfig) {
  if base.Host == nil {
    base.Host = defaults.Host
  }
  if base.Port == nil {
    base.Port = defaults.Port
  }
  if base.DataDir == nil {
    base.DataDir = defaults.DataDir
  }
  if base.DefaultPage == nil {
    base.DefaultPage = defaults.DefaultPage
  }
  if base.AllowInsecureMarkup == nil {
    base.AllowInsecureMarkup = defaults.AllowInsecureMarkup
  }
  if base.Lock == nil {
    base.Lock = defaults.Lock
  }
  if base.DebounceSave == nil {
    base.DebounceSave = defaults.DebounceSave
  }
  if base.Diary == nil {
    base.Diary = defaults.Diary
  }
  if base.AccessCode == nil {
    base.AccessCode = defaults.AccessCode
  }
  if base.FileUploadsAllowed == nil {
    base.FileUploadsAllowed = defaults.FileUploadsAllowed
  }
  if base.MaxFileUploadMb == nil {
    base.MaxFileUploadMb = defaults.MaxFileUploadMb
  }
  if base.MaxDocumentLength == nil {
    base.MaxDocumentLength = defaults.MaxDocumentLength
  }
  if base.TLS == nil {
    base.TLS = defaults.TLS
  }
  if base.CookieKeys == nil {
    base.CookieKeys = defaults.CookieKeys
  }
}

func (c *Config) SetDefaults() {
  copyDefaults(&c.Default, &DefaultSiteConfig)
  for i := range c.Sites {
    copyDefaults(&c.Sites[i], &c.Default)
  }
}
