package cowyo

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/schollz/cowyo2/internal/database"
	log "github.com/schollz/logger"
)

func TestPageEditLoggingIsDebugLevelAndIdentifiesMutation(t *testing.T) {
	t.Setenv("LOGGER", "")
	previousLevel := log.GetLevel()
	var output bytes.Buffer
	log.SetOutput(&output)
	t.Cleanup(func() {
		log.SetOutput(os.Stdout)
		log.SetLevel(previousLevel)
	})

	log.SetLevel("info")
	logPageEdit("quiet-page", "test", "edit", 4)
	if output.Len() != 0 {
		t.Fatalf("info-level output unexpectedly contains debug log %q", output.String())
	}

	setUpHandlerTest(t, Page{Title: "debug-page", Text: "before"})
	log.SetLevel("debug")
	if _, err := applyWebsocketUpdate(context.Background(), "debug-page", Page{
		Text: "after",
	}); err != nil {
		t.Fatalf("apply WebSocket update: %v", err)
	}

	logged := output.String()
	for _, field := range []string{
		`page edited:`,
		`path="/debug-page"`,
		`source="websocket"`,
		`operation="edit"`,
		`bytes=5`,
	} {
		if !strings.Contains(logged, field) {
			t.Errorf("debug log %q does not contain %q", logged, field)
		}
	}

	output.Reset()
	usePostingLimiter(t, 100, time.Minute, 100)
	request := newPostRequest("http://example.com/debug-page", "posted")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("apply HTTP POST update: %v", err)
	}

	logged = output.String()
	for _, field := range []string{
		`path="/debug-page"`,
		`source="http-post"`,
		`operation="edit"`,
		`bytes=6`,
	} {
		if !strings.Contains(logged, field) {
			t.Errorf("HTTP debug log %q does not contain %q", logged, field)
		}
	}

	if err := pageStore.UpsertPage(context.Background(), database.Page{
		Title:        "one-time-log",
		Text:         "burn",
		SelfDestruct: true,
	}); err != nil {
		t.Fatalf("seed self-destruct page: %v", err)
	}
	output.Reset()
	selfDestructRequest := httptest.NewRequest(http.MethodGet, "http://example.com/one-time-log", nil)
	selfDestructResponse := httptest.NewRecorder()
	if err := handle(selfDestructResponse, selfDestructRequest); err != nil {
		t.Fatalf("consume self-destruct page: %v", err)
	}

	logged = output.String()
	for _, field := range []string{
		`path="/one-time-log"`,
		`source="http-get"`,
		`operation="self-destruct"`,
		`bytes=4`,
	} {
		if !strings.Contains(logged, field) {
			t.Errorf("self-destruct debug log %q does not contain %q", logged, field)
		}
	}
}
