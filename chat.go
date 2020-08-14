package main

import (
	"net/url"
	"sort"
	"strings"

	"github.com/awesome-gocui/gocui"
	"github.com/voloshink/dggchat"
)

const maxChatHistory = 10

type chat struct {
	config     *config
	username   string
	Session    *dggchat.Session
	emotes     []string
	guiwrapper *guiwrapper

	helpactive     bool
	debugActive    bool
	userListActive bool

	messageHistory []string
	historyIndex   int

	lastSuggestions []string
	tabIndex        int
}

func newChat(config *config, g *gocui.Gui) (*chat, error) {

	sgg, err := dggchat.New(";jwt=" + config.AuthToken)
	if err != nil {
		return nil, err
	}

	if config.CustomURL != "" {
		url, err := url.Parse(config.CustomURL)
		if err != nil {
			return nil, err
		}
		sgg.SetURL(*url)
	}

	chat := &chat{
		config:         config,
		messageHistory: []string{},
		historyIndex:   -1,
		tabIndex:       -1,
		emotes:         make([]string, 0),
		username:       config.Username,
		Session:        sgg,
		guiwrapper: &guiwrapper{
			gui:        g,
			messages:   []*guimessage{},
			maxlines:   config.Maxlines,
			timeformat: config.Timeformat,
		},
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

	// ability to send messages starting with "/"
	if len(message) >= 2 && message[:2] == "//" {
		err = c.Session.SendMessage(message[1:])
	} else if message[:1] == "/" {
		err = c.handleCommand(message)
	} else {
		err = c.Session.SendMessage(message)
	}

	if err != nil {
		c.renderError(err.Error())
		// don't return on error, append message to history
	}

	if len(c.messageHistory) > (maxChatHistory - 1) {
		c.messageHistory = append([]string{message}, c.messageHistory[:(maxChatHistory-1)]...)
	} else {
		c.messageHistory = append([]string{message}, c.messageHistory...)
	}
	c.historyIndex = -1
	c.tabIndex = -1
}

func (c *chat) tabComplete(v *gocui.View) {
	buffer := v.Buffer()

	if buffer == "" {
		return
	}

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

	// add commands to suggestions
	for cmd := range commands {
		nameSlice = append(nameSlice, cmd)
	}

	for _, name := range nameSlice {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(s)) {
			suggestions = append(suggestions, name)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

func (c *chat) sortUsers(u []dggchat.User) {
	sort.SliceStable(u, func(i, j int) bool { return strings.ToLower(u[i].Nick) < strings.ToLower(u[j].Nick) })
	sort.SliceStable(u, func(i, j int) bool {
		return c.config.Tags[strings.ToLower(u[i].Nick)] > c.config.Tags[strings.ToLower(u[j].Nick)]
	})
}
