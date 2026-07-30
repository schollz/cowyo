package cowyo

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/schollz/cowyo2/internal/database"
	log "github.com/schollz/logger"
)

const (
	websocketMessageEdit        = "edit"
	websocketMessageOperation   = "operation"
	websocketMessageCursor      = "cursor"
	websocketMessageCursorLeave = "cursor-leave"
	websocketMessageUpdate      = "update"
	websocketMessageAck         = "ack"
	websocketMessageError       = "error"

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
	Type         string `json:"type"`
	Text         string `json:"text"`
	CursorStart  int64  `json:"cursor_start"`
	CursorEnd    int64  `json:"cursor_end"`
	Published    bool   `json:"published"`
	SelfDestruct bool   `json:"self_destruct"`
	Locked       bool   `json:"locked"`
	Operation    string `json:"operation,omitempty"`
	Password     string `json:"password,omitempty"`
	Error        string `json:"error,omitempty"`
	Current      bool   `json:"current,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
}

type Connection struct {
	id          string
	conn        *websocket.Conn
	place       string
	send        chan websocketMessage
	done        chan struct{}
	closeOnce   sync.Once
	cursorStart int64
	cursorEnd   int64
	hasCursor   bool
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

	snapshots := registerConnection(connection)
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
		case websocketMessageCursor:
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
	saved, err := applyWebsocketUpdate(r.Context(), connection.place, update)
	if err != nil {
		log.Warnf("rejected update to %q: %s", connection.place, err)
		sendWebsocketError(connection, saved, update.Operation, err)
		return
	}

	broadcastPageUpdate(connection, saved, update)
	if !connection.enqueue(websocketMessage{
		Type:         websocketMessageAck,
		Text:         saved.Text,
		Published:    saved.Published,
		SelfDestruct: saved.SelfDestruct,
		Locked:       saved.Locked,
		Operation:    update.Operation,
	}) {
		connection.close()
	}
}

func sendWebsocketError(
	connection *Connection,
	saved database.Page,
	operation string,
	err error,
) {
	if !connection.enqueue(websocketMessage{
		Type:         websocketMessageError,
		Text:         saved.Text,
		Published:    saved.Published,
		SelfDestruct: saved.SelfDestruct,
		Locked:       saved.Locked,
		Operation:    operation,
		Error:        websocketErrorMessage(err),
		Current:      saved.Title != "",
	}) {
		connection.close()
	}
}

func registerConnection(current *Connection) []websocketMessage {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	snapshots := make([]websocketMessage, 0)
	for id, connection := range connections {
		if connection.place != current.place || !connection.hasCursor {
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
		Type:         websocketMessageUpdate,
		Text:         saved.Text,
		Published:    saved.Published,
		SelfDestruct: saved.SelfDestruct,
		Locked:       saved.Locked,
		Operation:    update.Operation,
		CursorStart:  update.CursorStart,
		CursorEnd:    update.CursorEnd,
		ClientID:     current.id,
	}

	connectionsMu.Lock()
	if connections[current.id] != current {
		connectionsMu.Unlock()
		return
	}
	current.cursorStart = update.CursorStart
	current.cursorEnd = update.CursorEnd
	current.hasCursor = true
	slow := enqueueForPlaceLocked(current.place, current.id, message)
	connectionsMu.Unlock()

	closeSlowConnections(slow)
}

func broadcastExternalPageUpdate(
	place string,
	saved database.Page,
	operation string,
) {
	message := websocketMessage{
		Type:         websocketMessageUpdate,
		Text:         saved.Text,
		Published:    saved.Published,
		SelfDestruct: saved.SelfDestruct,
		Locked:       saved.Locked,
		Operation:    operation,
		CursorStart:  saved.CursorStart,
		CursorEnd:    saved.CursorEnd,
	}

	connectionsMu.Lock()
	slow := enqueueForPlaceLocked(place, "", message)
	connectionsMu.Unlock()

	closeSlowConnections(slow)
}

func enqueueForPlaceLocked(
	place string,
	excludedClientID string,
	message websocketMessage,
) []*Connection {
	var slow []*Connection
	for id, connection := range connections {
		if id == excludedClientID || connection.place != place {
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
