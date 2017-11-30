package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/voloshink/dggchat"
)

const maxChatHistory = 10

type chat struct {
	config         *config
	gui            *gocui.Gui
	messageHistory []string
	historyIndex   int
	username       string
	Session        *dggchat.Session
}

func newChat(config *config, g *gocui.Gui) (*chat, error) {

	dgg, err := dggchat.New(config.DGGKey)
	if err != nil {
		return nil, err
	}

	if config.CustomURL != "" {
		url, err := url.Parse(config.CustomURL)
		if err != nil {
			return nil, err
		}
		dgg.SetURL(*url)
	}

	chat := &chat{
		config:         config,
		gui:            g,
		messageHistory: []string{},
		historyIndex:   -1,
		username:       config.Username,
		Session:        dgg,
	}

	return chat, nil
}

func (c *chat) handleInput(message string, g *gocui.Gui) {

	var err error

	//TODO cannot send message starting with "/"
	if message[:1] == "/" {
		err = c.handleCommand(message)
	} else {
		err = c.Session.SendMessage(message)
	}

	if err != nil {
		c.renderError(err.Error())
		return // TODO do we not want to append on error?
	}

	if len(c.messageHistory) > (maxChatHistory - 1) {
		c.messageHistory = append([]string{message}, c.messageHistory[:(maxChatHistory-1)]...)
	} else {
		c.messageHistory = append([]string{message}, c.messageHistory...)
	}
	c.historyIndex = -1
}

func (c *chat) handleCommand(message string) error {
	s := strings.Split(message, " ")

	//TODO make nickindex better; implement more commands

	switch s[0] {
	case "/w", "/whisper": //TODO chat frontend defines more of those
		if len(s) < 3 {
			return errors.New("Usage: /w user message")
		}
		nickindex := strings.Index(message, s[1])
		err := c.SendPrivateMessage(s[1], message[nickindex+len(s[1]):])
		if err != nil {
			return err
		}

	case "/mute":
		if len(s) != 3 { //TODO duration is optional
			return errors.New("Usage: /mute user [duration in seconds]")
		}
		nickindex := strings.Index(message, s[1])
		duration, err := strconv.ParseInt(message[nickindex+1+len(s[1]):], 10, 64)
		if err != nil {
			return err
		}
		err = c.Session.SendMute(s[1], time.Duration(duration)*time.Second)
		if err != nil {
			return err
		}
	case "/highlight":
		// TODO add user to config and save
		if len(s) < 2 {
			return errors.New("Usage: /highlight user")
		}

		return c.addHighlight(s[1])

	case "/unhighlight":
		// TODO add user to config and save
		if len(s) < 2 {
			return errors.New("Usage: /unhighlight user")
		}

		return c.removeHighlight(s[1])

	default:
		return nil //TODO
	}
	return nil //TODO
}

func (c *chat) SendPrivateMessage(nick string, message string) error {

	err := c.Session.SendPrivateMessage(nick, message)
	if err != nil {
		return err
	}

	c.gui.Update(func(g *gocui.Gui) error {
		messagesView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		tm := time.Unix(time.Now().Unix()/1000, 0)
		formattedDate := tm.Format(time.Kitchen)

		formattedMessage := fmt.Sprintf("[%s]  \u001b[37;1m\u001b[1m[Whisper]%s: %s %s %s", formattedDate, c.username, nick, message, colorReset)

		fmt.Fprintln(messagesView, formattedMessage)
		return nil
	})
	return err
}

func (c *chat) addHighlight(user string) error {

	if contains(c.config.Highlighted, user) {
		return errors.New(user + " is already highlighted")
	}

	c.config.Lock()
	c.config.Highlighted = append(c.config.Highlighted, user)
	c.config.Unlock()

	return c.config.save()
}

func (c *chat) removeHighlight(user string) error {

	c.config.Lock()
	defer c.config.Unlock()

	for i := 0; i < len(c.config.Highlighted); i++ {
		if strings.ToLower(c.config.Highlighted[i]) == strings.ToLower(user) {
			c.config.Highlighted = append(c.config.Highlighted[:i], c.config.Highlighted[i+1:]...)
			return c.config.save()
		}
	}

	return errors.New("User: " + user + " is not in highlight list")
}
