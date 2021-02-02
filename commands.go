package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type command struct {
	c          func(*chat, []string) error
	usage      string
	privileged bool
}

// TODO need to refactor this... usage strings incomplete/double
var commands = map[string]command{
	"/w":           {sendWhisper, "user message", false},
	"/whisper":     {sendWhisper, "user message", false},
	"/me":          {sendAction, "message", false},
	"/tag":         {addTag, "user color", false},
	"/untag":       {removeTag, "user", false},
	"/highlight":   {addHighlight, "user", false},
	"/unhighlight": {removeHighlight, "user", false},
	"/ignore":      {addIgnore, "user", false},
	"/unignore":    {removeIgnore, "user", false},
	"/stalk":       {addStalk, "user", false},
	"/unstalk":     {removeStalk, "user", false},
	"/mute":        {sendMute, "user [time (in seconds)]", true},
	"/unmute":      {sendUnmute, "user", true},
	// TODO reason is forced to be single string here without good reason.
	"/ban":       {sendBan, "user reason [time (in seconds)]", true},
	"/ipban":     {sendBan, "user reason [time (in seconds)]", true},
	"/perm":      {sendPermBan, "user reason", true},
	"/permip":    {sendPermBan, "user reason", true},
	"/unban":     {sendUnban, "user", true},
	"/subonly":   {sendSubOnly, "{on,off}", true},
	"/broadcast": {sendBroadcast, "message", true},
}

// translate user tags into colors...
var tagMap = map[string]color{
	"black":   bgBlack,
	"red":     bgRed,
	"green":   bgGreen,
	"yellow":  bgYellow,
	"blue":    bgBlue,
	"magenta": bgMagenta,
	"cyan":    bgCyan,
	"white":   bgWhite,
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
		return errors.New("usage: /highlight user")
	}

	user := strings.ToLower(tokens[1])
	if contains(c.config.Highlighted, user) {
		return fmt.Errorf("%s is already highlighted", user)
	}

	c.config.Lock()
	c.config.Highlighted = append(c.config.Highlighted, user)
	c.config.Unlock()

	err := c.config.save()
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Highlighted %s", user)
	c.renderCommand(msg)
	return nil
}

func removeHighlight(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("usage: /unhighlight user")
	}

	user := strings.ToLower(tokens[1])
	c.config.Lock()
	defer c.config.Unlock()

	for i := 0; i < len(c.config.Highlighted); i++ {
		if strings.ToLower(c.config.Highlighted[i]) == user {
			c.config.Highlighted = append(c.config.Highlighted[:i], c.config.Highlighted[i+1:]...)
			err := c.config.save()
			if err != nil {
				return err
			}
			msg := fmt.Sprintf("Unhighlighted %s", user)
			c.renderCommand(msg)
			return nil
		}
	}

	return fmt.Errorf("%s is not in highlight list", user)
}

func addStalk(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("usage: /stalk user")
	}

	user := strings.ToLower(tokens[1])
	if contains(c.config.Stalks, user) {
		return fmt.Errorf("already stalking %s", user)
	}

	c.config.Lock()
	c.config.Stalks = append(c.config.Stalks, user)
	c.config.Unlock()

	err := c.config.save()
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Now stalking %s", user)
	c.renderCommand(msg)
	return nil
}

func removeStalk(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("usage: /unstalk user")
	}

	user := strings.ToLower(tokens[1])
	c.config.Lock()
	defer c.config.Unlock()

	for i := 0; i < len(c.config.Stalks); i++ {
		if strings.ToLower(c.config.Stalks[i]) == user {
			c.config.Stalks = append(c.config.Stalks[:i], c.config.Stalks[i+1:]...)
			err := c.config.save()
			if err != nil {
				return err
			}
			msg := fmt.Sprintf("No longer stalking %s", user)
			c.renderCommand(msg)
			return nil
		}
	}

	return fmt.Errorf("%s is not in stalk list", user)
}

func addTag(c *chat, tokens []string) error {
	if len(tokens) < 3 {
		return errors.New("usage: /tag user [Black, Red, Green, Yellow, Blue, Magenta, Cyan, White]")
	}

	color := strings.ToLower(tokens[2])
	user := strings.ToLower(tokens[1])

	newcolor, ok := tagMap[color]
	if !ok {
		return fmt.Errorf("invalid color: %s", color)
	}

	c.config.Lock()
	if c.config.Tags == nil {
		c.config.Tags = make(map[string]string)
	}
	c.config.Tags[user] = color
	c.config.Unlock()

	err := c.config.save()
	if err != nil {
		return err
	}

	newTag := fmt.Sprintf("%s   %s", newcolor, reset)
	c.guiwrapper.applyTag(newTag, user)
	msg := fmt.Sprintf("Tagged %s", user)
	c.renderCommand(msg)
	return nil
}

func removeTag(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("usage: /untag user")
	}

	user := strings.ToLower(tokens[1])

	c.config.Lock()
	defer c.config.Unlock()

	if _, ok := c.config.Tags[user]; ok {
		delete(c.config.Tags, user)
		err := c.config.save()
		if err != nil {
			return err
		}
		newTag := fmt.Sprintf("%s   %s", none, reset)
		c.guiwrapper.applyTag(newTag, user)
		msg := fmt.Sprintf("Untagged %s", user)
		c.renderCommand(msg)
		return nil
	}

	return fmt.Errorf("%s is not tagged", user)
}

func sendMute(c *chat, tokens []string) error {
	if len(tokens) < 2 || len(tokens) > 3 {
		return errors.New("usage: /mute user [time in seconds]")
	}

	var err error
	var duration int64 // server chooses default duration

	if len(tokens) >= 3 {
		duration, err = strconv.ParseInt(strings.TrimSpace(tokens[2]), 10, 64)
		if err != nil {
			return err
		}
	}

	return c.Session.SendMute(tokens[1], time.Duration(duration)*time.Second)
}

func sendUnmute(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("usage: /unmute user")
	}

	return c.Session.SendUnmute(tokens[1])
}

func sendBan(c *chat, tokens []string) error {
	if len(tokens) < 3 || len(tokens) > 4 {
		return errors.New("usage: /[ip]ban user reason [time (in seconds)]")
	}

	var err error
	var duration int64 // server chooses default duration
	banip := tokens[0] == "/ipban"

	if len(tokens) == 4 {
		duration, err = strconv.ParseInt(strings.TrimSpace(tokens[3]), 10, 64)
		if err != nil {
			return err
		}
	}

	return c.Session.SendBan(tokens[1], tokens[2], time.Duration(duration)*time.Second, banip)
}

func sendUnban(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("usage: /unban user")
	}

	return c.Session.SendUnban(tokens[1])
}

func sendPermBan(c *chat, tokens []string) error {
	if len(tokens) != 3 {
		return errors.New("usage: /perm[ip] user reason")
	}
	banip := tokens[0] == "/permip"
	return c.Session.SendPermanentBan(tokens[1], tokens[2], banip)
}

func sendSubOnly(c *chat, tokens []string) error {
	so := tokens[1]
	if len(tokens) != 2 || (so != "on" && so != "off") {
		return errors.New("usage: /subonly {on,off}")
	}

	subonly := so == "on"
	return c.Session.SendSubOnly(subonly)
}

func sendAction(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("usage: /me message")
	}
	message := strings.Join(tokens[1:], " ")
	return c.Session.SendAction(message)
}

func sendBroadcast(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("usage: /broadcast message")
	}

	message := strings.Join(tokens[1:], " ")
	return c.Session.SendBroadcast(message)
}

func sendWhisper(c *chat, tokens []string) error {
	if len(tokens) < 3 {
		return errors.New("usage: /w user message")
	}

	nick := tokens[1]
	message := strings.Join(tokens[2:], " ")

	c.renderSendPrivateMessage(nick, message)
	return c.Session.SendPrivateMessage(nick, message)
}

func addIgnore(c *chat, tokens []string) error {
	if len(tokens) > 2 {
		return errors.New("usage: /ignore user")
	}

	c.config.Lock()
	defer c.config.Unlock()

	if len(tokens) == 1 {
		ignores := strings.Join(c.config.Ignores, ", ")
		msg := fmt.Sprintf("Ignoring the following people: %s", ignores)
		c.renderCommand(msg)
		return nil
	}
	user := strings.ToLower(tokens[1])
	if !contains(c.config.Ignores, user) {
		c.config.Ignores = append(c.config.Ignores, user)
		err := c.config.save()
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Ignoring %s", user)
		c.renderCommand(msg)
		return nil
	}
	return fmt.Errorf("%s is already ignored", user)
}

func removeIgnore(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("usage: /unignore user")
	}
	user := strings.ToLower(tokens[1])

	c.config.Lock()
	defer c.config.Unlock()

	for i, u := range c.config.Ignores {
		if u == user {
			c.config.Ignores = append(c.config.Ignores[:i], c.config.Ignores[i+1:]...)
			err := c.config.save()
			if err != nil {
				return err
			}
			msg := fmt.Sprintf("%s removed from your ignore list", u)
			c.renderCommand(msg)
			return nil
		}
	}
	return fmt.Errorf("%s is not ignored", user)
}
