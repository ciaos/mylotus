package internal

import (
	"errors"
	"fmt"
	"math/rand"
	"server/conf"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"server/tool"
	"strings"
	"time"

	"github.com/ciaos/leaf/log"
)

const (
	ROOM_STATUS_NONE = "room_status_none"
	ROOM_CONNECTING  = "room_connecting"
	ROOM_FIGHTING    = "room_fighting"
	ROOM_END         = "room_end"
	ROOM_CLEAR       = "room_clear"

	MEMBER_UNCONNECTED  = "member_unconnected"
	MEMBER_CONNECTED    = "member_connected"
	MEMBER_RECONNECTING = "member_reconnecting"
	MEMBER_RECONNECTED  = "member_reconnected"
	MEMBER_OFFLINE      = "member_offline"
	MEMBER_END          = "member_end"

	SYNC_BSINFO_STEP = 10
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
	skinid       int32
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
	lastSyncInfoTime = time.Now().Add(time.Second * time.Duration(-SYNC_BSINFO_STEP))
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
		if (member.status == MEMBER_CONNECTED || member.status == MEMBER_RECONNECTED || member.status == MEMBER_RECONNECTING) && member.ownerid == 0 {
			allOffLine = false
			break
		}
	}
	if allOffLine {
		log.Debug("AllMemberOffline %v", (*room).roomid)
		room.changeRoomStatus(ROOM_END)
	}
}

func getRoomByCharID(charid uint32, nolog bool) *Room {
	roomid, ok := PlayerRoomIDMap[charid]
	if ok {
		room, ok := RoomManager[roomid]
		if ok {
			return room
		} else {
			delete(PlayerRoomIDMap, charid)
		}
	}
	if nolog == false {
		log.Error("getRoomByCharID nil Charid %v", charid)
	}
	return nil
}

func getRoomByRoomID(roomid int32) *Room {
	room, ok := RoomManager[roomid]
	if ok {
		return room
	}
	return nil
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
							log.Debug("Reconnect All Frame Sent CharID %v, FrameID %v , Room.FrameID %v", member.charid, member.frameid, room.frameid)
							member.changeMemberStatus(MEMBER_CONNECTED)
							break
						}
					}
				} else {
					log.Error("CharID %v Invalid FrameID %v Room.FrameID %v", member.charid, member.frameid, room.frameid)
					member.frameid = 0 //restart
				}
			}
		}

		//bug
		if now.Unix()-(*room).createtime.Unix() > int64(3600) {
			log.Error("room Fight TimeOut Now %v CreateTime %v", now.Unix(), room.createtime)
			room.changeRoomStatus(ROOM_END)
			return
		}

		if (*now).Unix()-(*room).checktime.Unix() > 5 {
			(*room).checktime = (*now)
			room.checkOffline()
		}
	} else if (*room).status == ROOM_CONNECTING {
		//loading状态30秒直接切换
		if (*now).Unix()-(*room).createtime.Unix() > 30 {
			room.changeRoomStatus(ROOM_FIGHTING)
			return
		}
	} else if (*room).status == ROOM_CLEAR {
		if (*now).Unix()-(*room).checktime.Unix() > 5 {
			room.deleteRoomMember()
			room.deleteRoom()
		}
	} else if (*room).status == ROOM_STATUS_NONE {
		if (*now).Unix()-(*room).checktime.Unix() > 5 {
			log.Error("ROOM_STATUS_NONE TimeOut %v", (*room).roomid)
			room.changeRoomStatus(ROOM_END)
		}
	}
}

func (room *Room) changeRoomStatus(status string) {
	(*room).status = status
	room.checktime = time.Now()
	log.Debug("changeRoomStatus Room %v Status %v", (*room).roomid, (*room).status)

	if (*room).status == ROOM_END {
		//notify finish
		for _, member := range room.members {
			if member.ownerid == 0 && member.status != MEMBER_END {
				SendMessageTo(member.gameserverid, conf.Server.GameServerRename, member.charid, proxymsg.ProxyMessageType_PMT_BS_GS_FINISH_BATTLE, &proxymsg.Proxy_BS_GS_FINISH_BATTLE{CharID: member.charid})
			}
		}
		room.messagesbackup = append([]*clientmsg.Transfer_Command{})
		room.changeRoomStatus(ROOM_CLEAR)
	} else if (*room).status == ROOM_FIGHTING {
		room.notifyBattleStart()
	}
}

func (member *Member) changeMemberStatus(status string) {
	(*member).status = status
	log.Debug("changeMemberStatus Member %v Status %v", (*member).charid, (*member).status)
}

func syncBSInfoToMS() {
	msg := &proxymsg.Proxy_BS_MS_SyncBSInfo{
		BattleServerID:    int32(conf.Server.ServerID),
		BattleRoomCount:   int32(len(RoomManager)),
		BattleMemberCount: int32(len(PlayerRoomIDMap)),
	}
	BroadCastMessageTo("matchserver", 0, proxymsg.ProxyMessageType_PMT_BS_MS_SYNCBSINFO, msg)
}

func UpdateRoomManager(now *time.Time) {
	for _, room := range RoomManager {
		(*room).update(now)
	}

	if now.Unix()-lastSyncInfoTime.Unix() >= SYNC_BSINFO_STEP {
		syncBSInfoToMS()
		lastSyncInfoTime = *now
	}
}

func (room *Room) deleteRoom() {
	log.Debug("DeleteRoom RoomID %v", room.roomid)
	delete(RoomManager, room.roomid)
}

func (room *Room) deleteRoomMember() {
	for charid, member := range room.members {
		if member.ownerid == 0 {
			rid, exist := PlayerRoomIDMap[charid]
			if exist && rid == room.roomid { //玩家已进入另外一场战斗，不删除映射信息和网络连接
				delete(PlayerRoomIDMap, charid)
				RemoveBattlePlayer(member.charid, "", REASON_CLEAR)
			}
		}
		delete(room.members, charid)
	}
}

func createRoom(msg *proxymsg.Proxy_MS_BS_AllocBattleRoom) (error, int32, []byte) {
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
			skinid:       mem.SkinID,
		}

		//Leave Previous Room
		prevroom := getRoomByCharID(mem.CharID, true)
		if prevroom != nil {
			log.Debug("CharID %v Leave Previous RoomID %v", mem.CharID, prevroom.roomid)
			prevroom.LeaveRoom(mem.CharID)
			RemoveBattlePlayer(mem.CharID, "", REASON_REPLACED)
		}

		member.changeMemberStatus(MEMBER_UNCONNECTED)
		room.members[member.charid] = member
		log.Debug("JoinRoom RoomID %v CharID %v OwnerID %v", room.roomid, member.charid, member.ownerid)
	}

	room.changeRoomStatus(ROOM_STATUS_NONE)
	RoomManager[room.roomid] = room

	log.Release("Create RoomID %v For TableID %v MatchMode %v MapID %v", room.roomid, msg.Matchtableid, msg.Matchmode, msg.Mapid)
	return nil, room.roomid, room.battlekey
}

func (room *Room) notifyBattleStart() {
	rsp := &clientmsg.Rlt_StartBattle{
		RandSeed: room.seed,
	}
	log.Debug("notifyBattleStart %v", room.roomid)
	room.broadcast(rsp)
}

func (room *Room) loadingRoom(charid uint32, req *clientmsg.Transfer_Loading_Progress) {
	log.Debug("SetLoadingProgress RoomID %v CharID %v PlayerID %v Progress %v RoomStatus %v", room.roomid, charid, (*req).CharID, (*req).Progress, room.status)
	if room.status == ROOM_CONNECTING {
		member, ok := room.members[(*req).CharID]
		if ok {
			member.progress = (*req).Progress

			room.broadcast(req)

			if member.progress >= 100 {
				room.memberloadingok += 1

				if room.memberloadingok >= len(room.members) {
					room.changeRoomStatus(ROOM_FIGHTING)
				}
			}
		}
	} else {
		member, ok := room.members[(*req).CharID]
		if ok && member.status == MEMBER_RECONNECTING {
			room.sendmsg(member.charid, req)
			member.progress = (*req).Progress
			if member.progress >= 100 {
				rsp := &clientmsg.Rlt_StartBattle{
					RandSeed: room.seed,
				}
				room.sendmsg((*req).CharID, rsp)
				member.changeMemberStatus(MEMBER_RECONNECTED)
			}
		}
	}
}

func (room *Room) genRoomInfoPB(charid uint32, isreconnect bool) *clientmsg.Rlt_ConnectBS {
	rsp := &clientmsg.Rlt_ConnectBS{
		IsReconnect: isreconnect,
		RetCode:     clientmsg.Type_BattleRetCode_BRC_OK,
		MapID:       room.mapid,
	}

	for _, member := range room.members {
		m := &clientmsg.MemberInfo{
			CharID:   member.charid,
			CharName: member.charname,
			CharType: member.chartype,
			TeamID:   member.teamid,
			OwnerID:  member.ownerid,
			SkinID:   member.skinid,
			Progress: uint32(member.progress),
		}

		rsp.Member = append(rsp.Member, m)
	}
	return rsp
}

func (room *Room) connectRoom(charid uint32, battlekey []byte, remoteaddr string) (bool, string) {
	plaintext, err := tool.DesDecrypt(battlekey, []byte(tool.CRYPT_KEY))
	if err != nil {
		log.Error("ConnectRoom Battlekey Decrypt Err %v", err)
		return false, ""
	}

	if strings.Compare(string(plaintext), fmt.Sprintf(CRYPTO_PREFIX, room.roomid)) != 0 {
		log.Error("ConnectRoom Battlekey Mismatch")
		return false, ""
	}

	member, ok := room.members[charid]
	if ok {
		member.remoteaddr = remoteaddr
		member.changeMemberStatus(MEMBER_CONNECTED)
		PlayerRoomIDMap[charid] = room.roomid

		//set connect for robot
		for _, mem := range room.members {
			if mem.ownerid == charid {
				mem.changeMemberStatus(MEMBER_CONNECTED)
			}
		}

		if room.status == ROOM_STATUS_NONE {
			room.changeRoomStatus(ROOM_CONNECTING)
		} else if room.status == ROOM_FIGHTING {
			member.changeMemberStatus(MEMBER_RECONNECTED)
		}

		log.Release("ConnectRoom RoomID %v CharID %v", room.roomid, charid)
		return true, member.charname
	} else {
		log.Error("ConnectRoom RoomID %v Member Not Exist %v", room.roomid, charid)
	}
	return false, ""
}

func (room *Room) reConnectRoom(charid uint32, frameid uint32, battlekey []byte, remoteaddr string) (bool, string) {
	plaintext, err := tool.DesDecrypt(battlekey, []byte(tool.CRYPT_KEY))
	if err != nil {
		log.Error("ReConnectRoom Battlekey Decrypt Err %v", err)
		return false, ""
	}

	if strings.Compare(string(plaintext), fmt.Sprintf(CRYPTO_PREFIX, room.roomid)) != 0 {
		log.Error("ReConnectRoom Battlekey Mismatch")
		return false, ""
	}

	member, ok := room.members[charid]
	if ok {
		member.remoteaddr = remoteaddr
		member.changeMemberStatus(MEMBER_RECONNECTING)
		member.frameid = frameid
		log.Debug("ReConnectRoom RoomID %v CharID %v", room.roomid, charid)
		return true, member.charname
	} else {
		log.Error("ReConnectRoom RoomID %v Member Not Exist %v", room.roomid, charid)
	}
	return false, ""
}

func (room *Room) getMemberGSID(charid uint32) int32 {
	member, ok := room.members[charid]
	if ok {
		return member.gameserverid
	}
	return 0
}

func (room *Room) getMemberRemoteAddr(charid uint32) string {
	member, ok := room.members[charid]
	if ok {
		return member.remoteaddr
	}
	return ""
}

func (room *Room) LeaveRoom(charid uint32) {
	member, ok := room.members[charid]
	if ok {
		member.changeMemberStatus(MEMBER_OFFLINE)
		room.checkOffline()
	}
}

func (room *Room) EndBattle(charid uint32) {
	member, ok := room.members[charid]
	if ok {
		member.changeMemberStatus(MEMBER_END)

		if member.ownerid == 0 {
			SendMessageTo(member.gameserverid, conf.Server.GameServerRename, charid, proxymsg.ProxyMessageType_PMT_BS_GS_FINISH_BATTLE, &proxymsg.Proxy_BS_GS_FINISH_BATTLE{CharID: member.charid})
		}
	}
}

func (room *Room) AddFrameMessage(charid uint32, transcmd *clientmsg.Transfer_Command) {
	for _, message := range transcmd.Messages {
		message.CharID = charid
		room.messages = append(room.messages, message)
	}
}

func (room *Room) FormatRoomInfo() string {
	return fmt.Sprintf("RoomID:%v\tCreateTime:%v\tStatus:%v\tMemberCnt:%v\tMatchMode:%v\tMapID:%v\tFrameID:%v", (*room).roomid, (*room).createtime.Format(TIME_FORMAT), (*room).status, len((*room).members), room.matchmode, room.mapid, room.frameid)
}

func (room *Room) FormatMemberInfo() string {
	output := room.FormatRoomInfo()
	for _, member := range (*room).members {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%10v\tCharType:%v\tSkinID:%v\tTeamType:%v\tStatus:%v\tGSID:%v\tOwnerID:%v\tFrameID:%v\tCharName:%v", member.charid, member.chartype, member.skinid, member.teamid, member.status, member.gameserverid, member.ownerid, member.frameid, member.charname)}, "\r\n")
	}
	return output
}
