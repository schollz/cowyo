package main

import (
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
	n := DirectoryList()
	if len(n) != 3 {
		t.Error("Expected three directory entries")
		t.FailNow()
	}
	if n[0].Name != "testpage" {
		t.Error("Expected testpage to be first")
	}
	if n[1].Name != "testpage2" {
		t.Error("Expected testpage2 to be second")
	}
	if n[2].Name != "testpage3" {
		t.Error("Expected testpage3 to be last")
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

	p3 := Open("testpage: childpage")
	err = p3.Update("**child content**")
	if err != nil {
		t.Error(err)
	}

	children := p.ChildPageNames()
	if len(children) != 1 {
		t.Errorf("Expected 1 child page to be found, got %d", len(children))
		return
	}
	if children[0] != "testpage: childpage" {
		t.Errorf("Expected child page %s to be found (got %s)", "testpage: childpage", children[0])
	}
}
