package midi

import (
	"fmt"
	"time"

	"github.com/emicklei/melrose/core"
	"github.com/emicklei/melrose/notify"
	"github.com/rakyll/portmidi"
)

type listener struct {
	stream *portmidi.Stream
	quit   chan bool
	noteOn map[int]portmidi.Event
	ctx    core.Context
}

func newListener(ctx core.Context) *listener {
	return &listener{
		noteOn: map[int]portmidi.Event{},
		ctx:    ctx,
	}
}

func (l *listener) listen() {
	l.quit = make(chan bool)
	ch := l.stream.Listen()
	for {
		select {
		case <-l.quit:
			goto stop
		case e := <-ch:
			l.handle(e)
		}
	}
stop:
	close(l.quit)
}

func (l *listener) handle(event portmidi.Event) {
	nr := int(event.Data1)
	if event.Status == noteOn {
		if _, ok := l.noteOn[nr]; ok {
			return
		}
		l.noteOn[nr] = event
	} else if event.Status == noteOff {
		on, ok := l.noteOn[nr]
		if !ok {
			return
		}
		delete(l.noteOn, nr)
		frac := core.DurationToFraction(l.ctx.Control().BPM(), time.Duration(event.Timestamp-on.Timestamp)*time.Millisecond)
		note := core.MIDItoNote(frac, nr, int(on.Data2))
		fmt.Fprintf(notify.Console.DeviceIn, " %s", note)
	}
}

func (l *listener) stop() {
	// forget open notes
	l.noteOn = map[int]portmidi.Event{}
	l.quit <- true
}