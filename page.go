package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/schollz/versionedtext"
)

// Page is the basic struct
type Page struct {
	Name                    string
	Text                    versionedtext.VersionedText
	Meta                    string
	RenderedPage            string
	IsLocked                bool
	PassphraseToUnlock      string
	IsEncrypted             bool
	IsPrimedForSelfDestruct bool
	IsPublished             bool
	UnlockedFor             string
}

func (p Page) LastEditTime() time.Time {
	return time.Unix(p.LastEditUnixTime(), 0)
}

func (p Page) LastEditUnixTime() int64 {
	return p.Text.LastEditTime() / 1000000000
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

type DirectoryEntry struct {
	Path       string
	Length     int
	Numchanges int
	LastEdited time.Time
}

func (d DirectoryEntry) LastEditTime() string {
	return d.LastEdited.Format("Mon Jan 2 15:04:05 MST 2006")
}

func (d DirectoryEntry) Name() string {
	return d.Path
}

func (d DirectoryEntry) Size() int64 {
	return int64(d.Length)
}

func (d DirectoryEntry) Mode() os.FileMode {
	return os.ModePerm
}

func (d DirectoryEntry) ModTime() time.Time {
	return d.LastEdited
}

func (d DirectoryEntry) IsDir() bool {
	return false
}

func (d DirectoryEntry) Sys() interface{} {
	return nil
}

func DirectoryList() []os.FileInfo {
	files, _ := ioutil.ReadDir(pathToData)
	entries := make([]os.FileInfo, len(files))
	for i, f := range files {
		name := DecodeFileName(f.Name())
		p := Open(name)
		entries[i] = DirectoryEntry{
			Path:       name,
			Length:     len(p.Text.GetCurrent()),
			Numchanges: p.Text.NumEdits(),
			LastEdited: time.Unix(p.Text.LastEditTime()/1000000000, 0),
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ModTime().After(entries[j].ModTime()) })
	return entries
}

type UploadEntry struct {
	os.FileInfo
}

func UploadList() ([]os.FileInfo, error) {
	paths, err := filepath.Glob(path.Join(pathToData, "sha256*"))
	if err != nil {
		return nil, err
	}
	result := make([]os.FileInfo, len(paths))
	for i := range paths {
		result[i], err = os.Stat(paths[i])
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func DecodeFileName(s string) string {
	s2, _ := decodeFromBase32(strings.Split(s, ".")[0])
	return s2
}

// Update cleans the text and updates the versioned text
// and generates a new render
func (p *Page) Update(newText string) error {
	// Trim space from end
	newText = strings.TrimRight(newText, "\n\t ")

	// Update the versioned text
	p.Text.Update(newText)

	// Render the new page
	p.Render()

	return p.Save()
}

var rBracketPage = regexp.MustCompile(`\[\[(.*?)\]\]`)

func (p *Page) Render() {
	if p.IsEncrypted {
		p.RenderedPage = "<code>" + p.Text.GetCurrent() + "</code>"
		return
	}

	// Convert [[page]] to [page](/page/view)
	currentText := p.Text.GetCurrent()
	for _, s := range rBracketPage.FindAllString(currentText, -1) {
		currentText = strings.Replace(currentText, s, "["+s[2:len(s)-2]+"](/"+s[2:len(s)-2]+"/view)", 1)
	}
	p.Text.Update(currentText)
	p.RenderedPage = MarkdownToHtml(p.Text.GetCurrent())
}

var saveMut = sync.Mutex{}

func (p *Page) Save() error {
	saveMut.Lock()
	defer saveMut.Unlock()
	bJSON, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(pathToData, encodeToBase32(strings.ToLower(p.Name))+".json"), bJSON, 0644)
}

func (p *Page) ChildPageNames() []string {
	prefix := strings.ToLower(p.Name + ": ")
	files, err := filepath.Glob(path.Join(pathToData, "*"))
	if err != nil {
		panic("Filepath pattern cannot be malformed")
	}

	result := []string{}
	for i := range files {
		basename := filepath.Base(files[i])
		if strings.HasSuffix(basename, ".json") {
			cname, err := decodeFromBase32(basename[:len(basename)-len(".json")])
			if err == nil && strings.HasPrefix(strings.ToLower(cname), prefix) {
				result = append(result, cname)
			}
		}
	}
	return result
}

func (p *Page) IsNew() bool {
	return !exists(path.Join(pathToData, encodeToBase32(strings.ToLower(p.Name))+".json"))
}

func (p *Page) Erase() error {
	log.Trace("Erasing " + p.Name)
	return os.Remove(path.Join(pathToData, encodeToBase32(strings.ToLower(p.Name))+".json"))
}

func (p *Page) Published() bool {
	return p.IsPublished
}
