package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
)

type config struct {
	DGGKey string `json:"dgg_key"`
}

func main() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err)
	}

	var config config
	json.Unmarshal(file, &config)
	if config.DGGKey == "" {
		log.Fatalln("malformed configuration file")
	}

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)
	g.Mouse = false

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	chat := newChat(&config, g)
	defer chat.connection.Close()

	err = g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		chat.sendMessage(strings.TrimSpace(v.Buffer()))
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

	go chat.listen()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}

}
