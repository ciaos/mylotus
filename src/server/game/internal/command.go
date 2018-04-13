package internal

import (
	"fmt"
	"runtime/debug"
	"server/game/g"
	"strconv"
	"strings"
)

func init() {
	InitGM()

	skeleton.RegisterCommand("free", "free heap memory", commandFree)
	skeleton.RegisterCommand("gm", "gm command", commandGM)
	skeleton.RegisterCommand("r", "list room info", commandRoom)
	skeleton.RegisterCommand("rn", "room count", commandRoomCount)
	skeleton.RegisterCommand("rm", "list charid roomid map", commandRoomMap)
	skeleton.RegisterCommand("t", "list table info", commandTable)
	skeleton.RegisterCommand("tn", "table count", commandTableCount)
	skeleton.RegisterCommand("tm", "list charid tableid map", commandTableMap)
	skeleton.RegisterCommand("g", "list gameserver online member count", commandGPlayer)
	skeleton.RegisterCommand("gn", "gameserver online member count", commandGPlayerCount)
	skeleton.RegisterCommand("b", "list battleserver online member count", commandBPlayer)
	skeleton.RegisterCommand("bn", "battleserver online member count", commandBPlayerCount)
}

func commandFree(args []interface{}) interface{} {
	debug.FreeOSMemory()
	return "OK"
}

func commandGM(args []interface{}) interface{} {
	return RunGMCmd(args)
}

func commandRoom(args []interface{}) interface{} {
	if len(args) == 1 {
		roomid, _ := strconv.Atoi(args[0].(string))
		return g.FormatMemberInfo(int32(roomid))
	} else {
		var output string
		for i, _ := range g.RoomManager {
			output = strings.Join([]string{output, g.FormatRoomInfo(i)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(g.RoomManager), len(g.PlayerRoomIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandRoomCount(args []interface{}) interface{} {
	return fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(g.RoomManager), len(g.PlayerRoomIDMap))
}

func commandRoomMap(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		output := fmt.Sprintf("CharID:%v\tRoomID:%v", uint32(charid), g.PlayerRoomIDMap[uint32(charid)])
		return output
	} else {
		var output string
		for k, v := range g.PlayerRoomIDMap {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tRoomID:%v", k, v)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(g.RoomManager), len(g.PlayerRoomIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandTableMap(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		output := fmt.Sprintf("CharID:%v\tTableID:%v", uint32(charid), g.PlayerTableIDMap[uint32(charid)])
		return output
	} else {
		var output string
		for k, v := range g.PlayerTableIDMap {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tTableID:%v", k, v)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("TableCnt:%v TablePlayerTotal:%v", len(g.TableManager), len(g.PlayerTableIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandTableCount(args []interface{}) interface{} {
	return fmt.Sprintf("TableCnt:%v TablePlayerTotal:%v", len(g.TableManager), len(g.PlayerTableIDMap))
}

func commandTable(args []interface{}) interface{} {
	if len(args) == 1 {
		tableid, _ := strconv.Atoi(args[0].(string))
		return g.FormatSeatInfo(int32(tableid))
	} else {
		output := fmt.Sprintf("TableCnt:%v TablePlayerTotal:%v", len(g.TableManager), len(g.PlayerTableIDMap))
		for i, _ := range g.TableManager {
			output = strings.Join([]string{output, g.FormatTableInfo(i)}, "\r\n")
		}
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandGPlayer(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		return g.FormatOneGPlayerInfo(uint32(charid), "")
	} else if len(args) == 2 {
		charid, _ := strconv.Atoi(args[0].(string))
		return g.FormatOneGPlayerInfo(uint32(charid), args[1].(string))
	} else {
		return g.FormatGPlayerInfo()
	}
}

func commandGPlayerCount(args []interface{}) interface{} {
	return fmt.Sprintf("GamePlayerCnt:%d", len(g.GamePlayerManager))
}

func commandBPlayer(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		return g.FormatOneBPlayerInfo(uint32(charid))
	} else {
		return g.FormatBPlayerInfo()
	}
}

func commandBPlayerCount(args []interface{}) interface{} {
	return fmt.Sprintf("BattlePlayerCnt:%d", len(g.BattlePlayerManager))
}
