package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

// flair9 - twithcsub
// flair13 - t1
// flair1 - t2
// flair3 - t3
// flair8 - t4
// flair2 - notable
// protected
// bot
// vip - green
// admin - red

var flairs = []map[string]string{
	{"flair": "flair2", "badge": "n", "color": ""},
	{"flair": "flair9", "badge": "tw", "color": "\u001b[34;1m"},
	{"flair": "flair13", "badge": "t1", "color": "\u001b[34;1m"},
	{"flair": "flair1", "badge": "t2", "color": "\u001b[34;1m"},
	{"flair": "flair3", "badge": "t3", "color": "\u001b[34m"},
	{"flair": "flair8", "badge": "t4", "color": "\u001b[35m"},
	{"flair": "bot", "badge": "bot", "color": "\u001b[33m"},
	{"flair": "vip", "badge": "vip", "color": "\u001b[32m"},
	{"flair": "admin", "badge": "@", "color": "\u001b[31m"},
}

const colorReset = "\u001b[0m"

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	g.Cursor = true

	if messages, err := g.SetView("messages", 0, 0, maxX-20, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		messages.Title = " messages: "
		messages.Autoscroll = true
		messages.Wrap = true
	}

	if input, err := g.SetView("input", 0, maxY-3, maxX-20, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		input.Title = " send: "
		input.Autoscroll = false
		input.Wrap = true
		input.Editable = true

		g.SetCurrentView("input")
	}

	if users, err := g.SetView("users", maxX-20, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		users.Title = " users: "
		users.Autoscroll = false
		users.Wrap = false
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func renderMessage(g *gocui.Gui, m *chatMessage) {
	g.Update(func(g *gocui.Gui) error {
		messagesView, err := g.View("messages")
		if err != nil {
			log.Fatalln(err)
		}

		tm := time.Unix(m.Timestamp/1000, 0)
		formattedDate := tm.Format(time.Kitchen)

		taggedNick := m.Nick
		var coloredNick string

		for _, flair := range flairs {
			if contains(m.Features, flair["flair"]) {
				taggedNick = fmt.Sprintf("[%s]%s", flair["badge"], taggedNick)
				coloredNick = fmt.Sprintf("%s %s %s", flair["color"], taggedNick, colorReset)
			}
		}

		// why not
		if m.Nick == "Polecat" {
			taggedNick = fmt.Sprintf("[*]%s", taggedNick)
			coloredNick = fmt.Sprintf("\u001b[36m %s %s", taggedNick, colorReset)
		}

		if coloredNick == "" {
			coloredNick = fmt.Sprintf("%s %s %s", colorReset, taggedNick, colorReset)
		}

		formattedMessage := fmt.Sprintf("[%s] %s: %s", formattedDate, coloredNick, m.Data)
		fmt.Fprintln(messagesView, formattedMessage)
		return nil
	})
}

func renderError(g *gocui.Gui, errorString string) {
	g.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Fatalln(err)
		}

		errorMessage := fmt.Sprintf("\u001b[31m*Error sending message: %s*%s", errorString, colorReset)
		fmt.Fprintln(messageView, errorMessage)
		return nil
	})
}

func renderUsers(g *gocui.Gui, u *userList) {
	g.Update(func(g *gocui.Gui) error {
		userView, err := g.View("users")
		if err != nil {
			log.Fatalln(err)
		}

		userView.Title = fmt.Sprintf("%d users:", u.Count)
		sortUsers(u.Users)

		var users string
		for _, u := range u.Users {
			_, flair := highestFlair(u)
			color := colorReset

			if flair != nil {
				color = flair["color"]
			}
			users += fmt.Sprintf("%s%s%s\n", color, u.Nick, colorReset)
		}

		fmt.Fprintln(userView, users)
		return nil
	})
}

func contains(s []string, q string) bool {
	return indexOf(s, q) > -1
}

func indexOf(s []string, e string) int {
	for i, element := range s {
		if strings.EqualFold(element, e) {
			return i
		}
	}

	return -1
}

func sortUsers(u []user) {
	sort.SliceStable(u, func(i, j int) bool {
		iUser := u[i]
		jUser := u[j]

		iIndex, _ := highestFlair(iUser)
		jIndex, _ := highestFlair(jUser)

		return iIndex > jIndex
	})
}

func highestFlair(u user) (int, map[string]string) {
	index := -1
	var highestFlair map[string]string

	for i, flair := range flairs {
		if contains(u.Features, flair["flair"]) {
			index = i
			highestFlair = flair
		}
	}

	return index, highestFlair
}
