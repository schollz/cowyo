package main

import (
	// "fmt"
	"os"
	"strings"
	"testing"
)

func TestGeneral(t *testing.T) {
	defer os.RemoveAll("data")
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
