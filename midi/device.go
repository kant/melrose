package midi

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/emicklei/melrose/core"

	"github.com/emicklei/melrose/notify"
	"github.com/rakyll/portmidi"
)

// Midi is an melrose.AudioDevice
type Midi struct {
	enabled      bool
	stream       *portmidi.Stream
	deviceID     int
	echo         bool // TODO remove
	sustainPedal *SustainPedal

	defaultOutputChannel  int
	currentOutputDeviceID int
	currentInputDeviceID  int

	timeline *core.Timeline
}

type MIDIWriter interface {
	WriteShort(int64, int64, int64) error
	Close() error
	Abort() error
}

// https://www.midi.org/specifications-old/item/table-1-summary-of-midi-message
const (
	noteOn        int64 = 0x90 // 10010000 , 144
	noteOff       int64 = 0x80 // 10000000 , 128
	controlChange int64 = 176  // 10110000 , 176
	noteAllOff    int64 = 123  // 01111011 , 123
)

var (
	DefaultChannel = 1
)

func (m *Midi) Reset() {
	m.timeline.Reset()
	if m.stream != nil {
		// send note off all to all channels for current device
		for c := 1; c <= 16; c++ {
			if err := m.stream.WriteShort(controlChange|int64(c-1), noteAllOff, 0); err != nil {
				fmt.Println("portmidi write error:", err)
			}
		}
	}
	m.sustainPedal.Reset()
}

func (m *Midi) Timeline() *core.Timeline { return m.timeline }

// SetEchoNotes is part of melrose.AudioDevice
func (m *Midi) SetEchoNotes(echo bool) {
	m.echo = echo
}

// Command is part of melrose.AudioDevice
func (m *Midi) Command(args []string) notify.Message {
	if len(args) == 0 {
		m.printInfo()
		return nil
	}
	switch args[0] {
	case "echo":
		echoMIDISent = !echoMIDISent
		return notify.Infof("printing notes enabled:%v", echoMIDISent)
	case "channel":
		if len(args) != 2 {
			return notify.Warningf("missing channel number")
		}
		nr, err := strconv.Atoi(args[1])
		if err != nil {
			return notify.Errorf("bad channel number:%v", err)
		}
		if nr < 1 || nr > 16 {
			return notify.Errorf("bad channel number; must be in [1..16]")
		}
		m.defaultOutputChannel = nr
		return nil
	case "in":
		if len(args) != 2 {
			return notify.Warningf("missing device number")
		}
		nr, err := strconv.Atoi(args[1])
		if err != nil {
			return notify.Errorf("bad device number:%v", err)
		}
		m.currentInputDeviceID = nr
		return notify.Infof("Current input device id:%v", m.currentInputDeviceID)
	case "out":
		if len(args) != 2 {
			return notify.Warningf("missing device number")
		}
		nr, err := strconv.Atoi(args[1])
		if err != nil {
			return notify.Errorf("bad device number:%v", err)
		}
		if err := m.changeOutputDeviceID(nr); err != nil {
			return err
		}
		return notify.Infof("Current output device id:%v", m.currentOutputDeviceID)
	default:
		return notify.Warningf("unknown midi command: %s", args[0])
	}
}

func (m *Midi) printInfo() {
	fmt.Println("Usage:")
	fmt.Println(":m echo                --- toggle printing the notes that are send")
	fmt.Println(":m inp     <device-id> --- change the current MIDI input device id")
	fmt.Println(":m out     <device-id> --- change the current MIDI output device id")
	fmt.Println(":m channel <1..16>     --- change the default MIDI output channel")
	fmt.Println()

	var midiDeviceInfo *portmidi.DeviceInfo
	defaultOut := portmidi.DefaultOutputDeviceID()
	defaultIn := portmidi.DefaultInputDeviceID()

	for i := 0; i < portmidi.CountDevices(); i++ {
		midiDeviceInfo = portmidi.Info(portmidi.DeviceID(i)) // returns info about a MIDI device
		fmt.Printf("[midi] device id = %d: ", i)
		usage := "output"
		if midiDeviceInfo.IsInputAvailable {
			usage = "input"
		}
		oc := "open"
		if !midiDeviceInfo.IsOpened {
			oc = "closed"
		}
		fmt.Print("\"", midiDeviceInfo.Interface, "/", midiDeviceInfo.Name, "\"",
			", is ", oc,
			", use for ", usage)
		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("[midi] %v = echo notes\n", m.echo)
	fmt.Printf("[midi] %d = input  device id (default = %d)\n", m.currentInputDeviceID, defaultIn)
	fmt.Printf("[midi] %d = output device id (default = %d)\n", m.currentOutputDeviceID, defaultOut)
	fmt.Printf("[midi] %d = default output channel\n", m.defaultOutputChannel)
}

func Open() (*Midi, error) {
	m := new(Midi)
	m.sustainPedal = NewSustainPedal()
	portmidi.Initialize()
	deviceID := portmidi.DefaultOutputDeviceID()
	if deviceID == -1 {
		return nil, errors.New("no default output MIDI device available")
	}
	m.enabled = true
	m.echo = false
	// for output
	m.defaultOutputChannel = DefaultChannel
	m.changeOutputDeviceID(int(portmidi.DefaultOutputDeviceID()))

	// start timeline
	m.timeline = core.NewTimeline()
	go m.timeline.Play()

	return m, nil
}

func (m *Midi) changeOutputDeviceID(id int) notify.Message {
	if !m.enabled {
		return notify.Warningf("MIDI is not enabled")
	}
	if m.currentOutputDeviceID == id {
		// check stream
		if m.stream != nil {
			return nil
		}
	}
	// open new
	out, err := portmidi.NewOutputStream(portmidi.DeviceID(id), 1024, 0)
	if err != nil {
		return notify.Error(err)
	}
	if m.stream != nil {
		// close old stream
		m.stream.Close()
	}
	m.stream = out
	m.currentOutputDeviceID = id
	return nil
}

// Close is part of melrose.AudioDevice
func (m *Midi) Close() {
	if m.timeline != nil {
		m.timeline.Reset()
	}
	if m.enabled {
		m.stream.Abort()
		m.stream.Close()
	}
	portmidi.Terminate()
}

// 93 is bright yellow
func print(arg interface{}) {
	fmt.Printf("\033[2;93m" + fmt.Sprintf("%v ", arg) + "\033[0m")
}

func info(arg interface{}) {
	fmt.Printf("\033[2;33m" + fmt.Sprintf("%v ", arg) + "\033[0m")
}
