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

		if m.UpdateServer {
			p := CowyoData{strings.ToLower(m.Title), m.TextData}
			err := p.save()
			if err != nil {
				panic(err)
			}
		}
		if m.UpdateClient {
			p := CowyoData{strings.ToLower(m.Title), ""}
			err := p.load()
			if err != nil {
				panic(err)
			}
			m.UpdateClient = len(m.TextData) != len(p.Text)
			m.TextData = p.Text
		}
		newMsg, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		conn.WriteMessage(t, newMsg)
	}
}
