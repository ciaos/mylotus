package g

import (
	"fmt"
	"server/tool"
	"strings"
	"time"
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
	charname string
	chartype int32
	teamtype int32
}

type Room struct {
	createtime int64
	status     int32
	roomid     int32
	matchmode  int32
	battlekey  []byte

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

	for _, room := range RoomManager {
		(*room).status = (*room).update(now)
	}
}

func CreateRoom(matchmode int32) int32 {
	roomid += 1
	if roomid > MAX_ROOM_COUNT {
		roomid = 1
	}

	battlekey, _ := tool.DesEncrypt([]byte(fmt.Sprintf("room%d", roomid)), []byte(tool.CRYPT_KEY))

	room := &Room{
		roomid:     roomid,
		createtime: time.Now().Unix(),
		status:     ROOM_STATUS_NONE,
		matchmode:  matchmode,
		battlekey:  battlekey,
		members:    make(map[string]*Member),
	}

	RoomManager[roomid] = room
	return roomid
}

func JoinRoom(charid string, roomid int32, charname string, chartype int32) []byte {
	room, ok := RoomManager[roomid]
	if ok {
		member, ok := room.members[charid]
		if !ok {
			member = &Member{
				charid:   charid,
				teamtype: 0,
				charname: charname,
				chartype: chartype,
			}
			room.members[charid] = member
			return room.battlekey
		}
	}

	return nil
}

func FormatRoomInfo(roomid int32) string {
	room, ok := RoomManager[roomid]
	if ok {
		return fmt.Sprintf("Roomid:%v\tCreateTime:%v\tStatus:%v\tMemberCnt:%v", (*room).roomid, (*room).createtime, (*room).status, len((*room).members))
	}
	return ""
}

func FormatMemberInfo(roomid int32) string {
	output := fmt.Sprintf("RoomID %v", roomid)
	room, ok := RoomManager[roomid]
	if ok {
		for _, member := range (*room).members {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tCharType:%v\tTeamType:%v", (*member).charid, (*member).charname, (*member).chartype, (*member).teamtype)}, "\r\n")
		}
	}
	return output
}
