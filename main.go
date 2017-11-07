package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

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

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	chat := newChat(&config)
	defer chat.connection.Close()

	go chat.listen()

	go func() {
		for {
			message := <-chat.messages
			renderMessage(g, message)

		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}

}
