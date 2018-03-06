package g

import (
	"time"
)

const (
	ROOM_STATUS_NONE     = 1
	ROOM_SYNC_PLAYERINFO = 2
	ROOM_CONNECTING      = 3
	ROOM_FIGHTING        = 4
	ROOM_END             = 5
)

type Room struct {
	createtime int64
	status     int32
}

var PlayerRoomIDMap = make(map[string]int32)
var RoomManager = make(map[int32]*Room)

func (room *Room) update(now *time.Time) int32 {
	return ROOM_END
}

func UpdateRoomManager(now *time.Time) {
	for _, room := range RoomManager {
		(*room).status = (*room).update(now)
	}
}
