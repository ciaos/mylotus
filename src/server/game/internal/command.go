package internal

import (
	"fmt"
	"server/game/g"
	"strconv"
	"strings"
)

func init() {
	skeleton.RegisterCommand("echo", "echo user inputs", commandEcho)
	skeleton.RegisterCommand("lroom", "list room info", commandRoom)
	skeleton.RegisterCommand("ltable", "list table info", commandTable)
	skeleton.RegisterCommand("lseat", "list seat of specified table", commandSeat)
	skeleton.RegisterCommand("lmember", "list member of specified room", commandMember)
	skeleton.RegisterCommand("lgplayercount", "list gameserver online member count", commandGPlayerCount)
	skeleton.RegisterCommand("lbplayercount", "list battleserver online member count", commandBPlayerCount)
}

func commandEcho(args []interface{}) interface{} {
	return fmt.Sprintf("%v", args)
}

func commandRoom(args []interface{}) interface{} {
	output := fmt.Sprintf("RoomCnt %v", len(g.RoomManager))

	for i, _ := range g.RoomManager {
		output = strings.Join([]string{output, g.FormatRoomInfo(i)}, "\r\n")
	}

	return output
}

func commandTable(args []interface{}) interface{} {
	output := fmt.Sprintf("TableCnt %v", len(g.TableManager))

	for i, _ := range g.TableManager {
		output = strings.Join([]string{output, g.FormatTableInfo(i)}, "\r\n")
	}

	return output
}

func commandSeat(args []interface{}) interface{} {
	if len(args) == 1 {
		tableid, _ := strconv.Atoi(args[0].(string))
		return g.FormatSeatInfo(int32(tableid))
	}
	return ""
}

func commandMember(args []interface{}) interface{} {
	if len(args) == 1 {
		roomid, _ := strconv.Atoi(args[0].(string))
		return g.FormatMemberInfo(int32(roomid))
	}
	return ""
}

func commandGPlayerCount(args []interface{}) interface{} {
	return fmt.Sprintf("Player Count : %d", len(g.GamePlayerManager))
}

func commandBPlayerCount(args []interface{}) interface{} {
	return fmt.Sprintf("Player Count : %d", len(g.BattlePlayerManager))
}
