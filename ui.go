package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/voloshink/dggchat"
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
	{"flair": "flair11", "badge": "bot2", "color": "\u001b[30;1m"},
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

func (c *chat) renderMessage(m dggchat.Message) {
	c.gui.Update(func(g *gocui.Gui) error {
		messagesView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		formattedDate := m.Timestamp.Format(time.Kitchen)

		taggedNick := m.Sender.Nick
		var coloredNick string

		for _, flair := range flairs {
			if contains(m.Sender.Features, flair["flair"]) {
				taggedNick = fmt.Sprintf("[%s]%s", flair["badge"], taggedNick)
				coloredNick = fmt.Sprintf("%s %s %s", flair["color"], taggedNick, colorReset)
			}
		}

		for _, highlighted := range c.config.Highlighted {
			if strings.EqualFold(m.Sender.Nick, highlighted) {
				taggedNick = fmt.Sprintf("[*]%s", taggedNick)
				coloredNick = fmt.Sprintf("\u001b[36m %s %s", taggedNick, colorReset)
			}
		}

		if coloredNick == "" {
			coloredNick = fmt.Sprintf("%s %s %s", colorReset, taggedNick, colorReset)
		}

		formattedData := m.Message
		if c.username != "" && strings.Contains(strings.ToLower(m.Message), strings.ToLower(c.username)) {
			formattedData = fmt.Sprintf("\u001b[46;1m%s %s", m.Message, colorReset)
		}

		formattedMessage := fmt.Sprintf("[%s] %s: %s", formattedDate, coloredNick, formattedData)

		fmt.Fprintln(messagesView, formattedMessage)
		return nil
	})
}

func (c *chat) renderBroadcast(b dggchat.Broadcast) {
	c.gui.Update(func(g *gocui.Gui) error {
		messagesView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		formattedDate := b.Timestamp.Format(time.Kitchen)

		formattedMessage := fmt.Sprintf("\u001b[33;1m[%s] %s: %s %s", formattedDate, " Broadcast", b.Message, colorReset)
		fmt.Fprintln(messagesView, formattedMessage)
		return nil
	})
}

func (c *chat) renderPrivateMessage(pm dggchat.PrivateMessage) {
	c.gui.Update(func(g *gocui.Gui) error {
		messagesView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		formattedDate := pm.Timestamp.Format(time.Kitchen)

		formattedMessage := fmt.Sprintf("[%s]  \u001b[37;1m\u001b[1m[Whisper]%s: %s %s", formattedDate, pm.User.Nick, pm.Message, colorReset)

		fmt.Fprintln(messagesView, formattedMessage)
		return nil
	})
}

func (c *chat) renderError(errorString string) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		errorMessage := fmt.Sprintf("\u001b[31m*Error sending message: %s*%s", errorString, colorReset)
		fmt.Fprintln(messageView, errorMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderDebug(s interface{}) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		errorMessage := fmt.Sprintf("DEBUG: %+v", s)
		fmt.Fprintln(messageView, errorMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderJoin(join dggchat.RoomAction) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		joinMessage := fmt.Sprintf("JOIN: %s%s", join.User.Nick, colorReset)
		fmt.Fprintln(messageView, joinMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderQuit(quit dggchat.RoomAction) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		quitMessage := fmt.Sprintf("QUIT: %s%s", quit.User.Nick, colorReset)
		fmt.Fprintln(messageView, quitMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderMute(mute dggchat.Mute) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		joinMessage := fmt.Sprintf("MUTE: %s muted by %s %s", mute.Target.Nick, mute.Sender.Nick, colorReset)
		fmt.Fprintln(messageView, joinMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderUnmute(mute dggchat.Mute) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		joinMessage := fmt.Sprintf("UNMUTE: %s unmuted by %s %s", mute.Target.Nick, mute.Sender.Nick, colorReset)
		fmt.Fprintln(messageView, joinMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderBan(ban dggchat.Ban) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		joinMessage := fmt.Sprintf("BAN: %s banned by %s %s", ban.Target.Nick, ban.Sender.Nick, colorReset)
		fmt.Fprintln(messageView, joinMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderUnban(ban dggchat.Ban) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		joinMessage := fmt.Sprintf("UNBAN: %s unbanned by %s %s", ban.Target.Nick, ban.Sender.Nick, colorReset)
		fmt.Fprintln(messageView, joinMessage)
		return nil
	})
}

// TODO colors
func (c *chat) renderSubOnly(so dggchat.SubOnly) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		joinMessage := fmt.Sprintf("SUBONLY: %s changed subonly mode to: %t %s", so.Sender.Nick, so.Active, colorReset)
		fmt.Fprintln(messageView, joinMessage)
		return nil
	})
}

func (c *chat) renderUsers(dggusers []dggchat.User) {
	c.gui.Update(func(g *gocui.Gui) error {
		userView, err := g.View("users")
		if err != nil {
			log.Println(err)
			return err
		}

		userView.Title = fmt.Sprintf("%d users:", len(dggusers))
		sortUsers(dggusers)

		var users string
		for _, u := range dggusers {
			_, flair := highestFlair(u)
			color := colorReset

			if flair != nil {
				color = flair["color"]
			}
			users += fmt.Sprintf("%s%s%s\n", color, u.Nick, colorReset)
		}

		userView.Clear()
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

func sortUsers(u []dggchat.User) {
	sort.SliceStable(u, func(i, j int) bool {
		iUser := u[i]
		jUser := u[j]

		iIndex, _ := highestFlair(iUser)
		jIndex, _ := highestFlair(jUser)

		return iIndex > jIndex
	})
}

func highestFlair(u dggchat.User) (int, map[string]string) {
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

func historyUp(g *gocui.Gui, v *gocui.View, chat *chat) error {
	if chat.historyIndex > maxChatHistory-2 || (chat.historyIndex+1) > len(chat.messageHistory)-1 {
		return nil
	}
	chat.historyIndex++
	v.Clear()
	v.SetCursor(0, 0)
	v.Write([]byte(chat.messageHistory[chat.historyIndex]))
	v.MoveCursor(len(chat.messageHistory[chat.historyIndex]), 0, true)
	return nil
}

func historyDown(g *gocui.Gui, v *gocui.View, chat *chat) error {
	if chat.historyIndex < 1 {
		return nil
	}

	chat.historyIndex--
	v.Clear()
	v.SetCursor(0, 0)
	v.Write([]byte(chat.messageHistory[chat.historyIndex]))
	v.MoveCursor(len(chat.messageHistory[chat.historyIndex]), 0, true)
	return nil
}
