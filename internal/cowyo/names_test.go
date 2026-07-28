package cowyo

import (
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var alliterativeNamePattern = regexp.MustCompile(`^[a-z]+-[a-z]+$`)

func TestAlliterations(t *testing.T) {
	got := alliterations(
		[]string{"calm", "silly", "bad slug", "Zany"},
		[]string{"cat", "salmon", "dog", "zebra"},
	)
	want := []string{"calm-cat", "silly-salmon"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("alliterations() = %v, want %v", got, want)
	}
}

func TestDocumentNamePool(t *testing.T) {
	if len(documentNames) < 1_000 {
		t.Fatalf("document name pool contains %d names, want at least 1000", len(documentNames))
	}

	for _, example := range []string{"agile-alligator", "calm-cat", "silly-salmon"} {
		if !slices.Contains(documentNames, example) {
			t.Errorf("document name pool does not contain %q", example)
		}
	}
	if slices.Contains(documentNames, "call-cat") {
		t.Error("document name pool still contains the old verb-animal name \"call-cat\"")
	}

	seen := make(map[string]struct{}, len(documentNames))
	for _, name := range documentNames {
		if !isAlliterativeDocumentName(name) {
			t.Errorf("document name %q is not an alliterative URL slug", name)
		}
		if _, exists := seen[name]; exists {
			t.Errorf("document name %q appears more than once", name)
		}
		seen[name] = struct{}{}
	}
}

func TestRandomDocumentName(t *testing.T) {
	names := make(map[string]struct{}, len(documentNames))
	for _, name := range documentNames {
		names[name] = struct{}{}
	}

	for attempt := 0; attempt < 100; attempt++ {
		name, err := randomDocumentName()
		if err != nil {
			t.Fatalf("randomDocumentName() error = %v", err)
		}
		if _, ok := names[name]; !ok {
			t.Fatalf("randomDocumentName() = %q, not in configured pool", name)
		}
	}
}

func isAlliterativeDocumentName(name string) bool {
	if !alliterativeNamePattern.MatchString(name) {
		return false
	}
	parts := strings.Split(name, "-")
	return len(parts) == 2 && parts[0][0] == parts[1][0]
}
