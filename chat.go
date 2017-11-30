package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/voloshink/dggchat"
)

const maxChatHistory = 10

type chat struct {
	config   *config
	gui      *gocui.Gui
	username string
	Session  *dggchat.Session
	emotes   []string

	messageHistory []string
	historyIndex   int

	lastSuggestions []string
	tabIndex        int
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
		tabIndex:       -1,
		emotes:         make([]string, 0),
		username:       config.Username,
		Session:        dgg,
	}

	// don't wait for emotes to load
	go func() {
		emotes, _ := getEmotes()
		chat.emotes = emotes
	}()

	return chat, nil
}

func (c *chat) handleInput(message string) {

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
	c.tabIndex = -1
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

func (c *chat) tabComplete(v *gocui.View) {
	buffer := v.Buffer()

	// strip \n
	buffer = buffer[:len(buffer)-1]
	if len(buffer) == 0 {
		return
	}

	x, _ := v.Cursor()
	strSlice := strings.Split(buffer, " ")

	runeIndex := 0
	var selectedWordIndex int

	for i, word := range strSlice {
		if i == len(strSlice)-1 {
			runeIndex++
		}

		if x >= runeIndex && x <= runeIndex+len(word) {
			selectedWordIndex = i
			break
		}
		runeIndex += len(word) + 1
	}

	selectedWord := strSlice[selectedWordIndex]

	if selectedWordIndex != len(strSlice)-1 {
		// for now just deal with tabbing the last word
		// tabbing words in the middle of a sentance
		// is very annoying to implement
		return
	}

	if len(selectedWord) < 2 && selectedWord != "" {
		return
	}

	var suggestions []string
	if c.tabIndex != -1 && c.lastSuggestions[c.tabIndex] == selectedWord {
		suggestions = c.lastSuggestions
	} else {
		suggestions = c.generateSuggestions(selectedWord)
	}

	if len(suggestions) == 0 {
		return
	}

	// movement logic
	if c.tabIndex < len(suggestions)-1 {
		c.tabIndex++
	} else {
		c.tabIndex = 0
	}

	c.lastSuggestions = suggestions
	suggestion := suggestions[c.tabIndex]
	strSlice[selectedWordIndex] = suggestion
	newBuffer := []byte(strings.Join(strSlice, " "))

	newCursor := len(newBuffer) + 1
	// for i, word := range strSlice {
	// 	if i <= selectedWordIndex {
	// 		newCursor += len(word) + 1
	// 	}
	// }

	v.Clear()
	v.SetOrigin(0, 0)
	v.Write(newBuffer)
	v.SetCursor(newCursor, 0)
}

func (c *chat) generateSuggestions(s string) []string {
	users := c.Session.GetUsers()
	suggestions := make([]string, 0)

	nameSlice := make([]string, 0)

	for _, user := range users {
		nameSlice = append(nameSlice, user.Nick)
	}

	nameSlice = append(nameSlice, c.emotes...)

	for _, name := range nameSlice {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(s)) {
			suggestions = append(suggestions, name)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}
