package main

import (
	"github.com/name5566/leaf"
	lconf "github.com/name5566/leaf/conf"
	"server/conf"
	"server/game"
	"server/gate"
	"server/login"
	"server/db"
	"server/tools"
)

func main() {
	
	lconf.LogLevel = conf.Server.LogLevel
	lconf.LogPath = conf.Server.LogPath
	lconf.LogFlag = conf.LogFlag
	lconf.ConsolePort = conf.Server.ConsolePort
	lconf.ProfilePath = conf.Server.ProfilePath
    tools.CreateGateUserData()
	leaf.Run(
		game.Module,
		gate.Module,
		login.Module,
		db.Module,
	)
	
}
