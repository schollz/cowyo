package cowyo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/schollz/cowyo2/internal/database"
)

const (
	apiRoot                     = "/api"
	apiPrefix                   = "/api/"
	apiPageOperationPrefix      = "/api/v1/pages/"
	apiPageOperationSuffix      = "/operations"
	maxAPIOperationRequestBytes = 64 * 1024
	maxAPIPageTitleBytes        = 200
)

type apiPageOperationRequest struct {
	Operation string  `json:"operation"`
	Password  string  `json:"password,omitempty"`
	Text      *string `json:"text,omitempty"`
}

type apiPageOperationResponse struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	Operation    string `json:"operation"`
	Published    bool   `json:"published"`
	SelfDestruct bool   `json:"self_destruct"`
	Locked       bool   `json:"locked"`
	Encrypted    bool   `json:"encrypted"`
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

func handleAPI(w http.ResponseWriter, r *http.Request) error {
	title, ok := apiPageOperationTitle(r.URL.Path)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "API endpoint not found")
		return nil
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	return handleAPIPageOperation(w, r, title)
}

func apiPageOperationTitle(path string) (string, bool) {
	if !strings.HasPrefix(path, apiPageOperationPrefix) ||
		!strings.HasSuffix(path, apiPageOperationSuffix) {
		return "", false
	}

	title := strings.TrimSuffix(
		strings.TrimPrefix(path, apiPageOperationPrefix),
		apiPageOperationSuffix,
	)
	title = strings.TrimSuffix(title, "/")
	if !validAPIPageTitle(title) {
		return "", false
	}
	return title, true
}

func validAPIPageTitle(title string) bool {
	if title == "" ||
		len(title) > maxAPIPageTitleBytes ||
		!utf8.ValidString(title) ||
		title == "about" ||
		strings.HasPrefix(title, "api/") {
		return false
	}
	for _, character := range title {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func handleAPIPageOperation(
	w http.ResponseWriter,
	r *http.Request,
	title string,
) error {
	admin := isAdminPost(r)
	if !admin {
		allowed, retryAfter := postingLimiter.Allow(postClientKey(r))
		if !allowed {
			writeAPIRateLimitError(w, retryAfter, "mutation rate limit exceeded")
			return nil
		}
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeAPIError(
			w,
			http.StatusUnsupportedMediaType,
			"Content-Type must be application/json",
		)
		return nil
	}

	if r.ContentLength > maxAPIOperationRequestBytes {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "API request is too large")
		return nil
	}

	var request apiPageOperationRequest
	body := http.MaxBytesReader(w, r.Body, maxAPIOperationRequestBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "API request is too large")
			return nil
		}
		writeAPIError(w, http.StatusBadRequest, "request body must be one valid JSON object")
		return nil
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return nil
	}

	stored, err := pageStore.GetPage(r.Context(), title)
	if errors.Is(err, database.ErrPageNotFound) {
		writeAPIError(w, http.StatusNotFound, "page not found")
		return nil
	}
	if err != nil {
		return fmt.Errorf("load API page %q: %w", title, err)
	}

	if message := validateAPIPageOperation(request, stored); message != "" {
		writeAPIError(w, http.StatusBadRequest, message)
		return nil
	}

	if !admin {
		allowed, retryAfter := allowPageOperation(r, title, request.Operation)
		if !allowed {
			writeAPIRateLimitError(
				w,
				retryAfter,
				"page operation rate limit exceeded",
			)
			return nil
		}
	}

	update := pageUpdate{
		Operation: request.Operation,
		Password:  request.Password,
	}
	if request.Text != nil {
		update.Text = *request.Text
	}

	saved, err := applyAPIUpdate(r.Context(), title, update)
	if err != nil {
		status, message := apiPageOperationError(err)
		if status != 0 {
			writeAPIError(w, status, message)
			return nil
		}
		return err
	}

	broadcastExternalPageUpdate(title, saved, request.Operation)
	location := postedDocumentURL(r, title)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Location", location)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	return json.NewEncoder(w).Encode(apiPageOperationResponse{
		Title:        saved.Title,
		URL:          location,
		Operation:    request.Operation,
		Published:    saved.Published,
		SelfDestruct: saved.SelfDestruct,
		Locked:       saved.Locked,
		Encrypted:    hasValidEncryptedBlock(saved.Text),
	})
}

func validateAPIPageOperation(
	request apiPageOperationRequest,
	stored database.Page,
) string {
	switch request.Operation {
	case operationPublish,
		operationUnpublish,
		operationSelfDestruct,
		operationCancelSelfDestruct:
		if request.Password != "" || request.Text != nil {
			return "this operation does not accept password or text"
		}
	case operationLock:
		if request.Text != nil {
			return "lock operations do not accept text"
		}
		if request.Password == "" {
			return "lock operations require a password"
		}
		if len(request.Password) < minLockPasswordLen {
			return fmt.Sprintf(
				"page lock password must be at least %d characters",
				minLockPasswordLen,
			)
		}
		if len(request.Password) > maxLockPasswordLen {
			return fmt.Sprintf(
				"page lock password must not exceed %d bytes",
				maxLockPasswordLen,
			)
		}
	case operationUnlock:
		if request.Text != nil {
			return "lock operations do not accept text"
		}
		if request.Password == "" {
			return "lock operations require a password"
		}
	case operationEncrypt:
		if request.Password != "" {
			return "encryption passwords must stay on the client"
		}
		if request.Text == nil {
			return "encrypt requires client-encrypted text"
		}
		if len(*request.Text) > int(maxPostBodyBytes) {
			return "encrypted text exceeds the 16 KiB API limit"
		}
		if !validEncryptedDocument(*request.Text) {
			return "encrypt requires one valid COWYO ENCRYPTED BLOCK V1"
		}
	case operationDecrypt:
		if request.Password != "" {
			return "encryption passwords must stay on the client"
		}
		if request.Text == nil {
			return "decrypt requires client-decrypted text"
		}
		if len(*request.Text) > int(maxPostBodyBytes) {
			return "decrypted text exceeds the 16 KiB API limit"
		}
		if !hasValidEncryptedBlock(stored.Text) {
			return "page does not contain a valid encrypted block"
		}
	default:
		return "unsupported page operation"
	}
	return ""
}

func apiPageOperationError(err error) (int, string) {
	switch {
	case errors.Is(err, database.ErrPageNotFound):
		return http.StatusNotFound, "page not found"
	case errors.Is(err, errPageLocked):
		return http.StatusLocked, "page is locked"
	case errors.Is(err, errPageAlreadyLocked):
		return http.StatusConflict, "page is already locked"
	case errors.Is(err, errPageNotLocked):
		return http.StatusConflict, "page is not locked"
	case errors.Is(err, errWrongLockPassword):
		return http.StatusForbidden, "wrong page lock password"
	case errors.Is(err, errSelfDestructArmed):
		return http.StatusConflict, "cancel self destruct before publishing"
	case errors.Is(err, errDecryptChangedPlaintext):
		return http.StatusBadRequest, err.Error()
	default:
		return 0, ""
	}
}

func writeAPIRateLimitError(
	w http.ResponseWriter,
	retryAfter time.Duration,
	message string,
) {
	retryAfterSeconds := max(1, int(math.Ceil(retryAfter.Seconds())))
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	writeAPIError(w, http.StatusTooManyRequests, message)
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{Error: message})
}
