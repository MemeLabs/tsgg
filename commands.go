package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

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
		if len(s) < 2 {
			return errors.New("Usage: /highlight user")
		}

		return c.addHighlight(s[1])

	case "/unhighlight":
		if len(s) < 2 {
			return errors.New("Usage: /unhighlight user")
		}

		return c.removeHighlight(s[1])

	case "/tag":
		if len(s) < 3 {
			return errors.New("Usage: /tag user [Red, Green, Yellow, Blue, Magenta, Cyan]")
		}

		return c.addTag(s[1], s[2])

	case "/untag":
		if len(s) < 2 {
			return errors.New("Usage: /untag user")
		}

		return c.removeTag(s[1])

	default:
		return nil //TODO
	}
	return nil //TODO
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

func (c *chat) addTag(user, color string) error {
	color = strings.ToLower(color)
	user = strings.ToLower(user)

	_, ok := backgrounds[color]
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

func (c *chat) removeTag(user string) error {
	user = strings.ToLower(user)

	c.config.Lock()
	defer c.config.Unlock()

	if _, ok := c.config.Tags[user]; ok {
		delete(c.config.Tags, user)
		return c.config.save()
	}

	return errors.New(user + " is not tagged")
}
