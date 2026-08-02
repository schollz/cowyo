package cowyo

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/schollz/cowyo2/internal/database"
	log "github.com/schollz/logger"
)

func testE2EECapability(seed byte) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strings.Repeat(string([]byte{seed}), 32)),
	)
}

func testE2EEDocument(seed byte) string {
	nonce := base64.RawURLEncoding.EncodeToString(make([]byte, 24))
	ciphertext := base64.RawURLEncoding.EncodeToString(
		[]byte{seed, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	)
	return e2eeDocumentStart + "\n" +
		`{"v":1,"cipher":"xchacha20-poly1305","nonce":"` + nonce +
		`","data":"` + ciphertext + `"}` + "\n" + e2eeDocumentEnd
}

func seedE2EEPage(t *testing.T, page database.Page, capability string) database.Page {
	t.Helper()
	hash, err := e2eeCapabilityHash(capability)
	if err != nil {
		t.Fatalf("hash capability: %v", err)
	}
	page.EndToEndEncrypted = true
	page.E2EEAuthHash = hash
	if err := pageStore.UpsertPage(context.Background(), page); err != nil {
		t.Fatalf("seed private page: %v", err)
	}
	return page
}

func TestE2EEEnvelopeAndCapabilityValidation(t *testing.T) {
	document := testE2EEDocument(1)
	if !validE2EEDocument(document) {
		t.Fatal("valid private document was rejected")
	}
	for _, invalid := range []string{
		"plaintext",
		document + "\n",
		strings.Replace(document, `"v":1`, `"v":2`, 1),
		strings.Replace(document, `"nonce":"`, `"extra":true,"nonce":"`, 1),
	} {
		if validE2EEDocument(invalid) {
			t.Fatalf("invalid private document accepted: %q", invalid)
		}
	}

	capability := testE2EECapability(7)
	hash, err := e2eeCapabilityHash(capability)
	if err != nil {
		t.Fatalf("hash capability: %v", err)
	}
	if hash == capability {
		t.Fatal("stored capability hash equals the raw capability")
	}
	if err := verifyE2EECapability(hash, capability); err != nil {
		t.Fatalf("verify capability: %v", err)
	}
	if err := verifyE2EECapability(hash, testE2EECapability(8)); !errors.Is(err, errE2EEInvalidCapability) {
		t.Fatalf("wrong capability error = %v", err)
	}
	for _, malformed := range []string{"", capability + "=", capability[:42]} {
		if _, err := e2eeCapabilityHash(malformed); !errors.Is(err, errE2EEInvalidCapability) {
			t.Fatalf("malformed capability %q error = %v", malformed, err)
		}
	}
}

func TestE2EEAtomicCreationConversionAndCollision(t *testing.T) {
	setUpHandlerTest(t, Page{Title: "ordinary", Text: "previous plaintext", Published: true})
	usePostingLimiter(t, 100, time.Minute, 100)
	isolateWebsocketConnections(t)
	request := httptest.NewRequest(http.MethodGet, "http://example.com/ws", nil)

	createConnection := &Connection{
		id: "creator", place: "private-new",
		send: make(chan websocketMessage, 4), done: make(chan struct{}),
	}
	capability := testE2EECapability(1)
	handleE2EEBootstrap(request, createConnection, websocketMessage{
		Type: websocketMessageE2EECreate, Text: testE2EEDocument(1),
		Capability: capability,
	})
	createdMessage := <-createConnection.send
	if createdMessage.Type != websocketMessageE2EEAuthenticated {
		t.Fatalf("create response = %+v", createdMessage)
	}
	created, err := pageStore.GetPage(context.Background(), "private-new")
	if err != nil || !created.EndToEndEncrypted || created.E2EEAuthHash == capability {
		t.Fatalf("created private page = %+v, %v", created, err)
	}

	collision := &Connection{
		id: "collision", place: "ordinary",
		send: make(chan websocketMessage, 2), done: make(chan struct{}),
	}
	handleE2EEBootstrap(request, collision, websocketMessage{
		Type: websocketMessageE2EECreate, Text: testE2EEDocument(2),
		Capability: testE2EECapability(2),
	})
	if message := <-collision.send; message.Type != websocketMessageError ||
		message.ErrorCode != "page-name-collision" ||
		!strings.Contains(message.Error, "already in use") {
		t.Fatalf("collision response = %+v", message)
	}
	ordinary, err := pageStore.GetPage(context.Background(), "ordinary")
	if err != nil || ordinary.EndToEndEncrypted || ordinary.Text != "previous plaintext" {
		t.Fatalf("collision converted ordinary page: %+v, %v", ordinary, err)
	}

	converter := &Connection{
		id: "converter", place: "ordinary",
		send: make(chan websocketMessage, 2), done: make(chan struct{}),
	}
	handleE2EEBootstrap(request, converter, websocketMessage{
		Type: websocketMessageE2EEConvert, Text: testE2EEDocument(3),
		Capability: testE2EECapability(3), CursorStart: 2, CursorEnd: 2,
	})
	if message := <-converter.send; message.Type != websocketMessageE2EEAuthenticated {
		t.Fatalf("conversion response = %+v", message)
	}
	converted, err := pageStore.GetPage(context.Background(), "ordinary")
	if err != nil || !converted.EndToEndEncrypted || converted.Published ||
		converted.Text != testE2EEDocument(3) {
		t.Fatalf("converted page = %+v, %v", converted, err)
	}
}

func TestE2EERejectsUnauthenticatedAndDowngradeMutations(t *testing.T) {
	const title = "private-rejections"
	setUpHandlerTest(t, Page{Title: "seed"})
	usePostingLimiter(t, 100, time.Minute, 100)
	capability := testE2EECapability(4)
	original := seedE2EEPage(t, database.Page{
		Title: title, Text: testE2EEDocument(4),
	}, capability)

	if _, err := applyAuthenticatedWebsocketUpdate(
		context.Background(), title, pageUpdate{Text: testE2EEDocument(5)}, false,
	); !errors.Is(err, errE2EEAuthenticationRequired) {
		t.Fatalf("unauthenticated edit error = %v", err)
	}
	if _, err := applyAuthenticatedWebsocketUpdate(
		context.Background(), title, pageUpdate{Text: "plaintext"}, true,
	); !errors.Is(err, errE2EEInvalidDocument) {
		t.Fatalf("plaintext edit error = %v", err)
	}
	for _, operation := range []string{
		operationEncrypt, operationDecrypt, operationPublish, operationUnpublish,
	} {
		if _, err := applyAuthenticatedWebsocketUpdate(
			context.Background(), title, pageUpdate{Operation: operation}, true,
		); !errors.Is(err, errE2EEIncompatibleOperation) {
			t.Errorf("%s error = %v", operation, err)
		}
	}

	for _, admin := range []bool{false, true} {
		request := newPostRequest("http://example.com/"+title, "plaintext replacement")
		if admin {
			t.Setenv(adminPostKeyEnvironment, "admin-secret")
			request.Header.Set(adminPostKeyHeader, "admin-secret")
		}
		response := httptest.NewRecorder()
		if err := handle(response, request); err != nil {
			t.Fatalf("named POST: %v", err)
		}
		if response.Code != http.StatusForbidden {
			t.Errorf("admin=%t POST status = %d, want %d", admin, response.Code, http.StatusForbidden)
		}
	}

	for _, operation := range []string{
		operationPublish, operationUnpublish, operationLock, operationUnlock,
		operationEncrypt, operationDecrypt, operationSelfDestruct,
		operationCancelSelfDestruct,
	} {
		request := apiPageOperationRequest{Operation: operation}
		if operation == operationLock || operation == operationUnlock {
			request.Password = "correct horse battery staple"
		}
		if operation == operationEncrypt || operation == operationDecrypt {
			text := testE2EEDocument(9)
			request.Text = &text
		}
		response := performAPIOperation(t, title, request, "192.0.2.90:9090", "admin-secret")
		if response.Code != http.StatusForbidden {
			t.Errorf("API %s status = %d, want %d", operation, response.Code, http.StatusForbidden)
		}
	}

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil || stored.Text != original.Text || !stored.EndToEndEncrypted {
		t.Fatalf("rejected mutations changed page: %+v, %v", stored, err)
	}
}

func TestE2EESelfDestructRequiresCapabilityAndReturnsExactlyOnce(t *testing.T) {
	const title = "private-final"
	setUpHandlerTest(t, Page{Title: "seed"})
	capability := testE2EECapability(5)
	document := testE2EEDocument(5)
	seedE2EEPage(t, database.Page{
		Title: title, Text: document, SelfDestruct: true,
	}, capability)

	for _, userAgent := range []string{"curl/8.0", "Mozilla/5.0"} {
		request := httptest.NewRequest(http.MethodGet, "http://example.com/"+title, nil)
		request.Header.Set("User-Agent", userAgent)
		response := httptest.NewRecorder()
		if err := handle(response, request); err != nil {
			t.Fatalf("keyless GET: %v", err)
		}
		if strings.Contains(response.Body.String(), document) {
			t.Errorf("keyless %s GET exposed final ciphertext", userAgent)
		}
		if _, err := pageStore.GetPage(context.Background(), title); err != nil {
			t.Fatalf("keyless GET consumed private page: %v", err)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/ws", nil)
	unauthenticated := &Connection{
		id: "unauthenticated", place: title,
		send: make(chan websocketMessage, 1), done: make(chan struct{}),
	}
	unauthenticated.e2eePage.Store(true)
	applyWebsocketMessage(request, unauthenticated, websocketMessage{
		Type: websocketMessageEdit, Text: testE2EEDocument(8),
	})
	if message := <-unauthenticated.send; message.Type != websocketMessageError || message.Text != "" {
		t.Fatalf("unauthenticated mutation exposed final ciphertext: %+v", message)
	}

	wrong := &Connection{
		id: "wrong", place: title,
		send: make(chan websocketMessage, 1), done: make(chan struct{}),
	}
	handleE2EEAuthentication(request, wrong, websocketMessage{
		Capability: testE2EECapability(6),
	})
	if message := <-wrong.send; message.Type != websocketMessageError {
		t.Fatalf("wrong capability response = %+v", message)
	}
	if _, err := pageStore.GetPage(context.Background(), title); err != nil {
		t.Fatalf("wrong capability consumed page: %v", err)
	}

	first := &Connection{
		id: "first", place: title,
		send: make(chan websocketMessage, 1), done: make(chan struct{}),
	}
	handleE2EEAuthentication(request, first, websocketMessage{Capability: capability})
	message := <-first.send
	if message.Type != websocketMessageE2EEAuthenticated || !message.Final || message.Text != document {
		t.Fatalf("authorized final response = %+v", message)
	}
	if _, err := pageStore.GetPage(context.Background(), title); !errors.Is(err, database.ErrPageNotFound) {
		t.Fatalf("authorized final load did not delete page: %v", err)
	}

	second := &Connection{
		id: "second", place: title,
		send: make(chan websocketMessage, 1), done: make(chan struct{}),
	}
	handleE2EEAuthentication(request, second, websocketMessage{Capability: capability})
	if message := <-second.send; message.Type != websocketMessageError {
		t.Fatalf("second final response = %+v", message)
	}
}

func TestE2EEWebsocketRequiresAuthenticationForOperationsAndCursors(t *testing.T) {
	const title = "private-socket-auth"
	setUpHandlerTest(t, Page{Title: "seed"})
	usePermissivePageOperationLimiters(t)
	isolateWebsocketConnections(t)
	capability := testE2EECapability(11)
	seedE2EEPage(t, database.Page{
		Title: title, Text: testE2EEDocument(11),
	}, capability)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handle(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)
	socketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?place=" + title
	dial := func(name string) *websocket.Conn {
		t.Helper()
		connection, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
		if err != nil {
			t.Fatalf("dial %s websocket: %v", name, err)
		}
		t.Cleanup(func() { _ = connection.Close() })
		if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("set %s deadline: %v", name, err)
		}
		return connection
	}
	first := dial("first")

	if err := first.WriteJSON(websocketMessage{Type: websocketMessageCursor}); err != nil {
		t.Fatalf("write unauthenticated cursor: %v", err)
	}
	var response websocketMessage
	if err := first.ReadJSON(&response); err != nil {
		t.Fatalf("read cursor rejection: %v", err)
	}
	if response.Type != websocketMessageError || !strings.Contains(response.Error, "Authenticate") {
		t.Fatalf("unauthenticated cursor response = %+v", response)
	}

	if err := first.WriteJSON(websocketMessage{
		Type: websocketMessageOperation, Operation: operationSelfDestruct,
	}); err != nil {
		t.Fatalf("write unauthenticated operation: %v", err)
	}
	if err := first.ReadJSON(&response); err != nil {
		t.Fatalf("read operation rejection: %v", err)
	}
	if response.Type != websocketMessageError || !strings.Contains(response.Error, "Authenticate") {
		t.Fatalf("unauthenticated operation response = %+v", response)
	}

	if err := first.WriteJSON(websocketMessage{
		Type: websocketMessageE2EEAuthenticate, Capability: testE2EECapability(12),
	}); err != nil {
		t.Fatalf("write wrong authentication: %v", err)
	}
	if err := first.ReadJSON(&response); err != nil {
		t.Fatalf("read wrong authentication response: %v", err)
	}
	if response.Type != websocketMessageError || !strings.Contains(response.Error, "invalid") {
		t.Fatalf("wrong authentication response = %+v", response)
	}

	if err := first.WriteJSON(websocketMessage{
		Type: websocketMessageE2EEAuthenticate, Capability: capability,
	}); err != nil {
		t.Fatalf("write authentication: %v", err)
	}
	if err := first.ReadJSON(&response); err != nil {
		t.Fatalf("read authentication response: %v", err)
	}
	if response.Type != websocketMessageE2EEAuthenticated ||
		!response.EndToEndEncrypted ||
		response.Text != testE2EEDocument(11) {
		t.Fatalf("authentication response = %+v", response)
	}

	second := dial("second")
	if err := second.WriteJSON(websocketMessage{
		Type: websocketMessageE2EEAuthenticate, Capability: capability,
	}); err != nil {
		t.Fatalf("authenticate second connection: %v", err)
	}
	if err := second.ReadJSON(&response); err != nil {
		t.Fatalf("read second authentication: %v", err)
	}
	if response.Type != websocketMessageE2EEAuthenticated {
		t.Fatalf("second authentication response = %+v", response)
	}

	if err := first.WriteJSON(websocketMessage{
		Type: websocketMessageCursor, CursorStart: 3, CursorEnd: 3,
	}); err != nil {
		t.Fatalf("write authenticated cursor: %v", err)
	}
	if err := second.ReadJSON(&response); err != nil {
		t.Fatalf("read authenticated cursor: %v", err)
	}
	if response.Type != websocketMessageCursor || response.CursorEnd != 3 {
		t.Fatalf("authenticated cursor response = %+v", response)
	}
}

func TestE2EEResponsesOmitExternalScriptsAndUseGenericMetadata(t *testing.T) {
	const title = "private-response"
	t.Setenv(googleTagEnvironment, "G-PRIVATE-TEST")
	t.Setenv(googleAdSenseEnvironment, "ca-pub-1234567890123456")
	t.Setenv(umamiURLEnvironment, "https://analytics.example")
	t.Setenv(umamiWebsiteIDEnvironment, "123e4567-e89b-12d3-a456-426614174000")
	setUpHandlerTest(t, Page{Title: "seed"})
	seedE2EEPage(t, database.Page{
		Title: title, Text: testE2EEDocument(7), Published: true,
	}, testE2EECapability(7))

	for _, path := range []string{"/" + title, "/fresh-fox?private=1", "/ordinary?convert=1"} {
		request := httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
		request.Header.Set("User-Agent", "Mozilla/5.0")
		response := httptest.NewRecorder()
		if err := handle(response, request); err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body := response.Body.String()
		for _, secret := range []string{
			"googletagmanager.com", "pagead2.googlesyndication.com",
			"analytics.example", "G-PRIVATE-TEST",
		} {
			if strings.Contains(body, secret) {
				t.Errorf("GET %s includes external tracker %q", path, secret)
			}
		}
		if got := response.Header().Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("GET %s Referrer-Policy = %q", path, got)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/"+title, nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("GET private page: %v", err)
	}
	body := response.Body.String()
	if !strings.Contains(body, "Private scratchpad — cowyo") ||
		!strings.Contains(body, robotsDirective(false)) ||
		strings.Contains(body, "private-response —") {
		t.Fatalf("private metadata is not generic/noindex")
	}
}

func TestE2EELoggingExcludesCredentialsAndPlaintext(t *testing.T) {
	setUpHandlerTest(t, Page{
		Title: "private-log", Text: "plaintext-that-must-not-be-logged",
	})
	isolateWebsocketConnections(t)
	previousLevel := log.GetLevel()
	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetLevel("debug")
	t.Cleanup(func() {
		log.SetOutput(os.Stdout)
		log.SetLevel(previousLevel)
	})

	capability := testE2EECapability(19)
	connection := &Connection{
		id: "private-log-client", place: "private-log",
		send: make(chan websocketMessage, 1), done: make(chan struct{}),
	}
	handleE2EEBootstrap(
		httptest.NewRequest(http.MethodGet, "http://example.com/ws", nil),
		connection,
		websocketMessage{
			Type: websocketMessageE2EEConvert, Text: testE2EEDocument(19),
			Capability: capability,
		},
	)
	<-connection.send
	logged := output.String()
	if !strings.Contains(logged, `operation="e2ee-convert"`) {
		t.Fatalf("conversion audit log is missing: %q", logged)
	}
	for _, secret := range []string{
		capability,
		"plaintext-that-must-not-be-logged",
	} {
		if strings.Contains(logged, secret) {
			t.Fatalf("E2EE log leaked %q: %q", secret, logged)
		}
	}
}
