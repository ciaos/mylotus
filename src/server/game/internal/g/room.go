package g

const (
	ROOM_STATUS_NONE     = 1
	ROOM_SYNC_PLAYERINFO = 2
	ROOM_CONNECTING      = 3
	ROOM_FIGHTING        = 4
	ROOM_END             = 5
)

type Room struct {
}

var PlayerRoomIDMap = make(map[string]int32)
var RoomMap = make(map[int32]*Room)
