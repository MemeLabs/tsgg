package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/voloshink/dggchat"
)

type config struct {
	DGGKey        string   `json:"dgg_key"`
	CustomURL     string   `json:"custom_url"`
	Username      string   `json:"username"`
	Highlighted   []string `json:"highlighted"`
	ShowJoinLeave bool     `json:"showjoinleave"`
}

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.json", "location of config file to be used")
}

func main() {

	flag.Parse()

	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	var config config
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Println("malformed configuration file:")
		log.Fatalln(err)
	}

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)
	g.Mouse = false

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatalln(err)
	}

	chat, err := newChat(&config, g)
	if err != nil {
		log.Println(err)
		return
	}

	if err := g.SetKeybinding("input", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		historyUp(g, v, chat)
		return nil
	}); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("input", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		historyDown(g, v, chat)
		return nil
	}); err != nil {
		log.Panicln(err)
	}

	err = g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		if v.Buffer() == "" {
			return nil
		}

		chat.handleInput(strings.TrimSpace(v.Buffer()), g)
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
		if chat.config.ShowJoinLeave {
			chat.renderJoin(r)
		}
		chat.renderUsers(chat.Session.GetUsers())
	})
	chat.Session.AddQuitHandler(func(r dggchat.RoomAction, s *dggchat.Session) {
		if chat.config.ShowJoinLeave {
			chat.renderQuit(r)
		}
		chat.renderUsers(chat.Session.GetUsers())
	})
	chat.Session.AddSubOnlyHandler(func(so dggchat.SubOnly, s *dggchat.Session) {
		chat.renderSubOnly(so)
	})
	chat.Session.AddBroadcastHandler(func(b dggchat.Broadcast, s *dggchat.Session) {
		chat.renderBroadcast(b)
	})
	chat.Session.AddPingHandler(func(p dggchat.Ping, s *dggchat.Session) {
		_ = p.Timestamp //TODO
	})

	err = chat.Session.Open()
	if err != nil {
		log.Println(err)
		return
	}
	defer chat.Session.Close()

	// TODO need to wait for lib to receive first NAMES message to be properly "initialized"
	// maybe add a handler for this instead
	for {
		if len(chat.Session.GetUsers()) != 0 {
			break
		}
		time.Sleep(time.Millisecond * 300)
	}

	chat.renderUsers(chat.Session.GetUsers())

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}

}
