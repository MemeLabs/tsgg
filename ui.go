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

var userlistShown = true

const (
	none  color = ""
	reset color = "\u001b[0m"

	Bold      color = "\u001b[1m"
	Underline color = "\u001b[4m"
	Reversed  color = "\u001b[7m"

	bgBlack   color = "\u001b[40m"
	bgRed     color = "\u001b[41m"
	bgGreen   color = "\u001b[42m"
	bgYellow  color = "\u001b[43m"
	bgBlue    color = "\u001b[44m"
	bgMagenta color = "\u001b[45m"
	bgCyan    color = "\u001b[46m"
	bgWhite   color = "\u001b[47m"

	bgBrightBlack   color = "\u001b[40;1m"
	bgBrightRed     color = "\u001b[41;1m"
	bgBrightGreen   color = "\u001b[42;1m"
	bgBrightYellow  color = "\u001b[43;1m"
	bgBrightBlue    color = "\u001b[44;1m"
	bgBrightMagenta color = "\u001b[45;1m"
	bgBrightCyan    color = "\u001b[46;1m"
	bgBrightWhite   color = "\u001b[47;1m"

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

	if messages, err := g.SetView("help", maxX/4*2, 0, maxX-20, maxY/2); err != nil {
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
			if !commands[k].privileged {
				fmt.Fprintf(messages, "  - %s %s\n", k, commands[k].usage)
			}
		}
		// TODO: only print those if user can use them, and remove the pasted loop.
		for _, k := range keys {
			if commands[k].privileged {
				fmt.Fprintf(messages, "  * %s %s\n", k, commands[k].usage)
			}
		}

	}

	var xDimension = maxX - 20
	if !userlistShown {
		xDimension = maxX - 1
	}

	if messages, err := g.SetView("messages", 0, 0, xDimension, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		messages.Title = " messages: "
		messages.Autoscroll = true
		messages.Wrap = true
	}

	if input, err := g.SetView("input", 0, maxY-3, xDimension, maxY-1); err != nil {
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

func (c *chat) showHelp(g *gocui.Gui, v *gocui.View) error {
	c.helpactive = !c.helpactive
	if !c.helpactive {
		_, err := g.SetViewOnTop("help")
		return err
	}
	_, err := g.SetViewOnBottom("help")
	return err
}

func (c *chat) showDebug(g *gocui.Gui, v *gocui.View) error {
	c.debugActive = !c.debugActive
	if !c.debugActive {
		_, err := g.SetViewOnTop("debug")
		return err
	}
	_, err := g.SetViewOnBottom("debug")
	return err
}

func (c *chat) showUserList(g *gocui.Gui, v *gocui.View) error {
	c.userListActive = !c.userListActive
	userlistShown = !userlistShown
	if !c.userListActive {
		_, err := g.SetViewOnTop("users")
		return err
	}
	_, err := g.SetViewOnBottom("users")
	return err
}

func (c *chat) renderDebug(s interface{}) {
	c.guiwrapper.gui.Update(func(g *gocui.Gui) error {
		debugView, err := g.View("debug")
		if err != nil {
			return err
		}

		errorMessage := fmt.Sprintf("DEBUG: %+v", s)
		fmt.Fprintln(debugView, errorMessage)
		return nil
	})
}

func (c *chat) renderError(errorString string) {
	tag := fmt.Sprintf(" %sX%s ", bgRed, reset)
	msg := fmt.Sprintf("%s*Error sending message: %s*%s", fgBrightRed, errorString, reset)
	c.guiwrapper.addMessage(guimessage{time.Now(), tag, msg, ""})
}

func (c *chat) isHighlighted(message string) bool {
	for _, highlighted := range c.config.Highlighted {
		if strings.Contains(strings.ToLower(message), strings.ToLower(highlighted)) {
			return true
		}
	}
	return false
}

func (c *chat) isTagged(user string) bool {
	for tag := range c.config.Tags {
		if strings.EqualFold(strings.ToLower(user), strings.ToLower(tag)) {
			return true
		}
	}
	return false
}

func (c *chat) renderMessage(m dggchat.Message) {

	taggedNick := m.Sender.Nick

	// don't show ignored users
	if contains(c.config.Ignores, strings.ToLower(taggedNick)) {
		return
	}

	var coloredNick string

	if c.isTagged(m.Sender.Nick) {
		coloredNick = fmt.Sprintf("%s%s %s", tagMap[c.config.Tags[strings.ToLower(m.Sender.Nick)]], taggedNick, reset) //change color of username if they are tagged
	}

	if coloredNick == "" {
		coloredNick = fmt.Sprintf("%s%s%s", Bold, taggedNick, reset)
	}

	formattedData := m.Message
	if c.username != "" && strings.Contains(strings.ToLower(m.Message), strings.ToLower(c.username)) || c.isHighlighted(m.Message) {
		formattedData = fmt.Sprintf("%s%s%s%s", c.config.HighlightBg, c.config.HighlightFg, m.Message, reset) //change message color if you get mentioned or the message contains a highlighed string
	} else if strings.HasPrefix(m.Message, ">") {
		formattedData = fmt.Sprintf("%s%s%s", fgGreen, m.Message, reset) //greentext
	}

	//currently not in use
	formattedTag := "   "
	c.config.RLock()
	if color, ok := c.config.Tags[strings.ToLower(m.Sender.Nick)]; ok {
		formattedTag = fmt.Sprintf("%s   %s", tagMap[color], reset)
	}
	c.config.RUnlock()

	// TODO alignment possibly
	// var align = 23
	// padlen := align - len(strings.TrimSpace(taggedNick)) //len of tagged, because color codes mess up calc
	// if padlen > 0 && len(strings.TrimSpace(taggedNick)) < align {
	// 	coloredNick = strings.Repeat(" ", padlen) + coloredNick
	// }

	msg := fmt.Sprintf("%s: %s", coloredNick, formattedData)
	c.guiwrapper.addMessage(guimessage{m.Timestamp, formattedTag, msg, m.Sender.Nick})
}

func (c *chat) renderPrivateMessage(pm dggchat.PrivateMessage) {
	tag := fmt.Sprintf(" %s%s*%s ", bgBlack, fgRed, reset)
	msg := fmt.Sprintf("%s[PM <- %s] %s %s", fgBrightWhite, pm.User.Nick, pm.Message, reset)
	c.guiwrapper.addMessage(guimessage{pm.Timestamp, tag, msg, ""})
}

func (c *chat) renderSendPrivateMessage(nick string, message string) {
	tag := fmt.Sprintf(" %s%s*%s ", bgBlack, fgRed, reset)
	msg := fmt.Sprintf("%s[PM -> %s] %s %s", fgBrightWhite, nick, message, reset)
	c.guiwrapper.addMessage(guimessage{time.Now(), tag, msg, ""})
}

func (c *chat) renderBroadcast(b dggchat.Broadcast) {
	tag := fmt.Sprintf(" %s!%s ", fgBrightYellow, reset)
	extra := ""
	if b.Sender.Nick != "" {
		extra = fmt.Sprintf("from %s ", b.Sender.Nick)
	}
	msg := fmt.Sprintf("%sBROADCAST %s: %s %s", fgBrightYellow, extra, b.Message, reset)
	c.guiwrapper.addMessage(guimessage{b.Timestamp, tag, msg, ""})
}

func (c *chat) renderJoin(join dggchat.RoomAction) {
	if contains(c.config.Stalks, strings.ToLower(join.User.Nick)) || c.config.ShowJoinLeave {
		tag := fmt.Sprintf(" %s>%s ", bgGreen, reset)
		msg := fmt.Sprintf("%s%s joined!%s", fgGreen, join.User.Nick, reset)
		c.guiwrapper.addMessage(guimessage{join.Timestamp, tag, msg, ""})
	}
}

func (c *chat) renderQuit(quit dggchat.RoomAction) {
	if contains(c.config.Stalks, strings.ToLower(quit.User.Nick)) || c.config.ShowJoinLeave {
		tag := fmt.Sprintf(" %s<%s ", bgRed, reset)
		msg := fmt.Sprintf("%s%s left.%s", fgRed, quit.User.Nick, reset)
		c.guiwrapper.addMessage(guimessage{quit.Timestamp, tag, msg, ""})
	}
}

func (c *chat) renderMute(mute dggchat.Mute) {
	tag := fmt.Sprintf(" %s!%s ", bgYellow, reset)
	msg := fmt.Sprintf("%s%s muted by %s%s", fgYellow, mute.Target.Nick, mute.Sender.Nick, reset)
	c.guiwrapper.addMessage(guimessage{mute.Timestamp, tag, msg, ""})
}

func (c *chat) renderUnmute(unmute dggchat.Mute) {
	tag := fmt.Sprintf(" %s!%s ", bgYellow, reset)
	msg := fmt.Sprintf("%s%s unmuted by %s%s", fgYellow, unmute.Target.Nick, unmute.Sender.Nick, reset)
	c.guiwrapper.addMessage(guimessage{unmute.Timestamp, tag, msg, ""})
}

func (c *chat) renderBan(ban dggchat.Ban) {
	tag := fmt.Sprintf(" %s!%s ", bgRed, reset)
	msg := fmt.Sprintf("%s%s banned by %s%s", fgRed, ban.Target.Nick, ban.Sender.Nick, reset)
	c.guiwrapper.addMessage(guimessage{ban.Timestamp, tag, msg, ""})
}

func (c *chat) renderUnban(unban dggchat.Ban) {
	tag := fmt.Sprintf(" %s!%s ", bgRed, reset)
	msg := fmt.Sprintf("%s%s unbanned by %s%s", fgRed, unban.Target.Nick, unban.Sender.Nick, reset)
	c.guiwrapper.addMessage(guimessage{unban.Timestamp, tag, msg, ""})
}

func (c *chat) renderSubOnly(so dggchat.SubOnly) {
	tag := fmt.Sprintf(" %s$%s ", bgMagenta, reset)
	msg := fmt.Sprintf("%s%s changed subonly mode to: %t %s", fgMagenta, so.Sender.Nick, so.Active, reset)
	c.guiwrapper.addMessage(guimessage{so.Timestamp, tag, msg, ""})
}

func (c *chat) renderCommand(s string) {
	tag := fmt.Sprintf(" %sI%s ", bgWhite, reset)
	msg := fmt.Sprintf("%s%s%s", fgWhite, s, reset)
	c.guiwrapper.addMessage(guimessage{time.Now(), tag, msg, ""})
}

func (c *chat) renderUsers(users []dggchat.User) {
	c.guiwrapper.gui.Update(func(g *gocui.Gui) error {
		userView, err := g.View("users")
		if err != nil {
			log.Println(err)
			return err
		}

		userView.Title = fmt.Sprintf("%d users:", len(users))
		c.sortUsers(users)

		var usersList string
		for _, u := range users {
			if c.isTagged(u.Nick) {
				usersList += fmt.Sprintf("%s%s%s\n", tagMap[c.config.Tags[strings.ToLower(u.Nick)]], u.Nick, reset)
			} else {
				usersList += fmt.Sprintf("%s%s\n", u.Nick, reset)
			}
		}

		userView.Clear()
		fmt.Fprintln(userView, usersList)
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

func (c *chat) historyUp(g *gocui.Gui, v *gocui.View) error {
	if c.historyIndex > maxChatHistory-2 || (c.historyIndex+1) > len(c.messageHistory)-1 {
		return nil
	}
	c.historyIndex++
	v.Clear()
	v.SetCursor(0, 0)
	v.Write([]byte(c.messageHistory[c.historyIndex]))
	v.MoveCursor(len(c.messageHistory[c.historyIndex]), 0, true)
	return nil
}

func (c *chat) historyDown(g *gocui.Gui, v *gocui.View) error {
	if c.historyIndex < 1 {
		c.historyIndex = -1
		v.Clear()
		v.SetCursor(0, 0)
		return nil
	}

	c.historyIndex--
	v.Clear()
	v.SetCursor(0, 0)
	v.Write([]byte(c.messageHistory[c.historyIndex]))
	v.MoveCursor(len(c.messageHistory[c.historyIndex]), 0, true)
	return nil
}

func scroll(dy int, chat *chat, view string) error {

	chat.guiwrapper.Lock()
	defer chat.guiwrapper.Unlock()

	// Grab the view that we want to scroll.
	v, _ := chat.guiwrapper.gui.View(view)

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
		chat.guiwrapper.redraw() // see comment in redraw()
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
