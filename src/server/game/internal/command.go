package internal

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

func init() {
	InitGM()

	skeleton.RegisterCommand("free", "free heap memory", commandFree)
	skeleton.RegisterCommand("gm", "gm command", commandGM)
	skeleton.RegisterCommand("room", "list room info", commandRoom)
	skeleton.RegisterCommand("roommap", "list charid roomid map", commandRoomMap)
	skeleton.RegisterCommand("table", "list table info", commandTable)
	skeleton.RegisterCommand("tablemap", "list charid tableid map", commandTableMap)
	skeleton.RegisterCommand("bench", "list bench info", commandBench)
	skeleton.RegisterCommand("benchmap", "list charid benchid map", commandBenchMap)
	skeleton.RegisterCommand("gplayer", "list gameserver online member count", commandGPlayer)
	skeleton.RegisterCommand("bplayer", "list battleserver online member count", commandBPlayer)
	skeleton.RegisterCommand("bsinfo", "list battleserver info for matchserver", commandBSInfo)
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
		return FormatMemberInfo(int32(roomid))
	} else {
		var output string
		for i, _ := range RoomManager {
			output = strings.Join([]string{output, FormatRoomInfo(i)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(RoomManager), len(PlayerRoomIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandRoomMap(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		output := fmt.Sprintf("CharID:%v\tRoomID:%v", uint32(charid), PlayerRoomIDMap[uint32(charid)])
		return output
	} else {
		var output string
		for k, v := range PlayerRoomIDMap {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tRoomID:%v", k, v)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(RoomManager), len(PlayerRoomIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandTable(args []interface{}) interface{} {
	if len(args) == 1 {
		tableid, _ := strconv.Atoi(args[0].(string))
		return FormatSeatInfo(int32(tableid))
	} else {
		output := fmt.Sprintf("TableCnt:%v TablePlayerTotal:%v", len(TableManager), len(PlayerTableIDMap))
		for i, _ := range TableManager {
			output = strings.Join([]string{output, FormatTableInfo(i)}, "\r\n")
		}
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandTableMap(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		output := fmt.Sprintf("CharID:%v\tTableID:%v", uint32(charid), PlayerTableIDMap[uint32(charid)])
		return output
	} else {
		var output string
		for k, v := range PlayerTableIDMap {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tTableID:%v", k, v)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("TableCnt:%v TablePlayerTotal:%v", len(TableManager), len(PlayerTableIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandBench(args []interface{}) interface{} {
	if len(args) == 1 {
		benchid, _ := strconv.Atoi(args[0].(string))
		return FormatUnitInfo(int32(benchid))
	} else {
		output := fmt.Sprintf("BenchCnt:%v BenchPlayerTotal:%v", len(BenchManager), len(PlayerBenchIDMap))
		for i, _ := range BenchManager {
			output = strings.Join([]string{output, FormatBenchInfo(i)}, "\r\n")
		}
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandBenchMap(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		output := fmt.Sprintf("CharID:%v\tBenchID:%v", uint32(charid), PlayerBenchIDMap[uint32(charid)])
		return output
	} else {
		var output string
		for k, v := range PlayerBenchIDMap {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tBenchID:%v", k, v)}, "\r\n")
		}
		output = strings.Join([]string{output, fmt.Sprintf("BenchCnt:%v BenchPlayerTotal:%v", len(BenchManager), len(PlayerBenchIDMap))}, "\r\n")
		return strings.TrimLeft(output, "\r\n")
	}
}

func commandGPlayer(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		return FormatOneGPlayerInfo(uint32(charid), "")
	} else if len(args) == 2 {
		charid, _ := strconv.Atoi(args[0].(string))
		return FormatOneGPlayerInfo(uint32(charid), args[1].(string))
	} else {
		return FormatGPlayerInfo()
	}
}

func commandBPlayer(args []interface{}) interface{} {
	if len(args) == 1 {
		charid, _ := strconv.Atoi(args[0].(string))
		return FormatOneBPlayerInfo(uint32(charid))
	} else {
		return FormatBPlayerInfo()
	}
}

func commandBSInfo(args []interface{}) interface{} {
	output := fmt.Sprintf("BattleServerCnt:%v", len(BSOnlineManager))
	for i, _ := range BSOnlineManager {
		output = strings.Join([]string{output, FormatBSOnline(i)}, "\r\n")
	}
	return strings.TrimLeft(output, "\r\n")
}