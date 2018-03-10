package g

import (
	"fmt"
	"server/tool"
	"strings"
	//	"sync"
	"time"

	"github.com/ciaos/leaf/log"
)

const (
	ROOM_STATUS_NONE     = 1
	ROOM_SYNC_PLAYERINFO = 2
	ROOM_CONNECTING      = 3
	ROOM_FIGHTING        = 4
	ROOM_END             = 5

	MEMBER_UNCONNECTED = 0
	MEMBER_CONNECTED   = 1
	MEMBER_OFFLINE     = 2
	MEMBER_END         = 3
)

type Member struct {
	charid       string
	charname     string
	chartype     int32
	teamtype     int32
	status       int32
	gameserverid int32
}

type Room struct {
	createtime    int64
	nextchecktime int64
	status        int32
	roomid        int32
	matchmode     int32
	battlekey     []byte

	membercnt int32
	memberok  int32

	members map[string]*Member

	messages       []interface{}
	messagesbackup []interface{}
}

var PlayerRoomIDMap = make(map[string]int32)
var RoomManager = make(map[int32]*Room)
var roomid int32

//var mRoomID *sync.Mutex

func InitRoomManager() {
	roomid = 0

	//mRoomID = new(sync.Mutex)
}

func allocRoomID() {
	//mRoomID.Lock()
	//defer mRoomID.Unlock()
	roomid += 1
	if roomid > MAX_ROOM_COUNT {
		roomid = 1
	}
}

func (room *Room) broadcast(msgdata interface{}) {
	for _, member := range (*room).members {
		agent, ok := BattlePlayerManager[(*member).charid]
		if ok {
			(*agent).WriteMsg(msgdata)
		}
	}
}

func (room *Room) update(now *time.Time) int32 {
	if (*room).status == ROOM_FIGHTING {
		msgCnt := len((*room).messages)
		if msgCnt > 0 {
			//log.Debug("RoomID %v BroadCast Message Count %v", (*room).roomid, msgCnt)

			for _, message := range (*room).messages {
				(*room).broadcast(message)
			}

			//(*room).messagesbackup = append((*room).messagesbackup, (*room).messages...)
			(*room).messages = append([]interface{}{})
		}
	}

	if (*room).nextchecktime < (*now).Unix() {
		(*room).nextchecktime = (*now).Unix() + 5

		if (*room).status == ROOM_STATUS_NONE {
			log.Error("ROOM_STATUS_NONE TimeOut %v", (*room).roomid)
			return ROOM_END
		}

		var allOffLine = true
		for _, member := range (*room).members {
			if member.status == MEMBER_CONNECTED {
				allOffLine = false
				break
			}
		}
		if allOffLine {
			log.Debug("AllMemberOffline %v", (*room).roomid)
			return ROOM_END
		}
	}

	return (*room).status
}

func changeRoomStatus(room *Room, status int32) {
	(*room).status = status
	log.Debug("changeRoomStatus Room %v Status %v", (*room).roomid, (*room).status)
}

func changeMemberStatus(member *Member, status int32) {
	(*member).status = status
	log.Debug("changeMemberStatus Member %v Status %v", (*member).charid, (*member).status)
}

func UpdateRoomManager(now *time.Time) {
	for i, room := range RoomManager {
		(*room).status = (*room).update(now)

		if (*room).status == ROOM_END {
			DeleteRoom(i)
		}
	}
}

func DeleteRoom(roomid int32) {
	log.Debug("DeleteRoom RoomID %v", roomid)
	delete(RoomManager, roomid)
}

func deleteRoomMemberInfo(roomid int32) {
	room, ok := RoomManager[roomid]
	if ok {
		for charid, _ := range room.members {
			delete(PlayerRoomIDMap, charid)
		}
	}
}

func CreateRoom(matchmode int32, membercnt int32) int32 {
	allocRoomID()

	battlekey, _ := tool.DesEncrypt([]byte(fmt.Sprintf(CRYPTO_PREFIX, roomid)), []byte(tool.CRYPT_KEY))

	room := &Room{
		roomid:         roomid,
		createtime:     time.Now().Unix(),
		nextchecktime:  time.Now().Unix() + 5,
		status:         ROOM_STATUS_NONE,
		matchmode:      matchmode,
		battlekey:      battlekey,
		members:        make(map[string]*Member),
		messages:       append([]interface{}{}),
		messagesbackup: append([]interface{}{}),
		membercnt:      membercnt,
		memberok:       0,
	}

	RoomManager[roomid] = room

	log.Debug("Create RoomID %v", roomid)
	return roomid
}

func JoinRoom(charid string, roomid int32, charname string, chartype int32, gameserverid int32) []byte {
	room, ok := RoomManager[roomid]
	if ok {
		member, ok := room.members[charid]
		if !ok {
			member = &Member{
				charid:       charid,
				teamtype:     0,
				charname:     charname,
				chartype:     chartype,
				status:       MEMBER_UNCONNECTED,
				gameserverid: gameserverid,
			}
			room.members[charid] = member

			log.Debug("JoinRoom RoomID %v CharID %v", roomid, charid)

			changeRoomStatus(room, ROOM_SYNC_PLAYERINFO)
			return room.battlekey
		} else {
			log.Error("JoinRoom RoomID %v CharID %v Already Exist", roomid, charid)
		}
	} else {
		log.Error("JoinRoom RoomID %v Not Exist CharID %v", roomid, charid)
	}

	return nil
}

func ConnectRoom(charid string, roomid int32, battlekey []byte) bool {
	room, ok := RoomManager[roomid]
	if ok {
		plaintext, err := tool.DesDecrypt(battlekey, []byte(tool.CRYPT_KEY))
		if err != nil {
			log.Error("ConnectRoom Battlekey Decrypt Err %v", err)
			return false
		}

		if strings.Compare(string(plaintext), fmt.Sprintf(CRYPTO_PREFIX, roomid)) != 0 {
			log.Error("ConnectRoom Battlekey Mismatch")
			return false
		}

		member, ok := room.members[charid]
		if ok {
			changeMemberStatus(member, MEMBER_CONNECTED)
			PlayerRoomIDMap[charid] = roomid
			room.memberok += 1

			if room.memberok == room.membercnt {
				changeRoomStatus(room, ROOM_FIGHTING)
			}

			log.Debug("ConnectRoom RoomID %v CharID %v", roomid, charid)
			return true
		} else {
			log.Error("ConnectRoom RoomID %v Member Not Exist %v", roomid, charid)
		}
	} else {
		log.Error("ConnectRoom RoomID %v Not Exist", roomid)
	}
	return false
}

func LeaveRoom(charid string) {
	setRoomMemberStatus(charid, MEMBER_OFFLINE)
}

func EndBattle(charid string) {
	setRoomMemberStatus(charid, MEMBER_END)
}

func setRoomMemberStatus(charid string, status int32) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			member, ok := room.members[charid]
			if ok {
				changeMemberStatus(member, status)

				log.Debug("SetRoomMemberStatus RoomID %v CharID %v Status %v", roomid, charid, status)
			}
		}
		delete(PlayerRoomIDMap, charid)
	}
}

func AddMessage(charid string, msgdata interface{}) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			(*room).messages = append((*room).messages, msgdata)
		} else {
			log.Error("AddMessage RoomID %v Not Exist", roomid)
		}
	} else {
		log.Error("AddMessage CharID %v Not Exist", charid)
	}
}

func FormatRoomInfo(roomid int32) string {
	room, ok := RoomManager[roomid]
	if ok {
		return fmt.Sprintf("RoomID:%v\tCreateTime:%v\tStatus:%v\tMemberCnt:%v", (*room).roomid, (*room).createtime, (*room).status, len((*room).members))
	}
	return ""
}

func FormatMemberInfo(roomid int32) string {
	output := fmt.Sprintf("RoomID:%v", roomid)
	room, ok := RoomManager[roomid]
	if ok {
		for _, member := range (*room).members {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tCharType:%v\tTeamType:%v\tStatus:%v", (*member).charid, (*member).charname, (*member).chartype, (*member).teamtype, (*member).status)}, "\r\n")
		}
	}
	return output
}
