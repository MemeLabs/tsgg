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

var backgrounds = map[string]string{
	"red":        "\u001b[41m",
	"green":      "\u001b[42m",
	"yellow":     "\u001b[43m",
	"blue":       "\u001b[44m",
	"magenta":    "\u001b[45m",
	"cyan":       "\u001b[46m",
	"brightCyan": "\u001b[46;1m",
}

var colors = map[string]string{
	"reset":         "\u001b[0m",
	"black":         "\u001b[30m",
	"red":           "\u001b[31m",
	"green":         "\u001b[32m",
	"yellow":        "\u001b[33m",
	"blue":          "\u001b[34m",
	"magenta":       "\u001b[35m",
	"cyan":          "\u001b[36m",
	"white":         "\u001b[37m",
	"brightBlack":   "\u001b[30;1m",
	"brightRed":     "\u001b[31;1m",
	"brightGreen":   "\u001b[32;1m",
	"brightYellow":  "\u001b[33;1m",
	"brightBlue":    "\u001b[34;1m",
	"brightMagenta": "\u001b[35;1m",
	"brightCyan":    "\u001b[36;1m",
	"brightWhite":   "\u001b[37;1m",
}

var decorations = map[string]string{
	"bold": "\u001b[1m",
}

var flairs = []map[string]string{
	{"flair": "flair2", "badge": "n", "color": ""},
	{"flair": "flair9", "badge": "tw", "color": colors["brightBlue"]},
	{"flair": "flair13", "badge": "t1", "color": colors["brightBlue"]},
	{"flair": "flair1", "badge": "t2", "color": colors["brightBlue"]},
	{"flair": "flair3", "badge": "t3", "color": colors["blue"]},
	{"flair": "flair8", "badge": "t4", "color": colors["magenta"]},
	{"flair": "flair11", "badge": "bot2", "color": colors["brightBlack"]},
	{"flair": "flair12", "badge": "@", "color": colors["brightCyan"]},
	{"flair": "bot", "badge": "bot", "color": colors["yellow"]},
	{"flair": "vip", "badge": "vip", "color": colors["green"]},
	{"flair": "admin", "badge": "@", "color": colors["red"]},
}

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

var helpactive = false

// TODO more info?
func showHelp(g *gocui.Gui, v *gocui.View) error {
	if !helpactive {
		maxX, maxY := g.Size()
		if messages, err := g.SetView("help", maxX/4*2, 0, maxX-20, maxY/3); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			messages.Title = " help: "
			messages.Wrap = true
			fmt.Fprint(messages, `
Commands:
    /(un)tag       user color
    /(un)highlight user
    /w(hisper)     user message
`)
			helpactive = !helpactive
			g.SetViewOnTop("help")
		}
		return nil
	}
	helpactive = !helpactive
	return g.DeleteView("help")
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
				coloredNick = fmt.Sprintf("%s%s %s", flair["color"], taggedNick, colors["reset"])
			}
		}

		for _, highlighted := range c.config.Highlighted {
			if strings.EqualFold(m.Sender.Nick, highlighted) {
				taggedNick = fmt.Sprintf("[*]%s", taggedNick)
				coloredNick = fmt.Sprintf("%s%s %s", colors["cyan"], taggedNick, colors["reset"])
			}
		}

		if coloredNick == "" {
			coloredNick = fmt.Sprintf("%s%s %s", colors["reset"], taggedNick, colors["reset"])
		}

		formattedData := m.Message
		if c.username != "" && strings.Contains(strings.ToLower(m.Message), strings.ToLower(c.username)) {
			formattedData = fmt.Sprintf("%s%s %s", backgrounds["brightCyan"], m.Message, colors["reset"])
		} else if strings.HasPrefix(m.Message, ">") {
			formattedData = fmt.Sprintf("%s%s %s", colors["green"], m.Message, colors["reset"])
		}

		formattedTag := "  "
		c.config.RLock()
		if color, ok := c.config.Tags[strings.ToLower(m.Sender.Nick)]; ok {
			formattedTag = fmt.Sprintf("%s  %s", backgrounds[color], colors["reset"])
		}
		c.config.RUnlock()

		formattedMessage := fmt.Sprintf("[%s]%s%s: %s", formattedDate, formattedTag, coloredNick, formattedData)

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

		formattedMessage := fmt.Sprintf("%s[%s] %s: %s %s", colors["brightYellow"], formattedDate, " Broadcast", b.Message, colors["reset"])
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

		formattedMessage := fmt.Sprintf("[%s]  %s%s[Whisper]%s: %s %s", formattedDate, colors["brightWhite"], decorations["bold"], pm.User.Nick, pm.Message, colors["reset"])

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

		errorMessage := fmt.Sprintf("%s*Error sending message: %s*%s", colors["red"], errorString, colors["reset"])
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

		joinMessage := fmt.Sprintf("JOIN: %s%s", join.User.Nick, colors["reset"])
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

		quitMessage := fmt.Sprintf("QUIT: %s%s", quit.User.Nick, colors["reset"])
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

		joinMessage := fmt.Sprintf("MUTE: %s muted by %s %s", mute.Target.Nick, mute.Sender.Nick, colors["reset"])
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

		joinMessage := fmt.Sprintf("UNMUTE: %s unmuted by %s %s", mute.Target.Nick, mute.Sender.Nick, colors["reset"])
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

		joinMessage := fmt.Sprintf("BAN: %s banned by %s %s", ban.Target.Nick, ban.Sender.Nick, colors["reset"])
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

		joinMessage := fmt.Sprintf("UNBAN: %s unbanned by %s %s", ban.Target.Nick, ban.Sender.Nick, colors["reset"])
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

		joinMessage := fmt.Sprintf("SUBONLY: %s changed subonly mode to: %t %s", so.Sender.Nick, so.Active, colors["reset"])
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
			color := colors["reset"]

			if flair != nil {
				color = flair["color"]
			}
			users += fmt.Sprintf("%s%s%s\n", color, u.Nick, colors["reset"])
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

func scroll(dy int, chat *chat, view string) error {
	// Grab the view that we want to scroll.
	v, _ := chat.gui.View(view)

	// Get the size and position of the view.
	_, y := v.Size()
	ox, oy := v.Origin()

	// If we're at the bottom...
	if oy+dy > strings.Count(v.ViewBuffer(), "\n")-y-1 && view != "users" {
		// Set autoscroll to normal again.
		v.Autoscroll = true
	} else {
		// Set autoscroll to false and scroll.
		v.Autoscroll = false
		v.SetOrigin(ox, oy+dy)
	}
	return nil
}
