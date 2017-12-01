package main

import (
	"fmt"
	"log"
	"net/url"
	"sort"
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

		formattedMessage := fmt.Sprintf("[%s]  %s%s[Whisper]%s: %s %s %s", formattedDate, decorations["bold"], colors["brightWhite"], c.username, nick, message, colors["reset"])

		fmt.Fprintln(messagesView, formattedMessage)
		return nil
	})
	return err
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
