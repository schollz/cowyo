package cowyo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/schollz/cowyo2/internal/database"
)

func TestAPIPageStateAndLockLifecycle(t *testing.T) {
	const (
		title    = "api-lifecycle"
		content  = "keep this text"
		password = "correct horse battery staple"
	)
	setUpHandlerTest(t, Page{Title: title, Text: content})
	usePostingLimiter(t, 100, time.Minute, 100)
	usePermissivePageOperationLimiters(t)

	published := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationPublish,
	}, "192.0.2.1:1001")
	if published.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want %d", published.Code, http.StatusOK)
	}
	assertAPIOperationState(t, published, true, false, false, false)

	unpublished := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationUnpublish,
	}, "192.0.2.1:1001")
	assertAPIOperationState(t, unpublished, false, false, false, false)

	armed := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationSelfDestruct,
	}, "192.0.2.1:1001")
	assertAPIOperationState(t, armed, false, true, false, false)

	rejectedPublish := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationPublish,
	}, "192.0.2.1:1001")
	if rejectedPublish.Code != http.StatusConflict {
		t.Fatalf(
			"publish armed page status = %d, want %d",
			rejectedPublish.Code,
			http.StatusConflict,
		)
	}

	cancelled := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationCancelSelfDestruct,
	}, "192.0.2.1:1001")
	assertAPIOperationState(t, cancelled, false, false, false, false)

	locked := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationLock,
		Password:  password,
	}, "192.0.2.1:1001")
	assertAPIOperationState(t, locked, false, false, true, false)

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load API-locked page: %v", err)
	}
	if stored.Text != content {
		t.Errorf("API lock changed text to %q, want %q", stored.Text, content)
	}
	if stored.LockSalt == "" || stored.LockVerifier == "" {
		t.Fatal("API lock did not store lock verification credentials")
	}

	lockedPublish := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationPublish,
	}, "192.0.2.1:1001")
	if lockedPublish.Code != http.StatusLocked {
		t.Errorf(
			"publish locked page status = %d, want %d",
			lockedPublish.Code,
			http.StatusLocked,
		)
	}

	wrongUnlock := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationUnlock,
		Password:  "wrong page password",
	}, "192.0.2.1:1001")
	if wrongUnlock.Code != http.StatusForbidden {
		t.Errorf(
			"wrong unlock status = %d, want %d",
			wrongUnlock.Code,
			http.StatusForbidden,
		)
	}

	unlocked := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationUnlock,
		Password:  password,
	}, "192.0.2.1:1001")
	assertAPIOperationState(t, unlocked, false, false, false, false)

	stored, err = pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load API-unlocked page: %v", err)
	}
	if stored.Locked || stored.LockSalt != "" || stored.LockVerifier != "" {
		t.Fatal("API unlock did not clear page lock metadata")
	}
}

func TestAPIClientSideEncryptionLifecycle(t *testing.T) {
	const title = "api-encryption"
	setUpHandlerTest(t, Page{Title: title, Text: "plain text"})
	usePostingLimiter(t, 100, time.Minute, 100)
	usePermissivePageOperationLimiters(t)

	encryptedText := validEncryptedAPITestBlock(t)
	encrypted := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationEncrypt,
		Text:      &encryptedText,
	}, "192.0.2.2:1002")
	assertAPIOperationState(t, encrypted, false, false, false, true)

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load API-encrypted page: %v", err)
	}
	if stored.Text != encryptedText {
		t.Fatal("API encrypt operation did not store the encrypted block")
	}

	decryptedText := "decrypted locally"
	decrypted := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationDecrypt,
		Text:      &decryptedText,
	}, "192.0.2.2:1002")
	assertAPIOperationState(t, decrypted, false, false, false, false)

	stored, err = pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load API-decrypted page: %v", err)
	}
	if stored.Text != decryptedText {
		t.Errorf("decrypted text = %q, want %q", stored.Text, decryptedText)
	}
}

func TestAPIRejectsInvalidEncryptionAndPasswords(t *testing.T) {
	const title = "api-invalid-crypto"
	setUpHandlerTest(t, Page{Title: title, Text: "original"})
	usePostingLimiter(t, 100, time.Minute, 100)
	usePermissivePageOperationLimiters(t)

	invalidBlock := encryptedBlockStart + "\nnot-json\n" + encryptedBlockEnd
	for name, request := range map[string]apiPageOperationRequest{
		"malformed block": {
			Operation: operationEncrypt,
			Text:      &invalidBlock,
		},
		"server-side encryption password": {
			Operation: operationEncrypt,
			Password:  "do not send this",
			Text:      &invalidBlock,
		},
		"oversized transformed text": {
			Operation: operationEncrypt,
			Text: func() *string {
				text := strings.Repeat("x", int(maxPostBodyBytes)+1)
				return &text
			}(),
		},
		"oversized page-lock password": {
			Operation: operationLock,
			Password:  strings.Repeat("p", maxLockPasswordLen+1),
		},
	} {
		t.Run(name, func(t *testing.T) {
			response := performAPIOperation(
				t,
				title,
				request,
				"192.0.2.3:1003",
			)
			if response.Code != http.StatusBadRequest {
				t.Errorf(
					"status = %d, want %d",
					response.Code,
					http.StatusBadRequest,
				)
			}
		})
	}

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load page after invalid API encryption: %v", err)
	}
	if stored.Text != "original" {
		t.Fatal("invalid API encryption changed page text")
	}
}

func TestAPILockedCryptoPreservesProtectedPageState(t *testing.T) {
	const (
		title    = "api-locked-crypto"
		password = "correct horse battery staple"
		footer   = "\npublic footer"
	)
	encryptedText := validEncryptedAPITestBlock(t)
	setUpLockedHandlerTest(t, title, encryptedText+footer, password)
	usePostingLimiter(t, 100, time.Minute, 100)
	usePermissivePageOperationLimiters(t)

	rejectedEncrypt := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationEncrypt,
		Text:      &encryptedText,
	}, "192.0.2.4:1004")
	if rejectedEncrypt.Code != http.StatusLocked {
		t.Errorf(
			"locked encrypt status = %d, want %d",
			rejectedEncrypt.Code,
			http.StatusLocked,
		)
	}

	changedFooter := "decrypted locally\nchanged footer"
	rejectedDecrypt := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationDecrypt,
		Text:      &changedFooter,
	}, "192.0.2.4:1004")
	if rejectedDecrypt.Code != http.StatusBadRequest {
		t.Errorf(
			"changed-footer decrypt status = %d, want %d",
			rejectedDecrypt.Code,
			http.StatusBadRequest,
		)
	}

	decryptedText := "decrypted locally" + footer
	decrypted := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationDecrypt,
		Text:      &decryptedText,
	}, "192.0.2.4:1004")
	assertAPIOperationState(t, decrypted, false, false, true, false)

	stored, err := pageStore.GetPage(context.Background(), title)
	if err != nil {
		t.Fatalf("load locked API-decrypted page: %v", err)
	}
	if stored.Text != decryptedText ||
		!stored.Locked ||
		stored.LockSalt == "" ||
		stored.LockVerifier == "" {
		t.Fatalf("locked API decryption changed protected state: %+v", stored)
	}
}

func TestAPIStrictRequestValidation(t *testing.T) {
	const title = "api-validation"
	setUpHandlerTest(t, Page{Title: title, Text: "original"})
	usePostingLimiter(t, 100, time.Minute, 100)
	usePermissivePageOperationLimiters(t)

	tests := []struct {
		name        string
		method      string
		target      string
		contentType string
		body        string
		wantStatus  int
	}{
		{
			name:        "wrong method",
			method:      http.MethodGet,
			target:      apiOperationURL(title),
			contentType: "application/json",
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			target:      apiOperationURL(title),
			contentType: "text/plain",
			body:        `{"operation":"publish"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "unknown field",
			method:      http.MethodPost,
			target:      apiOperationURL(title),
			contentType: "application/json",
			body:        `{"operation":"publish","surprise":true}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "multiple objects",
			method:      http.MethodPost,
			target:      apiOperationURL(title),
			contentType: "application/json",
			body:        `{"operation":"publish"}{"operation":"unpublish"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "unsupported operation",
			method:      http.MethodPost,
			target:      apiOperationURL(title),
			contentType: "application/json",
			body:        `{"operation":"delete"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "state operation with hidden text",
			method:      http.MethodPost,
			target:      apiOperationURL(title),
			contentType: "application/json",
			body:        `{"operation":"publish","text":"spam"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "oversized request",
			method:      http.MethodPost,
			target:      apiOperationURL(title),
			contentType: "application/json",
			body:        strings.Repeat("x", maxAPIOperationRequestBytes+1),
			wantStatus:  http.StatusRequestEntityTooLarge,
		},
		{
			name:        "missing page",
			method:      http.MethodPost,
			target:      apiOperationURL("does-not-exist"),
			contentType: "application/json",
			body:        `{"operation":"publish"}`,
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "unknown API endpoint",
			method:      http.MethodPost,
			target:      "http://example.com/api/v1/nope",
			contentType: "application/json",
			body:        `{"operation":"publish"}`,
			wantStatus:  http.StatusNotFound,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				test.method,
				test.target,
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", test.contentType)
			request.RemoteAddr = "192.0.2." + strconv.Itoa(20+index) + ":2020"
			response := httptest.NewRecorder()

			if err := handle(response, request); err != nil {
				t.Fatalf("handle() error = %v", err)
			}
			if response.Code != test.wantStatus {
				t.Errorf(
					"status = %d, want %d; body = %s",
					response.Code,
					test.wantStatus,
					response.Body,
				)
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(
				got,
				"application/json",
			) {
				t.Errorf("Content-Type = %q, want JSON", got)
			}
		})
	}

	if _, err := pageStore.GetPage(
		context.Background(),
		"does-not-exist",
	); !errors.Is(err, database.ErrPageNotFound) {
		t.Fatalf("missing API operation created a page: %v", err)
	}
}

func TestAPIRateLimitsClientTargetAndCredentialOperations(t *testing.T) {
	const (
		title    = "api-rate-limits"
		password = "correct horse battery staple"
	)

	t.Run("shared mutation limiter", func(t *testing.T) {
		setUpHandlerTest(t, Page{Title: title, Text: "text"})
		usePostingLimiter(t, 1, time.Hour, 1)
		usePermissivePageOperationLimiters(t)

		first := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationPublish,
		}, "192.0.2.40:4040")
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
		}
		second := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationUnpublish,
		}, "192.0.2.40:4040")
		assertAPIRateLimited(t, second)
	})

	t.Run("per-client operation limiter", func(t *testing.T) {
		setUpHandlerTest(t, Page{Title: title, Text: "text"})
		usePostingLimiter(t, 100, time.Minute, 100)
		usePageOperationLimiterSet(
			t,
			newPostRateLimiter(1, time.Hour, 1),
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(100, time.Hour, 100),
		)

		first := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationPublish,
		}, "192.0.2.41:4141")
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
		}
		second := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationUnpublish,
		}, "192.0.2.41:4141")
		assertAPIRateLimited(t, second)
	})

	t.Run("per-page operation limiter", func(t *testing.T) {
		setUpHandlerTest(t, Page{Title: title, Text: "text"})
		usePostingLimiter(t, 100, time.Minute, 100)
		usePageOperationLimiterSet(
			t,
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(1, time.Hour, 1),
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(100, time.Hour, 100),
		)

		first := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationPublish,
		}, "192.0.2.42:4242")
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
		}
		second := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationUnpublish,
		}, "192.0.2.43:4343")
		assertAPIRateLimited(t, second)
	})

	t.Run("per-page credential limiter", func(t *testing.T) {
		setUpLockedHandlerTest(t, title, "text", password)
		usePostingLimiter(t, 100, time.Minute, 100)
		usePageOperationLimiterSet(
			t,
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(100, time.Hour, 100),
			newPostRateLimiter(1, time.Hour, 1),
		)

		first := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationUnlock,
			Password:  "first wrong password",
		}, "192.0.2.44:4444")
		if first.Code != http.StatusForbidden {
			t.Fatalf(
				"first wrong password status = %d, want %d",
				first.Code,
				http.StatusForbidden,
			)
		}
		second := performAPIOperation(t, title, apiPageOperationRequest{
			Operation: operationUnlock,
			Password:  "second wrong password",
		}, "192.0.2.45:4545")
		assertAPIRateLimited(t, second)
	})
}

func TestAdminKeyBypassesAPIMutationLimits(t *testing.T) {
	const (
		title    = "api-admin-rate"
		adminKey = "test-admin-key"
	)
	t.Setenv(adminPostKeyEnvironment, adminKey)
	setUpHandlerTest(t, Page{Title: title, Text: "text"})
	usePostingLimiter(t, 1, time.Hour, 1)
	usePageOperationLimiterSet(
		t,
		newPostRateLimiter(1, time.Hour, 1),
		newPostRateLimiter(1, time.Hour, 1),
		newPostRateLimiter(1, time.Hour, 1),
		newPostRateLimiter(1, time.Hour, 1),
	)

	for attempt, operation := range []string{
		operationPublish,
		operationUnpublish,
	} {
		response := performAPIOperation(
			t,
			title,
			apiPageOperationRequest{Operation: operation},
			"192.0.2.46:4646",
			adminKey,
		)
		if response.Code != http.StatusOK {
			t.Errorf(
				"admin attempt %d status = %d, want %d; body = %s",
				attempt+1,
				response.Code,
				http.StatusOK,
				response.Body,
			)
		}
	}
}

func TestAPIOperationBroadcastsToOpenBrowser(t *testing.T) {
	const title = "api-broadcast"
	setUpHandlerTest(t, Page{Title: title, Text: "text"})
	usePostingLimiter(t, 100, time.Minute, 100)
	usePermissivePageOperationLimiters(t)
	isolateWebsocketConnections(t)

	connection := &Connection{
		id:    "browser-client",
		place: title,
		send:  make(chan websocketMessage, 1),
		done:  make(chan struct{}),
	}
	connectionsMu.Lock()
	connections[connection.id] = connection
	connectionsMu.Unlock()

	response := performAPIOperation(t, title, apiPageOperationRequest{
		Operation: operationPublish,
	}, "192.0.2.50:5050")
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want %d", response.Code, http.StatusOK)
	}

	select {
	case message := <-connection.send:
		if message.Type != websocketMessageUpdate ||
			message.Operation != operationPublish ||
			!message.Published ||
			message.ClientID != "" {
			t.Errorf("broadcast message = %+v", message)
		}
	default:
		t.Fatal("API operation was not broadcast to the open browser")
	}
}

func TestWebsocketPageOperationsUseSharedRateLimiter(t *testing.T) {
	const title = "websocket-operation-rate"
	setUpHandlerTest(t, Page{Title: title, Text: "text"})
	usePageOperationLimiterSet(
		t,
		newPostRateLimiter(1, time.Hour, 1),
		newPostRateLimiter(100, time.Hour, 100),
		newPostRateLimiter(100, time.Hour, 100),
		newPostRateLimiter(100, time.Hour, 100),
	)
	isolateWebsocketConnections(t)

	connection := &Connection{
		id:    "rate-limited-browser",
		place: title,
		send:  make(chan websocketMessage, 4),
		done:  make(chan struct{}),
	}
	connectionsMu.Lock()
	connections[connection.id] = connection
	connectionsMu.Unlock()

	request := httptest.NewRequest(http.MethodGet, "http://example.com/ws", nil)
	request.RemoteAddr = "192.0.2.60:6060"
	applyWebsocketMessage(request, connection, websocketMessage{
		Type:      websocketMessageOperation,
		Operation: operationPublish,
	})
	first := <-connection.send
	if first.Type != websocketMessageAck || !first.Published {
		t.Fatalf("first WebSocket operation response = %+v", first)
	}

	applyWebsocketMessage(request, connection, websocketMessage{
		Type:      websocketMessageOperation,
		Operation: operationUnpublish,
	})
	second := <-connection.send
	if second.Type != websocketMessageError ||
		!strings.Contains(second.Error, "rate limit") {
		t.Fatalf("second WebSocket operation response = %+v", second)
	}
}

func performAPIOperation(
	t *testing.T,
	title string,
	operation apiPageOperationRequest,
	remoteAddr string,
	adminKeys ...string,
) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(operation)
	if err != nil {
		t.Fatalf("marshal API operation: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		apiOperationURL(title),
		strings.NewReader(string(body)),
	)
	request.Header.Set("Content-Type", "application/json")
	if len(adminKeys) > 0 {
		request.Header.Set(adminPostKeyHeader, adminKeys[0])
	}
	request.RemoteAddr = remoteAddr
	response := httptest.NewRecorder()
	if err := handle(response, request); err != nil {
		t.Fatalf("API handle() error = %v", err)
	}
	return response
}

func apiOperationURL(title string) string {
	return "http://example.com" +
		apiPageOperationPrefix +
		title +
		apiPageOperationSuffix
}

func assertAPIOperationState(
	t *testing.T,
	response *httptest.ResponseRecorder,
	published bool,
	selfDestruct bool,
	locked bool,
	encrypted bool,
) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf(
			"API status = %d, want %d; body = %s",
			response.Code,
			http.StatusOK,
			response.Body,
		)
	}

	var state apiPageOperationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode API response: %v", err)
	}
	if state.Published != published ||
		state.SelfDestruct != selfDestruct ||
		state.Locked != locked ||
		state.Encrypted != encrypted {
		t.Errorf(
			"API state = %+v, want published=%v self_destruct=%v locked=%v encrypted=%v",
			state,
			published,
			selfDestruct,
			locked,
			encrypted,
		)
	}
	if state.URL != "http://example.com/"+state.Title {
		t.Errorf("API URL = %q, want page URL for %q", state.URL, state.Title)
	}
}

func assertAPIRateLimited(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"status = %d, want %d; body = %s",
			response.Code,
			http.StatusTooManyRequests,
			response.Body,
		)
	}
	if response.Header().Get("Retry-After") == "" {
		t.Error("rate-limited API response has no Retry-After header")
	}
}

func validEncryptedAPITestBlock(t *testing.T) string {
	t.Helper()
	envelope, err := json.Marshal(encryptedBlockEnvelope{
		Version:    1,
		KDF:        "scrypt",
		N:          1 << 16,
		R:          8,
		P:          1,
		Salt:       base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
		Cipher:     "xchacha20-poly1305",
		Nonce:      base64.RawURLEncoding.EncodeToString(make([]byte, 24)),
		Ciphertext: base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
	})
	if err != nil {
		t.Fatalf("marshal encrypted test block: %v", err)
	}
	return encryptedBlockStart + "\n" + string(envelope) + "\n" + encryptedBlockEnd
}

func usePermissivePageOperationLimiters(t *testing.T) {
	t.Helper()
	usePageOperationLimiterSet(
		t,
		newPostRateLimiter(1000, time.Minute, 1000),
		newPostRateLimiter(1000, time.Minute, 1000),
		newPostRateLimiter(1000, time.Minute, 1000),
		newPostRateLimiter(1000, time.Minute, 1000),
	)
}

func usePageOperationLimiterSet(
	t *testing.T,
	client *postRateLimiter,
	target *postRateLimiter,
	credentialClient *postRateLimiter,
	credentialTarget *postRateLimiter,
) {
	t.Helper()

	previousClient := pageOperationClientLimiter
	previousTarget := pageOperationTargetLimiter
	previousCredentialClient := pageCredentialClientLimiter
	previousCredentialTarget := pageCredentialTargetLimiter
	pageOperationClientLimiter = client
	pageOperationTargetLimiter = target
	pageCredentialClientLimiter = credentialClient
	pageCredentialTargetLimiter = credentialTarget
	t.Cleanup(func() {
		pageOperationClientLimiter = previousClient
		pageOperationTargetLimiter = previousTarget
		pageCredentialClientLimiter = previousCredentialClient
		pageCredentialTargetLimiter = previousCredentialTarget
	})
}
