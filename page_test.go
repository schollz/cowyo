package main

import (
	// "fmt"
	"os"
	"strings"
	"testing"
)

func TestListFiles(t *testing.T) {
	pathToData = "testdata"
	os.MkdirAll(pathToData, 0755)
	defer os.RemoveAll(pathToData)
	p := Open("testpage")
	p.Update("Some data")
	p = Open("testpage2")
	p.Update("A different bunch of data")
	p = Open("testpage3")
	p.Update("Not much else")
	n, l := DirectoryList()
	if strings.Join(n, " ") != "testpage testpage2 testpage3" {
		t.Errorf("Names: %s, Lengths: %d", n, l)
	}
}

func TestGeneral(t *testing.T) {
	pathToData = "testdata"
	os.MkdirAll(pathToData, 0755)
	defer os.RemoveAll(pathToData)
	p := Open("testpage")
	err := p.Update("**bold**")
	if err != nil {
		t.Error(err)
	}
	if strings.TrimSpace(p.RenderedPage) != "<p><strong>bold</strong></p>" {
		t.Errorf("Did not render: '%s'", p.RenderedPage)
	}
	err = p.Update("**bold** and *italic*")
	if err != nil {
		t.Error(err)
	}
	p.Save()

	p2 := Open("testpage")
	if strings.TrimSpace(p2.RenderedPage) != "<p><strong>bold</strong> and <em>italic</em></p>" {
		t.Errorf("Did not render: '%s'", p2.RenderedPage)
	}

}
