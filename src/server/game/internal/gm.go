package internal

import (
	"fmt"
	"strconv"
	"strings"

	"server/game/g"
)

type gmCommand struct {
	_name string
	_help string
	f     interface{}
}

func RunGMCmd(args []interface{}) interface{} {
	if len(args) == 0 {
		return help()
	} else {
		return run(args)
	}
}

var gm_cmd = map[string]*gmCommand{}

func register(name string, help string, f interface{}) {
	c := new(gmCommand)
	c._name = name
	c._help = help
	c.f = f
	gm_cmd[name] = c
}

func InitGM() {
	register("help", "list gm command", gmHelp)
	register("echo", "echo [string]", gmEcho)
	register("addhero", "addhero [charid] [chartypeid] [deadlinetime]", gmAddHero)
}

func help() interface{} {
	output := ""
	for _, command := range gm_cmd {
		output = strings.Join([]string{output, fmt.Sprintf("%v - %v", command._name, command._help)}, "\r\n")
	}
	return strings.TrimLeft(output, "\r\n")
}

func run(args []interface{}) interface{} {
	command, ok := gm_cmd[args[0].(string)]
	if !ok {
		return "gm command not found, try `help` for help"
	}

	return command.f.(func([]interface{}) interface{})(args[1:])
}

func gmHelp(args []interface{}) interface{} {
	return help()
}

func gmEcho(args []interface{}) interface{} {
	if len(args) == 0 {
		return "echo [string]"
	}
	return fmt.Sprintf("%v", args[0].(string))
}

func gmAddHero(args []interface{}) interface{} {
	help := "addhero [charid] [chartypeid] [deadlinetime]"
	if len(args) != 3 {
		return help
	}

	charid, _ := strconv.Atoi(args[0].(string))
	chartypeid, _ := strconv.Atoi(args[1].(string))
	deadlinetime, _ := strconv.Atoi(args[2].(string))
	if charid == 0 || chartypeid == 0 {
		return help
	}

	player, _ := g.GetPlayer(uint32(charid))
	player.GetPlayerAsset().AssetHero_AddHero(uint32(charid), uint32(chartypeid), int64(deadlinetime))
	return "done"
}
