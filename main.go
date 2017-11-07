package main

import (
	"log"

	"github.com/jroimartin/gocui"
)

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalln(err)
	}

	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	// u := url.URL{Scheme: "wss", Host: "www.destiny.gg", Path: "/ws"}
	// h := make(http.Header, 0)
	// h.Set("Cookie", "'authtoken=")

	// c, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	// if err != nil {
	// 	log.Fatal("dial:", err)
	// }
	// defer c.Close()

	// go func() {
	// 	defer c.Close()
	// 	for {
	// 		_, message, err := c.ReadMessage()

	// 		if err != nil {
	// 			log.Println("read:", err)
	// 			return
	// 		}
	// 		log.Printf("recv: %s", message)
	// 	}
	// }()

	// for {
	// 	select {
	// 	case <-interrupt:
	// 		log.Println("got interrupt")
	// 		return
	// 	}
	// }

}
