package config

import (
  "github.com/BurntSushi/toml"
)

func ParseFile(path string) (Config, error) {
  c := Config{}
  if _, err := toml.DecodeFile("multisite_sample.toml", &c); err != nil {
    // handle error
    return c, err
  }
  c.SetDefaults()
  c.Validate()
  return c, nil
}

type Config struct {
  Default SiteConfig
  Sites   []SiteConfig
}

type SiteConfig struct {
  Host                *string
  Port                *int
  DataDir             *string
  DefaultPage         *string
  AllowInsecureMarkup *bool
  Lock                *string
  DebounceSave        *int
  Diary               *bool
  AccessCode          *string
  FileUploadsAllowed  *bool
  MaxFileUploadMb     *uint
  MaxDocumentLength *uint
  TLS                 *TLSConfig
  CookieKeys []CookieKey
}

type TLSConfig struct {
  CertPath string
  KeyPath  string
  Port     int
}

type CookieKey struct {
  AuthenticateBase64 string
  EncryptBase64 string
}

func (c Config) Validate() {
  for _, v := range c.Sites {
    v.sessionStore()
  }
}