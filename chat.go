package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/jroimartin/gocui"
)

type chat struct {
	connection *websocket.Conn
	g          *gocui.Gui
	userList   *userList
}

type userList struct {
	Count int    `json:"connectioncount"`
	Users []user `json:"users"`
}

type user struct {
	Nick     string   `json:"nick"`
	Features []string `json:"features"`
}

type chatMessage struct {
	Nick      string   `json:"nick"`
	Features  []string `json:"features"`
	Timestamp int64    `json:"timestamp"`
	Data      string   `json:"data"`
}

type broadcastMessage struct {
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"`
}

var socketMessageRegex = regexp.MustCompile(`(\w+)\s(.+)`)

func newChat(config *config, g *gocui.Gui) *chat {
	u := url.URL{Scheme: "wss", Host: "www.destiny.gg", Path: "/ws"}
	h := make(http.Header, 0)
	h.Set("Cookie", fmt.Sprintf("authtoken=%s", config.DGGKey))
	c, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		log.Fatalln(err)
	}

	chat := &chat{
		connection: c,
		g:          g,
	}

	return chat
}

func (c *chat) listen() {
	defer c.connection.Close()
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
		case "NAMES":
			var userList userList
			json.Unmarshal([]byte(match[2]), &userList)
			c.userList = &userList
			renderUsers(c.g, &userList)
		case "MSG":
			var chatMessage chatMessage
			json.Unmarshal([]byte(match[2]), &chatMessage)

			renderMessage(c.g, &chatMessage)
		case "ERR":
			renderError(c.g, match[2])
		case "QUIT":
			var quitter user
			json.Unmarshal([]byte(match[2]), &quitter)

			index := -1
			for i, u := range c.userList.Users {
				if strings.EqualFold(quitter.Nick, u.Nick) {
					index = i
				}
			}

			if index > -1 {
				c.userList.Users = append(c.userList.Users[:index], c.userList.Users[index+1:]...)
				c.userList.Count--
				renderUsers(c.g, c.userList)
			}

		case "JOIN":
			var joiner user
			json.Unmarshal([]byte(match[2]), &joiner)

			index := -1
			for i, u := range c.userList.Users {
				if strings.EqualFold(joiner.Nick, u.Nick) {
					index = i
				}
			}

			if index == -1 {
				c.userList.Users = append(c.userList.Users, joiner)
				c.userList.Count++
				renderUsers(c.g, c.userList)
			}

		case "BROADCAST":
			// handle broadcast
		}

	}
}

func (c *chat) sendMessage(message string) {
	message = fmt.Sprintf("MSG {\"data\":\"%s\"}", message)
	err := c.connection.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println(err)
	}
}
