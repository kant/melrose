package main

import (
	"github.com/emicklei/melrose"
	"github.com/emicklei/melrose/notify"
)

var cmdFuncMap = cmdFunctions()

func cmdFunctions() map[string]Command {
	cmds := map[string]Command{}
	cmds[":h"] = Command{Description: "show help, optional on a command or function", Func: showHelp}
	cmds[":s"] = Command{Description: "save memory to disk, optional use given filename", Func: varStore.SaveMemoryToDisk}
	cmds[":l"] = Command{Description: "load memory from disk, optional use given filename", Func: varStore.LoadMemoryFromDisk}
	cmds[":v"] = Command{Description: "show variables, optional filter on given prefix", Func: varStore.ListVariables}
	cmds[":m"] = Command{Description: "show MIDI information", Func: ShowDeviceInfo}
	cmds[":q"] = Command{Description: "quit"} // no Func because it is handled in the main loop
	return cmds
}

type Command struct {
	Description string
	Sample      string
	Func        func(args []string) notify.Message
}

func lookupCommand(args []string) (Command, bool) {
	if len(args) == 0 {
		return Command{}, false
	}
	if cmd, ok := cmdFuncMap[args[0]]; ok {
		return cmd, true
	}
	return Command{}, false
}

func ShowDeviceInfo(args []string) notify.Message {
	// TODO
	melrose.CurrentDevice().PrintInfo()
	return nil
}
