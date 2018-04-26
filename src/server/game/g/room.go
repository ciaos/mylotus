package g

import (
	"errors"
	"fmt"
	"math/rand"
	"server/conf"
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
	ROOM_CLEAR       = "room_clear"

	MEMBER_UNCONNECTED = "member_unconnected"
	MEMBER_CONNECTED   = "member_connected"
	MEMBER_RECONNECTED = "member_reconnected"
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
	frameid      uint32
	remoteaddr   string
}

type Room struct {
	createtime time.Time
	checktime  time.Time
	status     string
	roomid     int32
	matchmode  int32
	mapid      int32
	battlekey  []byte
	frameid    uint32
	seed       int32

	memberloadingok int

	members map[uint32]*Member

	messages       []*clientmsg.Transfer_Command_CommandData
	messagesbackup []*clientmsg.Transfer_Command
}

var PlayerRoomIDMap = make(map[uint32]int32, 1024)
var RoomManager = make(map[int32]*Room, 128)
var g_roomid int32

var lastSyncInfoTime time.Time

//var mRoomID *sync.Mutex

func InitRoomManager() {
	g_roomid = 0
	lastSyncInfoTime = time.Now()
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

		player, ok := BattlePlayerManager[(*member).charid]
		if ok {
			(*player.agent).WriteMsg(msgdata)
		} else {
			member.status = MEMBER_OFFLINE
		}
	}
}

func (room *Room) sendmsg(charid uint32, msgdata interface{}) {
	roomid, ok := PlayerRoomIDMap[charid]
	if !ok || roomid != room.roomid {
		log.Error("room Sendmsg Error %v RoomID %v CharID %v Room.RoomID %v", ok, roomid, charid, room.roomid)
		return
	}

	player, ok := BattlePlayerManager[charid]
	if ok {
		(*player.agent).WriteMsg(msgdata)
	}
}

func (room *Room) checkOffline() {
	var allOffLine = true
	for _, member := range room.members {
		if (member.status == MEMBER_CONNECTED || member.status == MEMBER_RECONNECTED) && member.ownerid == 0 {
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
		//log.Debug("frameid %v", room.frameid)

		//backup
		(*room).messagesbackup = append((*room).messagesbackup, rsp)
		//clear
		(*room).messages = append([]*clientmsg.Transfer_Command_CommandData{})

		for _, member := range room.members {
			if member.status == MEMBER_RECONNECTED {
				if member.frameid < room.frameid {
					maxSendCnt := 100
					for maxSendCnt > 0 {
						maxSendCnt--

						msgdata := room.messagesbackup[member.frameid]
						room.sendmsg(member.charid, msgdata)

						member.frameid = msgdata.FrameID

						if member.frameid >= room.frameid || member.frameid >= uint32(len(room.messagesbackup)) {
							log.Debug("Reconnect All Frame Sent 1 CharID %v, FrameID %v", member.charid, member.frameid)
							changeMemberStatus(member, MEMBER_CONNECTED)
							break
						}
					}
				} else {
					log.Debug("Reconnect All Frame Sent 2 CharID %v, FrameID %v", member.charid, member.frameid)
					changeMemberStatus(member, MEMBER_CONNECTED)
				}
			}
		}

		//bug
		if now.Unix()-(*room).createtime.Unix() > int64(3600) {
			log.Error("room Fight TimeOut Now %v CreateTime %v", now.Unix(), room.createtime)
			changeRoomStatus(room, ROOM_END)
			return
		}

		if (*now).Unix()-(*room).checktime.Unix() > 5 {
			(*room).checktime = (*now)
			room.checkOffline()
		}
	} else if (*room).status == ROOM_CONNECTING {
		//loading状态30秒直接切换
		if (*now).Unix()-(*room).createtime.Unix() > 30 {
			changeRoomStatus(room, ROOM_FIGHTING)
			return
		}
	} else if (*room).status == ROOM_CLEAR {
		if (*now).Unix()-(*room).checktime.Unix() > 5 {
			deleteRoomMemberInfo((*room).roomid)
			DeleteRoom((*room).roomid)
		}
	} else if (*room).status == ROOM_STATUS_NONE {
		if (*now).Unix()-(*room).checktime.Unix() > 5 {
			log.Error("ROOM_STATUS_NONE TimeOut %v", (*room).roomid)
			changeRoomStatus(room, ROOM_END)
		}
	}
}

func changeRoomStatus(room *Room, status string) {
	(*room).status = status
	room.checktime = time.Now()
	log.Debug("changeRoomStatus Room %v Status %v", (*room).roomid, (*room).status)

	if (*room).status == ROOM_END {
		//notify finish
		for _, member := range room.members {
			if member.ownerid == 0 && member.status != MEMBER_END {
				go SendMessageTo(member.gameserverid, conf.Server.GameServerRename, member.charid, proxymsg.ProxyMessageType_PMT_BS_GS_FINISH_BATTLE, &proxymsg.Proxy_BS_GS_FINISH_BATTLE{CharID: member.charid})
			}
		}
		room.messagesbackup = append([]*clientmsg.Transfer_Command{})
		changeRoomStatus(room, ROOM_CLEAR)
	} else if (*room).status == ROOM_FIGHTING {
		room.notifyBattleStart()
	}
}

func changeMemberStatus(member *Member, status string) {
	(*member).status = status
	log.Debug("changeMemberStatus Member %v Status %v", (*member).charid, (*member).status)
}

func syncBSInfoToMS() {
	msg := &proxymsg.Proxy_BS_MS_SyncBSInfo{
		BattleServerID : int32(conf.Server.ServerID),
		BattleRoomCount : int32(len(RoomManager)),
		BattleMemberCount : int32(len(PlayerRoomIDMap)),
	}
	go BroadCastMessageTo("matchserver", 0, proxymsg.ProxyMessageType_PMT_BS_MS_SYNCBSINFO, msg)
}

func UpdateRoomManager(now *time.Time) {
	for _, room := range RoomManager {
		(*room).update(now)
	}

	if now.Unix() - lastSyncInfoTime.Unix() >= 10 {
		syncBSInfoToMS()
		lastSyncInfoTime = *now
	}
}

func DeleteRoom(roomid int32) {
	log.Debug("DeleteRoom RoomID %v", roomid)
	delete(RoomManager, roomid)
}

func deleteRoomMemberInfo(roomid int32) {
	room, ok := RoomManager[roomid]
	if ok {
		for charid, member := range room.members {
			if member.ownerid == 0 {
				rid, exist := PlayerRoomIDMap[charid]
				if exist && rid == roomid { //玩家已进入另外一场战斗，不删除映射信息和网络连接
					delete(PlayerRoomIDMap, charid)
					RemoveBattlePlayer(member.charid, "", REASON_CLEAR)
				}
			}
			delete(room.members, charid)
		}
	}
}

func CreateRoom(msg *proxymsg.Proxy_MS_BS_AllocBattleRoom) (error, int32, []byte) {
	allocRoomID()

	//roomid has being used
	_, ok := RoomManager[g_roomid]
	if ok {
		log.Error("RoomID %v is Still Being Using, Current RoomCnt %v", g_roomid, len(RoomManager))
		return errors.New("roomid is using"), 0, nil
	}

	battlekey, _ := tool.DesEncrypt([]byte(fmt.Sprintf(CRYPTO_PREFIX, g_roomid)), []byte(tool.CRYPT_KEY))

	room := &Room{
		roomid:          g_roomid,
		createtime:      time.Now(),
		checktime:       time.Now(),
		status:          ROOM_STATUS_NONE,
		matchmode:       msg.Matchmode,
		mapid:           msg.Mapid,
		battlekey:       battlekey,
		members:         make(map[uint32]*Member, 10),
		messages:        make([]*clientmsg.Transfer_Command_CommandData, 0, 10),
		messagesbackup:  make([]*clientmsg.Transfer_Command, 0, 1000),
		memberloadingok: 0,
		frameid:         0,
		seed:            int32(rand.Intn(100000)),
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
			frameid:      0,
			remoteaddr:   "",
		}

		//Leave Previous Room
		oldroomid, ok := PlayerRoomIDMap[mem.CharID]
		if ok {
			log.Debug("CharID %v Leave Previous RoomID %v", mem.CharID, oldroomid)
			LeaveRoom(mem.CharID)
			RemoveBattlePlayer(mem.CharID, "", REASON_REPLACED)
		}

		changeMemberStatus(member, MEMBER_UNCONNECTED)
		room.members[member.charid] = member
		log.Debug("JoinRoom RoomID %v CharID %v OwnerID %v", room.roomid, member.charid, member.ownerid)
	}

	changeRoomStatus(room, ROOM_STATUS_NONE)
	RoomManager[room.roomid] = room

	log.Debug("Create RoomID %v", room.roomid)
	return nil, room.roomid, room.battlekey
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
				log.Error("Invalid Status %v RoomID %v Charid %v Progress %v", room.status, room.roomid, req.CharID, req.Progress)
				return
			}

			member, ok := room.members[(*req).CharID]
			if ok {
				member.progress = (*req).Progress
				log.Debug("SetLoadingProgress RoomID %v CharID %v PlayerID %v Progress %v", roomid, charid, (*req).CharID, (*req).Progress)
				room.broadcast(req)

				if member.progress >= 100 {
					room.memberloadingok += 1

					if room.memberloadingok >= len(room.members) {
						changeRoomStatus(room, ROOM_FIGHTING)
					}
				}
			}
		} else {
			delete(PlayerRoomIDMap, charid)
		}
	}
}

func QueryBattleInfo(charid uint32) (bool, []byte, string) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			_, ok := room.members[charid]
			if ok {
				return true, room.battlekey, conf.Server.ConnectAddr
			}
		}
	}
	return false, nil, ""
}

func GenRoomInfoPB(charid uint32, isreconnect bool) *clientmsg.Rlt_ConnectBS {
	rsp := &clientmsg.Rlt_ConnectBS{
		IsReconnect: isreconnect,
	}

	roomid, _ := PlayerRoomIDMap[charid]
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

func ConnectRoom(charid uint32, roomid int32, battlekey []byte, remoteaddr string) (bool, string) {
	room, ok := RoomManager[roomid]
	if ok {
		plaintext, err := tool.DesDecrypt(battlekey, []byte(tool.CRYPT_KEY))
		if err != nil {
			log.Error("ConnectRoom Battlekey Decrypt Err %v", err)
			return false, ""
		}

		if strings.Compare(string(plaintext), fmt.Sprintf(CRYPTO_PREFIX, roomid)) != 0 {
			log.Error("ConnectRoom Battlekey Mismatch")
			return false, ""
		}

		member, ok := room.members[charid]
		if ok {
			member.remoteaddr = remoteaddr
			changeMemberStatus(member, MEMBER_CONNECTED)
			PlayerRoomIDMap[charid] = roomid

			//set connect for robot
			for _, mem := range room.members {
				if mem.ownerid == charid {
					changeMemberStatus(mem, MEMBER_CONNECTED)
				}
			}

			if room.status == ROOM_STATUS_NONE {
				changeRoomStatus(room, ROOM_CONNECTING)
			} else if room.status == ROOM_FIGHTING {
				changeMemberStatus(member, MEMBER_RECONNECTED)
			}

			log.Debug("ConnectRoom RoomID %v CharID %v", roomid, charid)
			return true, member.charname
		} else {
			log.Error("ConnectRoom RoomID %v Member Not Exist %v", roomid, charid)
		}
	} else {
		log.Error("ConnectRoom RoomID %v Not Exist", roomid)
	}
	return false, ""
}

func ReConnectRoom(charid uint32, frameid uint32, battlekey []byte, remoteaddr string) (bool, string) {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			plaintext, err := tool.DesDecrypt(battlekey, []byte(tool.CRYPT_KEY))
			if err != nil {
				log.Error("ReConnectRoom Battlekey Decrypt Err %v", err)
				return false, ""
			}

			if strings.Compare(string(plaintext), fmt.Sprintf(CRYPTO_PREFIX, roomid)) != 0 {
				log.Error("ReConnectRoom Battlekey Mismatch")
				return false, ""
			}

			member, ok := room.members[charid]
			if ok {
				member.remoteaddr = remoteaddr
				changeMemberStatus(member, MEMBER_RECONNECTED)
				member.frameid = frameid
				log.Debug("ReConnectRoom RoomID %v CharID %v", roomid, charid)
				return true, member.charname
			} else {
				log.Error("ReConnectRoom RoomID %v Member Not Exist %v", roomid, charid)
			}
		} else {
			log.Error("ReConnectRoom RoomID %v Not Exist", roomid)
		}
	}
	return false, ""
}

func GetMemberGSID(charid uint32) int32 {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			member, ok := room.members[charid]
			if ok {
				return member.gameserverid
			}
		}
	}
	return 0
}

func GetMemberRemoteAddr(charid uint32) string {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			member, ok := room.members[charid]
			if ok {
				return member.remoteaddr
			}
		}
	}
	return ""
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

				if member.ownerid == 0 {
					go SendMessageTo(member.gameserverid, conf.Server.GameServerRename, charid, proxymsg.ProxyMessageType_PMT_BS_GS_FINISH_BATTLE, &proxymsg.Proxy_BS_GS_FINISH_BATTLE{CharID: member.charid})
				}
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
		return fmt.Sprintf("RoomID:%v\tCreateTime:%v\tStatus:%v\tMemberCnt:%v\tMatchMode:%v\tMapID:%v\tFrameID:%v", (*room).roomid, (*room).createtime.Format(TIME_FORMAT), (*room).status, len((*room).members), room.matchmode, room.mapid, room.frameid)
	}
	return ""
}

func FormatMemberInfo(roomid int32) string {
	output := FormatRoomInfo(roomid)
	room, ok := RoomManager[roomid]
	if ok {
		for _, member := range (*room).members {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%10v\tCharType:%v\tTeamType:%v\tStatus:%v\tGSID:%v\tOwnerID:%v\tFrameID:%v\tCharName:%v", member.charid, member.chartype, member.teamid, member.status, member.gameserverid, member.ownerid, member.frameid, member.charname)}, "\r\n")
		}
	}
	return output
}
