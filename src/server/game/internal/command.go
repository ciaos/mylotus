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
	skeleton.RegisterCommand("lroomcnt", "list room cnt", commandRoomCount)
	skeleton.RegisterCommand("ltable", "list table info", commandTable)
	skeleton.RegisterCommand("ltablecnt", "list table cnt", commandTableCount)
	skeleton.RegisterCommand("lgplayer", "list gameserver online member count", commandGPlayer)
	skeleton.RegisterCommand("lbplayer", "list battleserver online member count", commandBPlayer)
}

func commandEcho(args []interface{}) interface{} {
	return fmt.Sprintf("%v", args)
}

func commandRoom(args []interface{}) interface{} {
	if len(args) == 1 {
		roomid, _ := strconv.Atoi(args[0].(string))
		return g.FormatMemberInfo(int32(roomid))
	} else {
		output := fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(g.RoomManager), len(g.PlayerRoomIDMap))
		for i, _ := range g.RoomManager {
			output = strings.Join([]string{output, g.FormatRoomInfo(i)}, "\r\n")
		}
		return output
	}
}

func commandRoomCount(args []interface{}) interface{} {
	output := fmt.Sprintf("RoomCnt:%v RoomPlayerTotal:%v", len(g.RoomManager), len(g.PlayerRoomIDMap))
	return output
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
		return output
	}
}

func commandTableCount(args []interface{}) interface{} {
	output := fmt.Sprintf("TableCnt:%v TablePlayerTotal:%v", len(g.TableManager), len(g.PlayerTableIDMap))
	return output
}

func commandGPlayer(args []interface{}) interface{} {
	return g.FormatGPlayerInfo()
}

func commandBPlayer(args []interface{}) interface{} {
	return g.FormatBPlayerInfo()
}
