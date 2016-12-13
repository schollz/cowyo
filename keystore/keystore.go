package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

func get(key string) interface{} {
	c, _, err := websocket.DefaultDialer.Dial("wss://cowyo.com/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	done := make(chan struct{})
	var value interface{}
	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				// log.Println("read:", err)
				return
			}
			var m struct {
				TextData string
			}
			json.Unmarshal([]byte(message), &m)
			json.Unmarshal([]byte(m.TextData), &value)
			done <- struct{}{}
		}
	}()
	// ask for something
	c.WriteMessage(websocket.TextMessage, []byte(`{"Title":"`+key+`", "UpdateClient":true}`))
	<-done
	return value
}

func set(key string, message interface{}) interface{} {
	c, _, err := websocket.DefaultDialer.Dial("wss://cowyo.com/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	done := make(chan struct{})
	var value interface{}
	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				// log.Println("read:", err)
				return
			}
			var m struct {
				TextData string
			}
			json.Unmarshal([]byte(message), &m)
			json.Unmarshal([]byte(m.TextData), &value)
			done <- struct{}{}
		}
	}()
	// ask for something
	bJson, _ := json.Marshal(message)
	aa, _ := json.Marshal(map[string]interface{}{
		"Title":        key,
		"UpdateServer": true,
		"TextData":     string(bJson),
	})
	c.WriteMessage(websocket.TextMessage, aa)
	<-done
	return value
}

func main() {
	fmt.Println(get("data"))
	fmt.Println(get("data"))
	m := make(map[string]int)
	m["some string"] = 29
	fmt.Println(set("data2", m))
	fmt.Println(get("data2"))
}
