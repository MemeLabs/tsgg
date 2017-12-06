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

type color string

const (
	none  color = ""
	reset color = "\u001b[0m"

	bgBlack   color = "\u001b[40m"
	bgRed     color = "\u001b[41m"
	bgGreen   color = "\u001b[42m"
	bgYellow  color = "\u001b[43m"
	bgBlue    color = "\u001b[44m"
	bgMagenta color = "\u001b[45m"
	bgCyan    color = "\u001b[46m"
	bgWhite   color = "\u001b[47m"

	// TODO add bright bg colors

	fgBlack   color = "\u001b[30m"
	fgRed     color = "\u001b[31m"
	fgGreen   color = "\u001b[32m"
	fgYellow  color = "\u001b[33m"
	fgBlue    color = "\u001b[34m"
	fgMagenta color = "\u001b[35m"
	fgCyan    color = "\u001b[36m"
	fgWhite   color = "\u001b[37m"

	fgBrightBlack   color = "\u001b[30;1m"
	fgBrightRed     color = "\u001b[31;1m"
	fgBrightGreen   color = "\u001b[32;1m"
	fgBrightYellow  color = "\u001b[33;1m"
	fgBrightBlue    color = "\u001b[34;1m"
	fgBrightMagenta color = "\u001b[35;1m"
	fgBrightCyan    color = "\u001b[36;1m"
	fgBrightWhite   color = "\u001b[37;1m"
)

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

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	g.Cursor = true

	if messages, err := g.SetView("debug", maxX/4*2, 0, maxX-20, maxY/3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		messages.Title = " debug: "
		messages.Wrap = true
		messages.Autoscroll = true
	}

	if messages, err := g.SetView("help", maxX/4*2, 0, maxX-20, maxY/3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		messages.Title = " help: "
		messages.Wrap = true

		// command map is unordered, we want the help menu to be stable
		keys := make([]string, 0, len(commands))
		for key := range commands {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		fmt.Fprint(messages, "Commands:\n")
		for _, k := range keys {
			fmt.Fprintf(messages, "  - %s %s\n", k, commands[k].usage)
		}
	}

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

func showHelp(g *gocui.Gui, v *gocui.View) error {
	if !helpactive {
		helpactive = !helpactive
		_, err := g.SetViewOnTop("help")
		return err
	}
	helpactive = !helpactive
	_, err := g.SetViewOnBottom("help")
	return err
}

var debugActive = false

func showDebug(g *gocui.Gui, v *gocui.View) error {
	if !debugActive {
		debugActive = !debugActive
		_, err := g.SetViewOnTop("debug")
		return err
	}
	debugActive = !debugActive
	_, err := g.SetViewOnBottom("debug")
	return err
}

func (c *chat) renderError(errorString string) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		errorMessage := fmt.Sprintf("%s*Error sending message: %s*%s", fgRed, errorString, reset)
		fmt.Fprintln(messageView, errorMessage)
		return nil
	})
}

func (c *chat) renderDebug(s interface{}) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("debug")
		if err != nil {
			log.Println(err)
			return err
		}

		errorMessage := fmt.Sprintf("DEBUG: %+v", s)
		fmt.Fprintln(messageView, errorMessage)
		return nil
	})
}

func (c *chat) renderMessage(m dggchat.Message) {

	taggedNick := m.Sender.Nick

	// don't show ignored users
	if contains(c.config.Ignores, strings.ToLower(taggedNick)) {
		return
	}

	var coloredNick string

	for _, flair := range c.flairs {
		if contains(m.Sender.Features, flair.Name) {
			taggedNick = fmt.Sprintf("[%s]%s", flair.Badge, taggedNick)
			coloredNick = fmt.Sprintf("%s%s %s", flair.Color, taggedNick, reset)
		}
	}

	for _, highlighted := range c.config.Highlighted {
		if strings.EqualFold(m.Sender.Nick, highlighted) {
			taggedNick = fmt.Sprintf("[*]%s", taggedNick)
			coloredNick = fmt.Sprintf("%s%s %s", fgCyan, taggedNick, reset)
		}
	}

	if coloredNick == "" {
		coloredNick = fmt.Sprintf("%s%s %s", reset, taggedNick, reset)
	}

	formattedData := m.Message
	if c.username != "" && strings.Contains(strings.ToLower(m.Message), strings.ToLower(c.username)) {
		formattedData = fmt.Sprintf("%s%s %s", bgCyan, m.Message, reset)
	} else if strings.HasPrefix(m.Message, ">") {
		formattedData = fmt.Sprintf("%s%s %s", fgGreen, m.Message, reset)
	}

	formattedTag := "   "
	c.config.RLock()
	if color, ok := c.config.Tags[strings.ToLower(m.Sender.Nick)]; ok {
		formattedTag = fmt.Sprintf("%s   %s", tagMap[color], reset)
	}
	c.config.RUnlock()

	msg := fmt.Sprintf("%s%s: %s", formattedTag, coloredNick, formattedData)
	c.renderFormattedMessage(msg, m.Timestamp)
}

func (c *chat) renderPrivateMessage(pm dggchat.PrivateMessage) {
	tag := fmt.Sprintf(" %s*%s ", bgWhite, reset)
	msg := fmt.Sprintf("%s%s[PM <- %s] %s %s", tag, fgBrightWhite, pm.User.Nick, pm.Message, reset)
	c.renderFormattedMessage(msg, pm.Timestamp)
}

func (c *chat) renderBroadcast(b dggchat.Broadcast) {
	tag := fmt.Sprintf(" %s!%s ", fgBrightYellow, reset)
	msg := fmt.Sprintf("%s%sBROADCAST: %s %s", tag, fgBrightYellow, b.Message, reset)
	c.renderFormattedMessage(msg, b.Timestamp)
}

func (c *chat) renderJoin(join dggchat.RoomAction) {
	tag := fmt.Sprintf(" %s>%s ", bgGreen, reset)
	msg := fmt.Sprintf("%s%s%s joined!%s", tag, fgGreen, join.User.Nick, reset)
	c.renderFormattedMessage(msg, join.Timestamp)
}

func (c *chat) renderQuit(quit dggchat.RoomAction) {
	tag := fmt.Sprintf(" %s<%s ", bgRed, reset)
	msg := fmt.Sprintf("%s%s%s left.%s", tag, fgRed, quit.User.Nick, reset)
	c.renderFormattedMessage(msg, quit.Timestamp)
}

func (c *chat) renderMute(mute dggchat.Mute) {
	tag := fmt.Sprintf(" %s!%s ", bgYellow, reset)
	msg := fmt.Sprintf("%s%s%s muted by %s%s", tag, fgYellow, mute.Target.Nick, mute.Sender.Nick, reset)
	c.renderFormattedMessage(msg, mute.Timestamp)
}

func (c *chat) renderUnmute(mute dggchat.Mute) {
	tag := fmt.Sprintf(" %s!%s ", bgYellow, reset)
	msg := fmt.Sprintf("%s%s%s unmuted by %s%s", tag, fgYellow, mute.Target.Nick, mute.Sender.Nick, reset)
	c.renderFormattedMessage(msg, mute.Timestamp)
}

func (c *chat) renderBan(ban dggchat.Ban) {
	tag := fmt.Sprintf(" %s!%s ", bgRed, reset)
	msg := fmt.Sprintf("%s%s%s banned by %s%s", tag, fgRed, ban.Target.Nick, ban.Sender.Nick, reset)
	c.renderFormattedMessage(msg, ban.Timestamp)
}

func (c *chat) renderUnban(unban dggchat.Ban) {
	tag := fmt.Sprintf(" %s!%s ", bgRed, reset)
	msg := fmt.Sprintf("%s%s%s unbanned by %s%s", tag, fgRed, unban.Target.Nick, unban.Sender.Nick, reset)
	c.renderFormattedMessage(msg, unban.Timestamp)
}

func (c *chat) renderSubOnly(so dggchat.SubOnly) {
	tag := fmt.Sprintf(" %s$%s ", bgMagenta, reset)
	msg := fmt.Sprintf("%s%s%s changed subonly mode to: %t %s", tag, fgMagenta, so.Sender.Nick, so.Active, reset)
	c.renderFormattedMessage(msg, so.Timestamp)
}

func (c *chat) renderCommand(s string) {
	tm := time.Unix(time.Now().Unix()/1000, 0)
	tag := fmt.Sprintf(" %sI%s ", bgWhite, reset)
	msg := fmt.Sprintf("%s%s%s%s", tag, fgWhite, s, reset)
	c.renderFormattedMessage(msg, tm)
}

func (c *chat) renderUsers(dggusers []dggchat.User) {
	c.gui.Update(func(g *gocui.Gui) error {
		userView, err := g.View("users")
		if err != nil {
			log.Println(err)
			return err
		}

		userView.Title = fmt.Sprintf("%d users:", len(dggusers))
		c.sortUsers(dggusers)

		var users string
		for _, u := range dggusers {
			_, flair := c.highestFlair(u)
			users += fmt.Sprintf("%s%s%s\n", flair.Color, u.Nick, reset)
		}

		userView.Clear()
		fmt.Fprintln(userView, users)
		return nil
	})
}

func (c *chat) renderFormattedMessage(s string, t time.Time) {
	c.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			log.Println(err)
			return err
		}

		formattedDate := t.Format(c.config.Timeformat)
		m := fmt.Sprintf("[%s]%s", formattedDate, s)
		fmt.Fprintln(messageView, m)
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
		chat.historyIndex = -1
		v.Clear()
		v.SetCursor(0, 0)
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

	// target y
	ty := oy + dy

	// If we're at the bottom...
	lines := strings.Count(v.ViewBuffer(), "\n") - y - 1

	chat.renderDebug(fmt.Sprintf("scroll: ox: %d, oy: %d, dy: %d, y: %d, lines: %d", ox, oy, dy, y, lines))
	if ty > lines && view == "messages" {
		// Set autoscroll to normal again.
		v.Autoscroll = true
		return nil
	}
	// Set autoscroll to false and scroll.
	v.Autoscroll = false

	// If the scrolling "speed" (dy) is set too high, make sure we don't scroll into negative.
	if ty < 0 {
		ty = 0
	}

	// Do not scroll at all if the view is not full.
	if (view == "users" || view == "help" || view == "debug") && strings.Count(v.Buffer(), "\n") < y {
		ty = 0
	}

	// If the end (by amount of lines) of a view is reached, do not scroll even further down into nothingness.
	if oy > lines {
		ty = lines
	}

	v.SetOrigin(ox, ty)

	return nil
}
