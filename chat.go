package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gorilla/websocket"
)

type chat struct {
	connection *websocket.Conn
	messages   chan chatMessage
}

type chatMessage struct {
	Nick      string   `json:"nick"`
	Features  []string `json:"features"`
	Timestamp int64    `json:"timestamp"`
	Data      string   `json:"data"`
}

var socketMessageRegex = regexp.MustCompile(`(\w+)\s(\{.+\})`)

func newChat(config *config) *chat {
	u := url.URL{Scheme: "wss", Host: "www.destiny.gg", Path: "/ws"}
	h := make(http.Header, 0)
	h.Set("Cookie", fmt.Sprintf("authtoken=%s", config.DGGKey))
	c, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		log.Fatalln(err)
	}

	chat := &chat{
		connection: c,
		messages:   make(chan chatMessage),
	}

	return chat
}

func (c *chat) listen() {
	for {
		_, message, err := c.connection.ReadMessage()
		if err != nil {
			log.Println(err)
		}

		m := string(message[:])

		match := socketMessageRegex.FindStringSubmatch(m)
		if len(match) != 3 {
			log.Printf("Unknown message format: '%s'\n", message)
			return
		}

		switch match[1] {
		case "MSG":
			var chatMessage chatMessage
			json.Unmarshal([]byte(match[2]), &chatMessage)

			c.messages <- chatMessage
		}
	}
}
