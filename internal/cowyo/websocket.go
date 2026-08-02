package cowyo

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/schollz/cowyo2/internal/database"
	log "github.com/schollz/logger"
)

const (
	websocketMessageEdit              = "edit"
	websocketMessageOperation         = "operation"
	websocketMessageCursor            = "cursor"
	websocketMessageCursorLeave       = "cursor-leave"
	websocketMessageUpdate            = "update"
	websocketMessageAck               = "ack"
	websocketMessageError             = "error"
	websocketMessageE2EECreate        = "e2ee-create"
	websocketMessageE2EEConvert       = "e2ee-convert"
	websocketMessageE2EEAuthenticate  = "e2ee-authenticate"
	websocketMessageE2EEAuthenticated = "e2ee-authenticated"

	operationLock               = "lock"
	operationUnlock             = "unlock"
	operationEncrypt            = "encrypt"
	operationDecrypt            = "decrypt"
	operationPublish            = "publish"
	operationUnpublish          = "unpublish"
	operationSelfDestruct       = "self-destruct"
	operationCancelSelfDestruct = "cancel-self-destruct"

	websocketSendQueueSize      = 64
	maxWebsocketMessageSize     = 64 << 10
	websocketWriteTimeout       = 5 * time.Second
	minimumCursorUpdateInterval = 25 * time.Millisecond
)

type websocketMessage struct {
	Type              string `json:"type"`
	Text              string `json:"text"`
	CursorStart       int64  `json:"cursor_start"`
	CursorEnd         int64  `json:"cursor_end"`
	Published         bool   `json:"published"`
	SelfDestruct      bool   `json:"self_destruct"`
	Locked            bool   `json:"locked"`
	EndToEndEncrypted bool   `json:"end_to_end_encrypted"`
	Operation         string `json:"operation,omitempty"`
	Password          string `json:"password,omitempty"`
	Capability        string `json:"write_capability,omitempty"`
	Error             string `json:"error,omitempty"`
	ErrorCode         string `json:"error_code,omitempty"`
	Current           bool   `json:"current,omitempty"`
	Final             bool   `json:"final,omitempty"`
	ClientID          string `json:"client_id,omitempty"`
}

type Connection struct {
	id                string
	conn              *websocket.Conn
	place             string
	send              chan websocketMessage
	done              chan struct{}
	closeOnce         sync.Once
	cursorStart       int64
	cursorEnd         int64
	hasCursor         bool
	e2eePage          atomic.Bool
	e2eeAuthenticated atomic.Bool
	e2eeConsumed      atomic.Bool
}

var upgrader = websocket.Upgrader{}

func handleWebsocket(w http.ResponseWriter, r *http.Request) (err error) {
	place := r.URL.Query().Get("place")
	if place == "" {
		return fmt.Errorf("no place")
	}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	socket.SetReadLimit(maxWebsocketMessageSize)

	clientID, err := newWebsocketClientID()
	if err != nil {
		socket.Close()
		return fmt.Errorf("create websocket client ID: %w", err)
	}
	connection := &Connection{
		id:    clientID,
		conn:  socket,
		place: place,
		send:  make(chan websocketMessage, websocketSendQueueSize),
		done:  make(chan struct{}),
	}
	go connection.writePump()

	pageMutationMu.Lock()
	stored, loadErr := pageStore.GetPage(r.Context(), place)
	if loadErr != nil && !errors.Is(loadErr, database.ErrPageNotFound) {
		pageMutationMu.Unlock()
		connection.close()
		return fmt.Errorf("load websocket page: %w", loadErr)
	}
	connection.e2eePage.Store(loadErr == nil && stored.EndToEndEncrypted)
	snapshots := registerConnection(connection)
	pageMutationMu.Unlock()
	defer unregisterConnection(connection)
	for _, snapshot := range snapshots {
		if !connection.enqueue(snapshot) {
			connection.close()
			return nil
		}
	}

	var lastCursorUpdate time.Time
	for {
		var message websocketMessage
		if err := socket.ReadJSON(&message); err != nil {
			break
		}

		switch message.Type {
		case websocketMessageE2EEAuthenticate:
			handleE2EEAuthentication(r, connection, message)
		case websocketMessageE2EECreate, websocketMessageE2EEConvert:
			if !validCursorRange(message.CursorStart, message.CursorEnd) {
				sendWebsocketError(connection, database.Page{}, message.Type, fmt.Errorf("invalid cursor range"))
				continue
			}
			handleE2EEBootstrap(r, connection, message)
		case websocketMessageCursor:
			if connection.e2eeConsumed.Load() {
				sendWebsocketError(connection, database.Page{}, "", database.ErrPageNotFound)
				continue
			}
			if connection.e2eePage.Load() && !connection.e2eeAuthenticated.Load() {
				sendWebsocketError(connection, database.Page{}, "", errE2EEAuthenticationRequired)
				continue
			}
			if !validCursorRange(message.CursorStart, message.CursorEnd) {
				continue
			}
			now := time.Now()
			if !lastCursorUpdate.IsZero() &&
				now.Sub(lastCursorUpdate) < minimumCursorUpdateInterval {
				continue
			}
			lastCursorUpdate = now
			updateConnectionCursor(connection, message.CursorStart, message.CursorEnd)
		case websocketMessageEdit, websocketMessageOperation:
			if message.Type == websocketMessageEdit {
				message.Operation = ""
			} else if message.Operation == "" {
				sendWebsocketError(connection, database.Page{}, message.Operation, fmt.Errorf("unsupported page operation"))
				continue
			}
			if !validCursorRange(message.CursorStart, message.CursorEnd) {
				sendWebsocketError(connection, database.Page{}, message.Operation, fmt.Errorf("invalid cursor range"))
				continue
			}
			applyWebsocketMessage(r, connection, message)
		default:
			sendWebsocketError(connection, database.Page{}, message.Operation, fmt.Errorf("unsupported websocket message type"))
		}
	}

	return nil
}

func applyWebsocketMessage(r *http.Request, connection *Connection, message websocketMessage) {
	if message.Type == websocketMessageOperation {
		allowed, retryAfter := allowPageOperation(
			r,
			connection.place,
			message.Operation,
		)
		if !allowed {
			sendWebsocketError(
				connection,
				database.Page{},
				message.Operation,
				fmt.Errorf(
					"page operation rate limit exceeded; retry in %s",
					retryAfter.Round(time.Second),
				),
			)
			return
		}
	}

	update := pageUpdate{
		Text:        message.Text,
		CursorStart: message.CursorStart,
		CursorEnd:   message.CursorEnd,
		Operation:   message.Operation,
		Password:    message.Password,
	}
	log.Tracef(
		"updating '%s' with operation %q and %d bytes",
		connection.place,
		update.Operation,
		len(update.Text),
	)
	if connection.e2eeConsumed.Load() {
		sendWebsocketError(connection, database.Page{}, update.Operation, database.ErrPageNotFound)
		return
	}
	saved, err := applyAuthenticatedWebsocketUpdate(
		r.Context(),
		connection.place,
		update,
		connection.e2eeAuthenticated.Load(),
	)
	if err != nil {
		log.Warnf("rejected update to %q: %s", connection.place, err)
		if saved.EndToEndEncrypted && !connection.e2eeAuthenticated.Load() {
			saved.Text = ""
		}
		sendWebsocketError(connection, saved, update.Operation, err)
		return
	}

	broadcastPageUpdate(connection, saved, update)
	if !connection.enqueue(websocketMessage{
		Type:              websocketMessageAck,
		Text:              saved.Text,
		Published:         saved.Published,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: saved.EndToEndEncrypted,
		Operation:         update.Operation,
	}) {
		connection.close()
	}
}

func handleE2EEBootstrap(
	r *http.Request,
	connection *Connection,
	message websocketMessage,
) {
	if message.Type == websocketMessageE2EECreate {
		allowed, retryAfter := postingLimiter.Allow(postClientKey(r))
		if !allowed {
			sendWebsocketError(
				connection,
				database.Page{},
				message.Type,
				fmt.Errorf(
					"private page creation rate limit exceeded; retry in %s",
					retryAfter.Round(time.Second),
				),
			)
			return
		}
	} else if message.Type == websocketMessageE2EEConvert {
		allowed, retryAfter := allowPageOperation(r, connection.place, message.Type)
		if !allowed {
			sendWebsocketError(
				connection,
				database.Page{},
				message.Type,
				fmt.Errorf(
					"page operation rate limit exceeded; retry in %s",
					retryAfter.Round(time.Second),
				),
			)
			return
		}
	}
	if !validE2EEDocument(message.Text) {
		sendWebsocketError(connection, database.Page{}, message.Type, errE2EEInvalidDocument)
		return
	}
	authHash, err := e2eeCapabilityHash(message.Capability)
	if err != nil {
		sendWebsocketError(connection, database.Page{}, message.Type, err)
		return
	}

	next := database.Page{
		Title:             connection.place,
		Text:              message.Text,
		CursorStart:       message.CursorStart,
		CursorEnd:         message.CursorEnd,
		EndToEndEncrypted: true,
		E2EEAuthHash:      authHash,
	}

	pageMutationMu.Lock()
	var saved database.Page
	switch message.Type {
	case websocketMessageE2EECreate:
		var created bool
		created, err = pageStore.CreatePage(r.Context(), next)
		if err == nil && !created {
			err = errE2EEPageCollision
		}
		if err == nil {
			saved = next
		}
	case websocketMessageE2EEConvert:
		var converted bool
		converted, err = pageStore.ConvertPageToE2EE(r.Context(), next)
		if err == nil && !converted {
			err = errE2EEConversionUnavailable
		}
		if err == nil {
			saved, err = pageStore.GetPage(r.Context(), connection.place)
		}
	default:
		err = errors.New("unsupported private page bootstrap")
	}
	pageMutationMu.Unlock()
	if err != nil {
		loaded, loadErr := pageStore.GetPage(r.Context(), connection.place)
		current := database.Page{}
		if loadErr == nil {
			current = database.Page{
				Title:             loaded.Title,
				Published:         loaded.Published,
				SelfDestruct:      loaded.SelfDestruct,
				Locked:            loaded.Locked,
				EndToEndEncrypted: loaded.EndToEndEncrypted,
			}
		}
		sendWebsocketError(connection, current, message.Type, err)
		return
	}

	connection.e2eePage.Store(true)
	connection.e2eeAuthenticated.Store(true)
	logPageEdit(
		connection.place,
		"websocket",
		message.Type,
		len(saved.Text),
	)
	broadcastE2EETransition(connection, saved, message.Type)
	if !connection.enqueue(websocketMessage{
		Type:              websocketMessageE2EEAuthenticated,
		Text:              saved.Text,
		CursorStart:       saved.CursorStart,
		CursorEnd:         saved.CursorEnd,
		Published:         false,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: true,
		Operation:         message.Type,
	}) {
		connection.close()
	}
}

func handleE2EEAuthentication(
	r *http.Request,
	connection *Connection,
	message websocketMessage,
) {
	pageMutationMu.Lock()
	saved, err := pageStore.GetPage(r.Context(), connection.place)
	if err == nil && !saved.EndToEndEncrypted {
		err = errE2EEAuthenticationRequired
	}
	if err == nil {
		err = verifyE2EECapability(saved.E2EEAuthHash, message.Capability)
	}
	final := false
	if err == nil && saved.SelfDestruct {
		saved, err = pageStore.ConsumeE2EEPage(r.Context(), connection.place)
		final = err == nil
	}
	pageMutationMu.Unlock()
	if err != nil {
		sendWebsocketError(connection, database.Page{}, message.Type, err)
		return
	}

	connection.e2eePage.Store(true)
	connection.e2eeAuthenticated.Store(true)
	connection.e2eeConsumed.Store(final)
	if final {
		logPageEdit(
			connection.place,
			"websocket",
			operationSelfDestruct,
			len(saved.Text),
		)
	}
	if !connection.enqueue(websocketMessage{
		Type:              websocketMessageE2EEAuthenticated,
		Text:              saved.Text,
		CursorStart:       saved.CursorStart,
		CursorEnd:         saved.CursorEnd,
		Published:         false,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: true,
		Final:             final,
	}) {
		connection.close()
		return
	}
	if !final {
		enqueueAuthenticatedCursorSnapshots(connection)
	}
}

func enqueueAuthenticatedCursorSnapshots(current *Connection) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	for id, connection := range connections {
		if id == current.id ||
			connection.place != current.place ||
			!connection.hasCursor ||
			!connection.e2eeAuthenticated.Load() {
			continue
		}
		if !current.enqueue(websocketMessage{
			Type:        websocketMessageCursor,
			CursorStart: connection.cursorStart,
			CursorEnd:   connection.cursorEnd,
			ClientID:    id,
		}) {
			current.close()
			return
		}
	}
}

func broadcastE2EETransition(
	current *Connection,
	saved database.Page,
	operation string,
) {
	message := websocketMessage{
		Type:              websocketMessageUpdate,
		Text:              saved.Text,
		CursorStart:       saved.CursorStart,
		CursorEnd:         saved.CursorEnd,
		Published:         false,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: true,
		Operation:         operation,
		ClientID:          current.id,
	}

	connectionsMu.Lock()
	var slow []*Connection
	for id, connection := range connections {
		if id == current.id || connection.place != current.place {
			continue
		}
		connection.e2eePage.Store(true)
		connection.e2eeAuthenticated.Store(false)
		if !connection.enqueue(message) {
			slow = append(slow, connection)
		}
	}
	connectionsMu.Unlock()
	closeSlowConnections(slow)
}

func sendWebsocketError(
	connection *Connection,
	saved database.Page,
	operation string,
	err error,
) {
	if !connection.enqueue(websocketMessage{
		Type:              websocketMessageError,
		Text:              saved.Text,
		Published:         saved.Published,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: saved.EndToEndEncrypted,
		Operation:         operation,
		Error:             websocketErrorMessage(err),
		ErrorCode:         websocketErrorCode(err),
		Current:           saved.Title != "",
	}) {
		connection.close()
	}
}

func websocketErrorCode(err error) string {
	if errors.Is(err, errE2EEPageCollision) {
		return "page-name-collision"
	}
	return ""
}

func registerConnection(current *Connection) []websocketMessage {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	snapshots := make([]websocketMessage, 0)
	if current.e2eePage.Load() && !current.e2eeAuthenticated.Load() {
		connections[current.id] = current
		return snapshots
	}
	for id, connection := range connections {
		if connection.place != current.place || !connection.hasCursor {
			continue
		}
		if current.e2eePage.Load() && !connection.e2eeAuthenticated.Load() {
			continue
		}
		snapshots = append(snapshots, websocketMessage{
			Type:        websocketMessageCursor,
			CursorStart: connection.cursorStart,
			CursorEnd:   connection.cursorEnd,
			ClientID:    id,
		})
	}
	connections[current.id] = current
	return snapshots
}

func unregisterConnection(current *Connection) {
	connectionsMu.Lock()
	if connections[current.id] == current {
		delete(connections, current.id)
	}
	slow := enqueueForPlaceLocked(
		current.place,
		current.id,
		websocketMessage{
			Type:     websocketMessageCursorLeave,
			ClientID: current.id,
		},
		current.e2eePage.Load(),
	)
	connectionsMu.Unlock()

	current.close()
	closeSlowConnections(slow)
}

func updateConnectionCursor(current *Connection, cursorStart, cursorEnd int64) {
	connectionsMu.Lock()
	if connections[current.id] != current {
		connectionsMu.Unlock()
		return
	}
	current.cursorStart = cursorStart
	current.cursorEnd = cursorEnd
	current.hasCursor = true
	slow := enqueueForPlaceLocked(
		current.place,
		current.id,
		websocketMessage{
			Type:        websocketMessageCursor,
			CursorStart: cursorStart,
			CursorEnd:   cursorEnd,
			ClientID:    current.id,
		},
		current.e2eePage.Load(),
	)
	connectionsMu.Unlock()

	closeSlowConnections(slow)
}

func broadcastPageUpdate(
	current *Connection,
	saved database.Page,
	update pageUpdate,
) {
	message := websocketMessage{
		Type:              websocketMessageUpdate,
		Text:              saved.Text,
		Published:         saved.Published,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: saved.EndToEndEncrypted,
		Operation:         update.Operation,
		CursorStart:       update.CursorStart,
		CursorEnd:         update.CursorEnd,
		ClientID:          current.id,
	}

	connectionsMu.Lock()
	if connections[current.id] != current {
		connectionsMu.Unlock()
		return
	}
	current.cursorStart = update.CursorStart
	current.cursorEnd = update.CursorEnd
	current.hasCursor = true
	slow := enqueueForPlaceLocked(
		current.place,
		current.id,
		message,
		saved.EndToEndEncrypted,
	)
	connectionsMu.Unlock()

	closeSlowConnections(slow)
}

func broadcastExternalPageUpdate(
	place string,
	saved database.Page,
	operation string,
) {
	message := websocketMessage{
		Type:              websocketMessageUpdate,
		Text:              saved.Text,
		Published:         saved.Published,
		SelfDestruct:      saved.SelfDestruct,
		Locked:            saved.Locked,
		EndToEndEncrypted: saved.EndToEndEncrypted,
		Operation:         operation,
		CursorStart:       saved.CursorStart,
		CursorEnd:         saved.CursorEnd,
	}

	connectionsMu.Lock()
	slow := enqueueForPlaceLocked(place, "", message, saved.EndToEndEncrypted)
	connectionsMu.Unlock()

	closeSlowConnections(slow)
}

func enqueueForPlaceLocked(
	place string,
	excludedClientID string,
	message websocketMessage,
	e2eeOnly bool,
) []*Connection {
	var slow []*Connection
	for id, connection := range connections {
		if id == excludedClientID || connection.place != place {
			continue
		}
		if e2eeOnly && !connection.e2eeAuthenticated.Load() {
			continue
		}
		if !connection.enqueue(message) {
			slow = append(slow, connection)
		}
	}
	return slow
}

func closeSlowConnections(connections []*Connection) {
	for _, connection := range connections {
		connection.close()
	}
}

func (connection *Connection) enqueue(message websocketMessage) bool {
	select {
	case <-connection.done:
		return false
	default:
	}

	select {
	case connection.send <- message:
		return true
	case <-connection.done:
		return false
	default:
		return false
	}
}

func (connection *Connection) writePump() {
	for {
		select {
		case message := <-connection.send:
			if err := connection.conn.SetWriteDeadline(
				time.Now().Add(websocketWriteTimeout),
			); err != nil {
				connection.close()
				return
			}
			if err := connection.conn.WriteJSON(message); err != nil {
				connection.close()
				return
			}
		case <-connection.done:
			return
		}
	}
}

func (connection *Connection) close() {
	connection.closeOnce.Do(func() {
		close(connection.done)
		if err := connection.conn.Close(); err != nil {
			log.Tracef("closing websocket connection: %s", err)
		}
	})
}

func validCursorRange(cursorStart, cursorEnd int64) bool {
	return cursorStart >= 0 &&
		cursorEnd >= cursorStart &&
		cursorEnd <= maxWebsocketMessageSize
}

func newWebsocketClientID() (string, error) {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}
