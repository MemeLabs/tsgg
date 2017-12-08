package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
)

type guiwrapper struct {
	gui        *gocui.Gui
	messages   []*guimessage
	maxlines   int
	timeformat string
	sync.RWMutex
}

type guimessage struct {
	ts   time.Time
	tag  string // TODO assert len=3 ?
	msg  string
	nick string // TODO only set this when we want to modify tag/highlight/ignore/...? later on... not on system messages; even though they technically have a sender too ...
}

func (gw *guiwrapper) formatMessage(gm *guimessage) string {
	formattedDate := gm.ts.Format(gw.timeformat)
	return fmt.Sprintf("[%s]%s%s", formattedDate, gm.tag, gm.msg)
}

func (gw *guiwrapper) redraw() {
	gw.gui.Update(func(g *gocui.Gui) error {
		messageView, err := g.View("messages")
		if err != nil {
			return err
		}

		gw.Lock()
		defer gw.Unlock()

		// redraw everything
		newbuf := ""
		for _, msg := range gw.messages {
			newbuf += gw.formatMessage(msg) + "\n"
		}

		messageView.Clear()
		fmt.Fprint(messageView, newbuf)
		return nil
	})
}

func (gw *guiwrapper) addMessage(m guimessage) {

	gw.Lock()
	// only keep maxlines lines in memory
	if len(gw.messages) > gw.maxlines {
		gw.messages = append(gw.messages[1:], &m)
	} else {
		gw.messages = append(gw.messages, &m)
	}
	gw.Unlock()
	gw.redraw()
}

func (gw *guiwrapper) applyTag(tag string, nick string) {

	gw.Lock()
	defer gw.Unlock()

	for _, m := range gw.messages {
		if m.nick != "" && strings.ToLower(m.nick) == strings.ToLower(nick) {
			m.tag = tag
		}
	}
}
