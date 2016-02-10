package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		type Message struct {
			TextData     string
			Title        string
			UpdateServer bool
			UpdateClient bool
		}

		var m Message
		err = json.Unmarshal(msg, &m)
		if err != nil {
			panic(err)
		}

		var p WikiData
		err = p.load(strings.ToLower(m.Title))
		if err != nil {
			panic(err)
		}
		if m.UpdateServer {
			err := p.save(m.TextData)
			if err != nil {
				panic(err)
			}
			m.TextData = "saved"
		}
		if m.UpdateClient {
			m.UpdateClient = len(m.TextData) != len(p.CurrentText)
			m.TextData = p.CurrentText
		}
		newMsg, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		conn.WriteMessage(t, newMsg)
	}
}
