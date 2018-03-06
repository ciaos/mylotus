package g

import (
	"time"

	"github.com/name5566/leaf/log"
)

const (
	ROOM_STATUS_NONE     = 1
	ROOM_SYNC_PLAYERINFO = 2
	ROOM_CONNECTING      = 3
	ROOM_FIGHTING        = 4
	ROOM_END             = 5
)

type Member struct {
	charid   string
	teamtype int32
}

type Room struct {
	createtime int64
	status     int32
	roomid     int32
	matchmode  int32

	members map[string]*Member
}

var PlayerRoomIDMap = make(map[string]int32)
var RoomManager = make(map[int32]*Room)
var roomid int32

func InitRoomManager() {
	roomid = 0
}

func (room *Room) update(now *time.Time) int32 {
	return ROOM_END
}

func UpdateRoomManager(now *time.Time) {
	log.Debug("UpdateRoomManager %v", len(RoomManager))
	for _, room := range RoomManager {
		(*room).status = (*room).update(now)
	}
}

func CreateRoom(matchmode int32) int32 {
	roomid += 1
	if roomid > MAX_ROOM_COUNT {
		roomid = 1
	}

	room := &Room{
		roomid:     roomid,
		createtime: time.Now().Unix(),
		status:     ROOM_STATUS_NONE,
		matchmode:  matchmode,
	}

	RoomManager[roomid] = room
	return roomid
}

func JoinRoom(charid string, roomid int32) bool {
	room, ok := RoomManager[roomid]
	if ok {
		member, ok := room.members[charid]
		if !ok {
			member = &Member{
				charid:   charid,
				teamtype: 0,
			}
			room.members[charid] = member
			return true
		}
	}

	return false
}
