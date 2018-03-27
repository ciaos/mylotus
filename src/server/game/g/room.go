package g

import (
	"fmt"
	"math/rand"
	"server/tool"
	"strings"
	//	"sync"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"time"

	"github.com/ciaos/leaf/log"
)

const (
	ROOM_STATUS_NONE = "room_status_none"
	ROOM_CONNECTING  = "room_connecting"
	ROOM_FIGHTING    = "room_fighting"
	ROOM_END         = "room_end"

	MEMBER_UNCONNECTED = "member_unconnected"
	MEMBER_CONNECTED   = "member_connected"
	MEMBER_OFFLINE     = "member_offline"
	MEMBER_END         = "member_end"
)

type Member struct {
	charid       uint32
	charname     string
	chartype     int32
	teamid       int32
	status       string
	gameserverid int32
	ownerid      uint32 // robot 有控制者
	progress     int32
}

type Room struct {
	createtime    int64
	nextchecktime int64
	status        string
	roomid        int32
	matchmode     int32
	mapid         int32
	battlekey     []byte
	frameid       uint32
	seed          int32

	memberok int

	members map[uint32]*Member

	messages       []*clientmsg.Transfer_Command_CommandData
	messagesbackup []*clientmsg.Transfer_Command
}

var PlayerRoomIDMap = make(map[uint32]int32)
var RoomManager = make(map[int32]*Room)
var g_roomid int32

//var mRoomID *sync.Mutex

func InitRoomManager() {
	g_roomid = 0

	//mRoomID = new(sync.Mutex)
}

func UninitRoomManager() {
	for roomid, room := range RoomManager {
		for memberid, _ := range room.members {
			delete(room.members, memberid)
		}
		delete(RoomManager, roomid)
	}
	for charid := range PlayerRoomIDMap {
		delete(PlayerRoomIDMap, charid)
	}
}

func allocRoomID() {
	//mRoomID.Lock()
	//defer mRoomID.Unlock()
	g_roomid += 1
	if g_roomid > MAX_ROOM_COUNT {
		g_roomid = 1
	}
}

func (room *Room) broadcast(msgdata interface{}) {
	for _, member := range (*room).members {
		if member.ownerid != 0 {
			continue
		}

		if member.status != MEMBER_CONNECTED {
			continue
		}

		agent, ok := BattlePlayerManager[(*member).charid]
		if ok {
			(*agent).WriteMsg(msgdata)
		}
	}
}

func (room *Room) checkOffline() {
	var allOffLine = true
	for _, member := range room.members {
		if member.status == MEMBER_CONNECTED && member.ownerid == 0 {
			allOffLine = false
			break
		}
	}
	if allOffLine {
		log.Debug("AllMemberOffline %v", (*room).roomid)
		changeRoomStatus(room, ROOM_END)
	}
}

func (room *Room) update(now *time.Time) {
	if (*room).status == ROOM_FIGHTING {
		(*room).frameid++

		rsp := &clientmsg.Transfer_Command{
			FrameID: room.frameid,
		}
		for _, message := range (*room).messages {
			rsp.Messages = append(rsp.Messages, message)
		}
		(*room).broadcast(rsp)

		//backup
		(*room).messagesbackup = append((*room).messagesbackup, rsp)
		//clear
		(*room).messages = append([]*clientmsg.Transfer_Command_CommandData{})

		//bug
		if now.Unix()-(*room).createtime > int64(600) {
			room.broadcast(&clientmsg.Rlt_EndBattle{
				RetCode: clientmsg.Type_BattleRetCode_BRC_OK,
			})
			changeRoomStatus(room, ROOM_END)
			return
		}
	} else if (*room).status == ROOM_CONNECTING {
		if (*now).Unix()-(*room).createtime > 30 {
			changeRoomStatus(room, ROOM_FIGHTING)
			return
		}
	}

	if (*room).nextchecktime < (*now).Unix() {
		(*room).nextchecktime = (*now).Unix() + 5

		if (*room).status == ROOM_STATUS_NONE {
			log.Error("ROOM_STATUS_NONE TimeOut %v", (*room).roomid)
			changeRoomStatus(room, ROOM_END)
			return
		}

		room.checkOffline()
	}
}

func changeRoomStatus(room *Room, status string) {
	(*room).status = status
	log.Debug("changeRoomStatus Room %v Status %v", (*room).roomid, (*room).status)

	if (*room).status == ROOM_END {
		room.messagesbackup = append([]*clientmsg.Transfer_Command{})
		deleteRoomMemberInfo((*room).roomid)
		DeleteRoom((*room).roomid)
	} else if (*room).status == ROOM_FIGHTING {
		room.notifyBattleStart()
	}
}

func changeMemberStatus(member *Member, status string) {
	(*member).status = status
	log.Debug("changeMemberStatus Member %v Status %v", (*member).charid, (*member).status)
}

func UpdateRoomManager(now *time.Time) {
	for _, room := range RoomManager {
		(*room).update(now)
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

func CreateRoom(msg *proxymsg.Proxy_MS_BS_AllocBattleRoom) (int32, []byte) {
	allocRoomID()

	battlekey, _ := tool.DesEncrypt([]byte(fmt.Sprintf(CRYPTO_PREFIX, g_roomid)), []byte(tool.CRYPT_KEY))

	room := &Room{
		roomid:         g_roomid,
		createtime:     time.Now().Unix(),
		nextchecktime:  time.Now().Unix() + 10,
		status:         ROOM_STATUS_NONE,
		matchmode:      msg.Matchmode,
		mapid:          msg.Mapid,
		battlekey:      battlekey,
		members:        make(map[uint32]*Member),
		messages:       append([]*clientmsg.Transfer_Command_CommandData{}),
		messagesbackup: append([]*clientmsg.Transfer_Command{}),
		memberok:       0,
		frameid:        0,
		seed:           int32(rand.Intn(100000)),
	}

	for _, mem := range (*msg).Members {
		member := &Member{
			charid:       mem.CharID,
			teamid:       mem.TeamID,
			charname:     mem.CharName,
			chartype:     mem.CharType,
			status:       MEMBER_UNCONNECTED,
			gameserverid: mem.GameServerID,
			ownerid:      mem.OwnerID,
			progress:     0,
		}

		//Leave Previous Room
		_, ok := PlayerRoomIDMap[mem.CharID]
		if ok {
			LeaveRoom(mem.CharID)
		}

		changeMemberStatus(member, MEMBER_UNCONNECTED)
		room.members[member.charid] = member
		log.Debug("JoinRoom RoomID %v CharID %v OwnerID %v", room.roomid, member.charid, member.ownerid)
	}

	changeRoomStatus(room, ROOM_STATUS_NONE)
	RoomManager[room.roomid] = room

	log.Debug("Create RoomID %v", room.roomid)
	return room.roomid, room.battlekey
}

func (room *Room) notifyBattleStart() {
	rsp := &clientmsg.Rlt_StartBattle{
		RandSeed: room.seed,
	}
	log.Debug("notifyBattleStart %v", room.roomid)
	room.broadcast(rsp)
}

func LoadingRoom(charid uint32, req *clientmsg.Transfer_Loading_Progress) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			if room.status != ROOM_CONNECTING {
				log.Error("Invalid Status %v RoomID %v", room.status, room.roomid)
				return
			}

			member, ok := room.members[(*req).CharID]
			if ok {
				member.progress = (*req).Progress
				log.Debug("SetLoadingProgress RoomID %v CharID %v PlayerID %v Progress %v", roomid, charid, (*req).CharID, (*req).Progress)
				room.broadcast(req)

				if member.progress >= 100 {
					room.memberok += 1

					if room.memberok >= len(room.members) {
						changeRoomStatus(room, ROOM_FIGHTING)
					}
				}
			}
		} else {
			delete(PlayerRoomIDMap, charid)
		}
	}
}

func GenRoomInfoPB(roomid int32) *clientmsg.Rlt_ConnectBS {
	rsp := &clientmsg.Rlt_ConnectBS{}
	room, ok := RoomManager[roomid]
	if ok {
		rsp.RetCode = clientmsg.Type_BattleRetCode_BRC_OK
		rsp.MapID = room.mapid

		for _, member := range room.members {
			m := &clientmsg.Rlt_ConnectBS_MemberInfo{
				CharID:   member.charid,
				CharName: member.charname,
				CharType: member.chartype,
				TeamID:   member.teamid,
				OwnerID:  member.ownerid,
			}

			rsp.Member = append(rsp.Member, m)
		}
	} else {
		log.Error("GenRoomInfoPB RoomID %v Error", roomid)
		rsp.RetCode = clientmsg.Type_BattleRetCode_BRC_OTHER
	}
	return rsp
}

func ConnectRoom(charid uint32, roomid int32, battlekey []byte) bool {
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

			if room.status == ROOM_STATUS_NONE {
				changeRoomStatus(room, ROOM_CONNECTING)
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

func LeaveRoom(charid uint32) {
	setRoomMemberStatus(charid, MEMBER_OFFLINE)
}

func EndBattle(charid uint32) {
	setRoomMemberStatus(charid, MEMBER_END)
}

func setRoomMemberStatus(charid uint32, status string) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			member, ok := room.members[charid]
			if ok {
				changeMemberStatus(member, status)

				log.Debug("SetRoomMemberStatus RoomID %v CharID %v Status %v", roomid, charid, status)
				room.checkOffline()
			}
		} else {
			delete(PlayerRoomIDMap, charid)
		}
	}
}

func AddMessage(charid uint32, transcmd *clientmsg.Transfer_Command) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			for _, message := range transcmd.Messages {
				message.CharID = charid
				(*room).messages = append((*room).messages, message)
			}

		} else {
			log.Error("AddMessage RoomID %v Not Exist", roomid)
			delete(PlayerRoomIDMap, charid)
		}
	} else {
		//正常情况
		//log.Debug("AddMessage CharID %v Not Exist Size %v", charid, len(PlayerRoomIDMap))
	}
}

func TransferRoomMessage(charid uint32, transcmd *clientmsg.Transfer_Battle_Message) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			room.broadcast(transcmd)
		} else {
			log.Error("TransferRoomMessage RoomID %v Not Exist", roomid)
			delete(PlayerRoomIDMap, charid)
		}
	} else {
		log.Error("TransferRoomMessage CharID %v Not Exist", charid)
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
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tCharType:%v\tTeamType:%v\tStatus:%v", (*member).charid, (*member).charname, (*member).chartype, (*member).teamid, (*member).status)}, "\r\n")
		}
	}
	return output
}
