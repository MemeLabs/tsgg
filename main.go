package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/jroimartin/gocui"
	"github.com/voloshink/dggchat"
)

type config struct {
	AuthToken       string            `toml:"auth_token"`
	CustomURL       string            `toml:"custom_url"`
	Username        string            `toml:"username"`
	Timeformat      string            `toml:"timeformat"`
	Maxlines        int               `toml:"maxlines"`
	ScrollingSpeed  int               `toml:"scrolling_speed"`
	PageUpDownSpeed int               `toml:"page_up_down_Speed"`
	Highlighted     []string          `toml:"highlighted"`
	Tags            map[string]string `toml:"tags"`
	Ignores         []string          `toml:"ignores"`
	Stalks          []string          `toml:"stalks"`
	ShowJoinLeave   bool              `toml:"showjoinleave"`
	HighlightColor  string            `toml:"highlight_color"`
	TagColor        string            `toml:"tag_color"`
	HighlightBg     string            `toml:"highlight_bg_color"`
	HighlightFg     string            `toml:"highlight_fg_color"`
	sync.RWMutex
}

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.toml", "location of config file to be used")
	flag.Parse()
}

func main() {

	// defaults that won't be set corretly if omitted in config file
	config := config{
		Timeformat:      time.Kitchen,
		Maxlines:        1000,
		ScrollingSpeed:  1,
		PageUpDownSpeed: 10,
	}

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Fatalf("malformed configuration file: %v\n", err)
	}

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)
	g.Mouse = true

	chat, err := newChat(&config, g)
	if err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyF1, gocui.ModNone, chat.showHelp); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyF12, gocui.ModNone, chat.showDebug); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyF2, gocui.ModNone, chat.showUserList); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("input", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		err = chat.historyUp(g, v)
		return err
	}); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("input", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		err = chat.historyDown(g, v)
		return err
	}); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyPgup, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		err = scroll(-chat.config.PageUpDownSpeed, chat, "messages")
		return err
	}); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyPgdn, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		err = scroll(chat.config.PageUpDownSpeed, chat, "messages")
		return err
	}); err != nil {
		log.Panicln(err)
	}

	chat.mustAddScroll("messages", chat.config.ScrollingSpeed, gocui.MouseWheelUp, gocui.MouseWheelDown)
	chat.mustAddScroll("users", chat.config.ScrollingSpeed, gocui.MouseWheelUp, gocui.MouseWheelDown)
	chat.mustAddScroll("help", 1, gocui.MouseWheelUp, gocui.MouseWheelDown)
	chat.mustAddScroll("debug", 1, gocui.MouseWheelUp, gocui.MouseWheelDown)

	err = g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		if v.Buffer() == "" {
			return nil
		}

		chat.handleInput(strings.TrimSpace(v.Buffer()))
		g.Update(func(g *gocui.Gui) error {
			v.Clear()
			v.SetCursor(0, 0)
			v.SetOrigin(0, 0)
			return nil
		})

		return nil
	})

	if err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("input", gocui.KeyTab, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		chat.tabComplete(v)
		return nil
	}); err != nil {
		log.Panicln(err)
	}

	chat.Session.AddNamesHandler(func(n dggchat.Names, s *dggchat.Session) {
		chat.renderCommand("Connected!")
		chat.renderUsers(n.Users)
	})
	chat.Session.AddSocketErrorHandler(func(err error, s *dggchat.Session) {
		chat.renderError(err.Error() + " - Trying to reconnect...")
	})
	chat.Session.AddMessageHandler(func(m dggchat.Message, s *dggchat.Session) {
		chat.renderMessage(m)
	})
	chat.Session.AddMessageHandler(func(m dggchat.Message, s *dggchat.Session) {
		chat.renderMessage(m)
	})
	chat.Session.AddErrorHandler(func(e string, s *dggchat.Session) {
		chat.renderError(e)
	})
	chat.Session.AddMuteHandler(func(m dggchat.Mute, s *dggchat.Session) {
		chat.renderMute(m)
	})
	chat.Session.AddUnmuteHandler(func(m dggchat.Mute, s *dggchat.Session) {
		chat.renderUnmute(m)
	})
	chat.Session.AddBanHandler(func(b dggchat.Ban, s *dggchat.Session) {
		chat.renderBan(b)
	})
	chat.Session.AddUnbanHandler(func(b dggchat.Ban, s *dggchat.Session) {
		chat.renderUnban(b)
	})
	chat.Session.AddJoinHandler(func(r dggchat.RoomAction, s *dggchat.Session) {
		chat.renderJoin(r)
		chat.renderUsers(chat.Session.GetUsers())
	})
	chat.Session.AddQuitHandler(func(r dggchat.RoomAction, s *dggchat.Session) {
		chat.renderQuit(r)
		chat.renderUsers(chat.Session.GetUsers())
	})
	chat.Session.AddSubOnlyHandler(func(so dggchat.SubOnly, s *dggchat.Session) {
		chat.renderSubOnly(so)
	})
	chat.Session.AddBroadcastHandler(func(b dggchat.Broadcast, s *dggchat.Session) {
		chat.renderBroadcast(b)
	})
	chat.Session.AddPMHandler(func(pm dggchat.PrivateMessage, s *dggchat.Session) {
		chat.renderPrivateMessage(pm)
	})
	chat.Session.AddPingHandler(func(p dggchat.Ping, s *dggchat.Session) {
		_ = p.Timestamp //TODO
	})

	err = chat.Session.Open()
	if err != nil {
		// Most common problem is that the connection couldn't be established.
		log.Panicln(err)
	}
	defer chat.Session.Close()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (cfg *config) save() error {
	buf := bytes.NewBuffer([]byte{})
	err := toml.NewEncoder(buf).Encode(&cfg)
	if err != nil {
		return fmt.Errorf("error marshaling config: %v", err)
	}

	err = ioutil.WriteFile(configFile, buf.Bytes(), 0600)
	if err != nil {
		return fmt.Errorf("error saving config: %v", err)
	}
	return nil
}

func (c *chat) mustAddScroll(view string, speed int, up gocui.Key, down gocui.Key) {
	var err error
	err = c.guiwrapper.gui.SetKeybinding(view, down, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return scroll(speed, c, view)
	})
	if err != nil {
		log.Panicln(err)
	}
	err = c.guiwrapper.gui.SetKeybinding(view, up, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return scroll(-speed, c, view)
	})
	if err != nil {
		log.Panicln(err)
	}
}
