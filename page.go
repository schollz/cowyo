package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/schollz/versionedtext"
)

// Page is the basic struct
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
	bJSON, err := ioutil.ReadFile(path.Join(pathToData, encodeToBase32(strings.ToLower(name))+".json"))
	if err != nil {
		return
	}
	err = json.Unmarshal(bJSON, &p)
	if err != nil {
		p = new(Page)
	}
	return p
}

func DirectoryList() (names []string, lengths []int, numchanges []int, lastEdited []string) {
	files, _ := ioutil.ReadDir(pathToData)
	names = make([]string, len(files))
	lengths = make([]int, len(files))
	numchanges = make([]int, len(files))
	lastEdited = make([]string, len(files))
	for i, f := range files {
		names[i] = DecodeFileName(f.Name())
		p := Open(names[i])
		lengths[i] = len(p.Text.GetCurrent())
		numchanges[i] = p.Text.NumEdits()
		lastEdited[i] = time.Unix(p.Text.LastEditTime()/1000000000, 0).Format("Mon Jan 2 15:04:05 MST 2006")
	}
	return
}

func DecodeFileName(s string) string {
	s2, _ := decodeFromBase32(strings.Split(s, ".")[0])
	return s2
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

	// Convert [[page]] to [page](/page/view)
	r, _ := regexp.Compile("\\[\\[(.*?)\\]\\]")
	currentText := p.Text.GetCurrent()
	for _, s := range r.FindAllString(currentText, -1) {
		currentText = strings.Replace(currentText, s, "["+s[2:len(s)-2]+"](/"+s[2:len(s)-2]+"/view)", 1)
	}
	p.Text.Update(currentText)
	p.RenderedPage = MarkdownToHtml(p.Text.GetCurrent())
}

func (p *Page) Save() error {
	bJSON, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(pathToData, encodeToBase32(strings.ToLower(p.Name))+".json"), bJSON, 0755)
}

func (p *Page) Erase() error {
	log.Trace("Erasing " + p.Name)
	return os.Remove(path.Join(pathToData, encodeToBase32(strings.ToLower(p.Name))+".json"))
}
