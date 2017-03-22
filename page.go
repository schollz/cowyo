package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/schollz/versionedtext"
)

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
