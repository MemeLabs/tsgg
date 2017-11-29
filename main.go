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

	messages := make(chan dggchat.Message)
	errors := make(chan string)
	pings := make(chan dggchat.Ping)
	mutes := make(chan dggchat.Mute)
	unmutes := make(chan dggchat.Mute)
	bans := make(chan dggchat.Ban)
	unbans := make(chan dggchat.Ban)
	joins := make(chan dggchat.RoomAction)
	quits := make(chan dggchat.RoomAction)
	subonly := make(chan dggchat.SubOnly)
	broadcasts := make(chan dggchat.Broadcast)

	chat.Session.AddMessageHandler(func(m dggchat.Message, s *dggchat.Session) {
		messages <- m
	})
	chat.Session.AddErrorHandler(func(e string, s *dggchat.Session) {
		errors <- e
	})
	chat.Session.AddMuteHandler(func(m dggchat.Mute, s *dggchat.Session) {
		mutes <- m
	})
	chat.Session.AddUnmuteHandler(func(m dggchat.Mute, s *dggchat.Session) {
		unmutes <- m
	})
	chat.Session.AddBanHandler(func(b dggchat.Ban, s *dggchat.Session) {
		bans <- b
	})
	chat.Session.AddUnbanHandler(func(b dggchat.Ban, s *dggchat.Session) {
		unbans <- b
	})
	chat.Session.AddJoinHandler(func(r dggchat.RoomAction, s *dggchat.Session) {
		joins <- r
	})
	chat.Session.AddQuitHandler(func(r dggchat.RoomAction, s *dggchat.Session) {
		quits <- r
	})
	chat.Session.AddSubOnlyHandler(func(so dggchat.SubOnly, s *dggchat.Session) {
		subonly <- so
	})
	chat.Session.AddBroadcastHandler(func(b dggchat.Broadcast, s *dggchat.Session) {
		broadcasts <- b
	})
	chat.Session.AddPingHandler(func(p dggchat.Ping, s *dggchat.Session) {
		pings <- p
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

	go func() { //TODO
		for {
			select {
			case m := <-messages:
				chat.renderMessage(m)
			case e := <-errors:
				chat.renderError(e)
			case p := <-pings:
				_ = p.Timestamp //TODO
			case m := <-mutes:
				chat.renderMute(m)
			case m := <-unmutes:
				chat.renderUnmute(m)
			case b := <-bans:
				chat.renderBan(b)
			case b := <-unbans:
				chat.renderUnban(b)
			case j := <-joins:
				if chat.config.ShowJoinLeave {
					chat.renderJoin(j)
				}
				chat.renderUsers(chat.Session.GetUsers())
			case j := <-quits:
				if chat.config.ShowJoinLeave {
					chat.renderQuit(j)
				}
				chat.renderUsers(chat.Session.GetUsers())
			case so := <-subonly:
				chat.renderSubOnly(so)
			case b := <-broadcasts:
				chat.renderBroadcast(b)
			}
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}

}
