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

const maxChatHistory = 10

type chat struct {
	connection     *websocket.Conn
	g              *gocui.Gui
	userList       *userList
	messageHistory []string
	historyIndex   int
	username       string
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

func newChat(config *config, g *gocui.Gui) (*chat, error) {
	var u string
	if config.CustomURL == "" {
		url := url.URL{Scheme: "wss", Host: "www.destiny.gg", Path: "/ws"}
		u = url.String()
	} else {
		u = config.CustomURL
	}

	h := http.Header{}
	h.Set("Cookie", fmt.Sprintf("authtoken=%s", config.DGGKey))
	c, _, err := websocket.DefaultDialer.Dial(u, h)
	if err != nil {
		return &chat{}, err
	}

	chat := &chat{
		connection:     c,
		g:              g,
		messageHistory: []string{},
		historyIndex:   -1,
		username:       config.Username,
	}

	return chat, nil
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
			return
		}

		switch match[1] {
		case "NAMES":
			var userList userList
			json.Unmarshal([]byte(match[2]), &userList)
			c.userList = &userList
			c.renderUsers(&userList)
		case "MSG":
			var chatMessage chatMessage
			json.Unmarshal([]byte(match[2]), &chatMessage)

			c.renderMessage(&chatMessage)
		case "ERR":
			c.renderError(match[2])
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
				c.renderUsers(c.userList)
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
				c.renderUsers(c.userList)
			}

		case "BROADCAST":
			var broadcastMessage broadcastMessage
			json.Unmarshal([]byte(match[2]), &broadcastMessage)

			c.renderBroadcast(&broadcastMessage)
		}

	}
}

func (c *chat) sendMessage(message string, g *gocui.Gui) {
	// TODO commands
	jsonMessage := fmt.Sprintf("MSG {\"data\":\"%s\"}", message)
	err := c.connection.WriteMessage(websocket.TextMessage, []byte(jsonMessage))
	if err != nil {
		c.renderError(err.Error())
		return
	}
	if len(c.messageHistory) > (maxChatHistory - 1) {
		c.messageHistory = append([]string{message}, c.messageHistory[:(maxChatHistory-1)]...)
	} else {
		c.messageHistory = append([]string{message}, c.messageHistory...)
	}
	c.historyIndex = -1
}
