package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TODO more fields maybe?
type command struct {
	c           func(*chat, []string) error
	usage       string
	description string
}

// TODO fill out or remove if unnecessary
var commands = map[string]command{
	"/w":           {sendWhisper, "user message", ""},
	"/whisper":     {sendWhisper, "user message", ""},
	"/mute":        {sendMute, "user [duration in seconds]", ""},
	"/unmute":      {sendUnmute, "user", ""},
	"/highlight":   {addHighlight, "user", ""},
	"/unhighlight": {removeHighlight, "user", ""},
	"/tag":         {addTag, "user color", ""},
	"/untag":       {removeTag, "user", ""},
}

func (c *chat) handleCommand(message string) error {
	s := strings.Split(message, " ")

	f, ok := commands[s[0]]
	if ok {
		return f.c(c, s)
	}

	return fmt.Errorf("unknown command: %s", s[0])
}

func addHighlight(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /highlight user")
	}

	user := strings.ToLower(tokens[1])
	if contains(c.config.Highlighted, user) {
		return errors.New(user + " is already highlighted")
	}

	c.config.Lock()
	c.config.Highlighted = append(c.config.Highlighted, user)
	c.config.Unlock()

	return c.config.save()
}

func removeHighlight(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /unhighlight user")
	}

	user := strings.ToLower(tokens[1])
	c.config.Lock()
	defer c.config.Unlock()

	for i := 0; i < len(c.config.Highlighted); i++ {
		if strings.ToLower(c.config.Highlighted[i]) == user {
			c.config.Highlighted = append(c.config.Highlighted[:i], c.config.Highlighted[i+1:]...)
			return c.config.save()
		}
	}

	return errors.New("User: " + user + " is not in highlight list")
}

func addTag(c *chat, tokens []string) error {
	if len(tokens) < 3 {
		return errors.New("Usage: /tag user [Black, Red, Green, Yellow, Blue, Magenta, Cyan, White]")
	}

	color := strings.ToLower(tokens[2])
	user := strings.ToLower(tokens[1])

	_, ok := tagMap[color]
	if !ok {
		return errors.New("invalid color: " + color)
	}

	c.config.Lock()
	if c.config.Tags == nil {
		c.config.Tags = make(map[string]string)
	}
	c.config.Tags[user] = color
	c.config.Unlock()

	return c.config.save()
}

func removeTag(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /untag user")
	}

	user := strings.ToLower(tokens[1])

	c.config.Lock()
	defer c.config.Unlock()

	if _, ok := c.config.Tags[user]; ok {
		delete(c.config.Tags, user)
		return c.config.save()
	}

	return errors.New(user + " is not tagged")
}

func sendMute(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /mute user [duration in seconds]")
	}

	var err error
	duration := int64(60)

	if len(tokens) >= 3 {
		duration, err = strconv.ParseInt(strings.TrimSpace(tokens[2]), 10, 64)
		if err != nil {
			return err
		}
	}

	return c.Session.SendMute(tokens[1], time.Duration(duration)*time.Second)
}

func sendUnmute(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /mute user")
	}

	return c.Session.SendUnmute(tokens[1])
}

func sendWhisper(c *chat, tokens []string) error {
	if len(tokens) < 3 {
		return errors.New("Usage: /w user message")
	}

	nick := tokens[1]
	message := strings.Join(tokens[2:], " ")

	tm := time.Unix(time.Now().Unix()/1000, 0)
	tag := fmt.Sprintf(" %s*%s ", bgWhite, reset)
	msg := fmt.Sprintf("%s%s[PM -> %s] %s %s", tag, fgBrightWhite, nick, message, reset)
	c.Session.SendPrivateMessage(nick, message)
	c.renderFormattedMessage(msg, tm)
	return nil
}
