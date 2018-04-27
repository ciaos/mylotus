package internal

import (
	"fmt"
	"server/msg/clientmsg"
	"strconv"
	"strings"
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
	register("delhero", "delhero [charid] [chartypeid]", gmDelHero)
	register("addcash", "addcash [charid] [cashtype] [cashnum]", gmAddCash)
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

	player, _ := GetPlayer(uint32(charid))
	player.GetPlayerAsset().AssetHero_AddHero(uint32(charid), uint32(chartypeid), int64(deadlinetime))
	return "done"
}

func gmDelHero(args []interface{}) interface{} {
	help := "delhero [charid] [chartypeid]"
	if len(args) != 2 {
		return help
	}

	charid, _ := strconv.Atoi(args[0].(string))
	chartypeid, _ := strconv.Atoi(args[1].(string))
	if charid == 0 || chartypeid == 0 {
		return help
	}

	player, _ := GetPlayer(uint32(charid))
	player.GetPlayerAsset().AssetHero_DelHero(uint32(charid), uint32(chartypeid))
	return "done"
}

func gmAddCash(args []interface{}) interface{} {
	help := "addcash [charid] [cashtype] [cashnum]"
	if len(args) != 3 {
		return help
	}

	charid, _ := strconv.Atoi(args[0].(string))
	tcashtype, _ := strconv.Atoi(args[1].(string))
	cashtype := clientmsg.Type_CashType(tcashtype)
	cashnum, _ := strconv.Atoi(args[2].(string))
	if charid == 0 || cashnum == 0 {
		return help
	}

	player, _ := GetPlayer(uint32(charid))

	switch cashtype {
	case clientmsg.Type_CashType_TCT_DIAMOND:
		player.GetPlayerAsset().AssetCash_AddDiamondCoin(cashnum)
	case clientmsg.Type_CashType_TCT_EXP:
		player.GetPlayerAsset().AssetCash_AddExp(uint32(cashnum))
	case clientmsg.Type_CashType_TCT_GOLD:
		player.GetPlayerAsset().AssetCash_AddGoldCoin(cashnum)
	case clientmsg.Type_CashType_TCT_SILVER:
		player.GetPlayerAsset().AssetCash_AddSilverCoin(cashnum)
	default:
		return help
	}
	return "done"
}
