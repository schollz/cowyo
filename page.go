package main

import (
	"encoding/base32"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/schollz/versionedtext"
)

var pathToData = "data"

func init() {
	os.MkdirAll(pathToData, 0755)
}

type Page struct {
	Name                    string
	Text                    versionedtext.VersionedText
	RenderedPage            string
	IsLocked                bool
	PassphraseToUnlock      string
	IsEncrypted             bool
	IsPrimedForSelfDestruct bool
}

func Open(name string) (p *Page) {
	p = new(Page)
	p.Name = name
	p.Text = versionedtext.NewVersionedText("")
	p.Render()
	bJSON, err := ioutil.ReadFile(path.Join(pathToData, encodeToBase32(name)+".json"))
	if err != nil {
		return
	}
	err = json.Unmarshal(bJSON, &p)
	if err != nil {
		p = new(Page)
	}
	return p
}

func (p *Page) Update(newText string) error {
	p.Text.Update(newText)
	p.Render()
	return p.Save()
}

func (p *Page) Render() {
	if p.IsEncrypted {
		p.RenderedPage = "<code>" + p.Text.GetCurrent() + "</code>"
		return
	}
	p.RenderedPage = MarkdownToHtml(p.Text.GetCurrent())
}

func MarkdownToHtml(s string) string {
	unsafe := blackfriday.MarkdownCommon([]byte(s))
	pClean := bluemonday.UGCPolicy()
	pClean.AllowElements("img")
	pClean.AllowAttrs("alt").OnElements("img")
	pClean.AllowAttrs("src").OnElements("img")
	pClean.AllowAttrs("class").OnElements("a")
	pClean.AllowAttrs("href").OnElements("a")
	pClean.AllowAttrs("id").OnElements("a")
	pClean.AllowDataURIImages()
	html := pClean.SanitizeBytes(unsafe)
	return string(html)
}

func (p *Page) Save() error {
	bJSON, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(pathToData, encodeToBase32(p.Name)+".json"), bJSON, 0755)
}

func (p *Page) Erase() error {
	return os.Remove(path.Join(pathToData, encodeToBase32(p.Name)+".json"))
}

func encodeToBase32(s string) string {
	return base32.StdEncoding.EncodeToString([]byte(s))
}

func decodeFromBase32(s string) (s2 string, err error) {
	bString, err := base32.StdEncoding.DecodeString(s)
	s2 = string(bString)
	return
}
