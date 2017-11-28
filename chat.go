package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jroimartin/gocui"
)

const maxChatHistory = 10

type chat struct {
	config         *config
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
	chat := &chat{
		config:         config,
		g:              g,
		messageHistory: []string{},
		historyIndex:   -1,
		username:       config.Username,
	}

	err := chat.connect()
	return chat, err
}

func (c *chat) connect() error {
	var u string
	if c.config.CustomURL == "" {
		url := url.URL{Scheme: "wss", Host: "www.destiny.gg", Path: "/ws"}
		u = url.String()
	} else {
		u = c.config.CustomURL
	}

	h := http.Header{}
	h.Set("Cookie", fmt.Sprintf("authtoken=%s", c.config.DGGKey))
	var err error
	c.connection, _, err = websocket.DefaultDialer.Dial(u, h)
	return err
}

func (c *chat) reconnect() {
	var timeout = 2
	for {
		c.renderError(fmt.Sprintf("reconnecting in %d seconds...", timeout))
		time.Sleep(time.Second * time.Duration(timeout))
		err := c.connect()
		if err != nil {
			c.renderError(fmt.Sprintf("failed establishing connection to the chat"))
			if timeout > 60 {
				timeout = 1
			}
			timeout = timeout * 2
			continue
		}
		return
	}
}

func (c *chat) listen() {
	defer c.connection.Close()
	for {
		err := c.connection.SetReadDeadline(time.Now().Add(time.Minute * 3))
		if err != nil {
			c.renderError("exceeded ReadDeadline")
			c.reconnect()
			time.Sleep(3 * time.Second)
			continue
		}
		_, message, err := c.connection.ReadMessage()
		if err != nil {
			c.renderError("error getting message")
			c.reconnect()
			time.Sleep(3 * time.Second)
			continue
		}

		m := string(message[:])

		match := socketMessageRegex.FindStringSubmatch(m)
		if len(match) != 3 {
			continue
		}

		switch match[1] {
		case "NAMES":
			var userList userList
			json.Unmarshal([]byte(match[2]), &userList)
			c.userList = &userList
			c.renderUsers(userList)
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
				c.renderUsers(*c.userList)
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
				c.renderUsers(*c.userList)
			}

		case "BROADCAST":
			var broadcastMessage broadcastMessage
			json.Unmarshal([]byte(match[2]), &broadcastMessage)

			c.renderBroadcast(&broadcastMessage)
		}

	}
}

func (c *chat) sendMessage(message string, g *gocui.Gui) {
	err := c.connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		c.renderError(err.Error())
		c.reconnect()
		return
	}

	// TODO commands
	jsonMessage := fmt.Sprintf("MSG {\"data\":\"%s\"}", message)
	err = c.connection.WriteMessage(websocket.TextMessage, []byte(jsonMessage))
	if err != nil {
		c.renderError(err.Error())
		c.reconnect()
		return
	}
	if len(c.messageHistory) > (maxChatHistory - 1) {
		c.messageHistory = append([]string{message}, c.messageHistory[:(maxChatHistory-1)]...)
	} else {
		c.messageHistory = append([]string{message}, c.messageHistory...)
	}
	c.historyIndex = -1
}
