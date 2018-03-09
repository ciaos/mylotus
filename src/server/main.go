package main

import (
	"fmt"
	"server/conf"
	"server/game"
	"server/gate"
	"server/login"
	"server/queue"
	"server/tick"

	"github.com/ciaos/leaf"
	lconf "github.com/ciaos/leaf/conf"
)

func main() {
	lconf.LogLevel = conf.Server.LogLevel
	lconf.LogPath = conf.Server.LogPath
	lconf.LogFlag = conf.LogFlag
	lconf.ConsolePort = conf.Server.ConsolePort
	lconf.ProfilePath = conf.Server.ProfilePath

	fmt.Println("Rose Start...")
	leaf.Run(
		game.Module,
		gate.Module,
		login.Module,
		tick.Module,
		queue.Module,
	)
}
