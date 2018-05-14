package internal

import (
	"server/conf"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"github.com/golang/protobuf/proto"
)

func init() {
	skeleton.RegisterChanRPC("NewAgent", rpcNewAgent)
	skeleton.RegisterChanRPC("CloseAgent", rpcCloseAgent)
	skeleton.RegisterChanRPC("TickFrame", updateFrame)
	skeleton.RegisterChanRPC("QueueMessage", queueMessage)
}

var (
	lastTickTime int64
)

func queueMessage(args []interface{}) {
	pmsg := &proxymsg.InternalMessage{}
	err := proto.Unmarshal(args[0].([]byte), pmsg)
	if err != nil {
		log.Error("queueMessage InnerMsg Decode Error %v", err)
		return
	}
	if time.Now().Unix()-pmsg.Time > 1 {
		log.Error("server busy proxymsg createts %v nowts %v msgtype %v", pmsg.Time, time.Now().Unix(), proxymsg.ProxyMessageType(pmsg.Msgid))
	}

	switch proxymsg.ProxyMessageType(pmsg.Msgid) {
	case proxymsg.ProxyMessageType_PMT_GS_MS_MATCH:
		proxyHandleGSMSMatch(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_BS_ALLOCBATTLEROOM:
		proxyHandleMSBSAllocBattleRoom(pmsg)
	case proxymsg.ProxyMessageType_PMT_BS_MS_ALLOCBATTLEROOM:
		proxyHandleBSMSAllocBattleRoom(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT:
		proxyHandleMSGSMatchResult(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_BEGIN_BATTLE:
		proxyHandleMSGSBeginBattle(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_MS_CHOOSE_OPERATE:
		proxyHandleGSMSTeamOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_CHOOSE_OPERATE:
		proxyHandleMSGSTeamOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_MS_OFFLINE:
		proxyHandleGSMSOffline(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_GS_FRIEND_OPERATE:
		proxyHandleGSGSFriendOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_BS_QUERY_BATTLEINFO:
		proxyHandleGSBSQueryBattleInfo(pmsg)
	case proxymsg.ProxyMessageType_PMT_BS_GS_QUERY_BATTLEINFO:
		proxyHandleBSGSQueryBattleInfo(pmsg)
	case proxymsg.ProxyMessageType_PMT_BS_GS_FINISH_BATTLE:
		proxyHandleBSGSFinishBattle(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_MS_RECONNECT:
		proxyHandleGSMSReconnect(pmsg)
	case proxymsg.ProxyMessageType_PMT_BS_MS_SYNCBSINFO:
		proxyHandleBSMSSyncBSInfo(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_MS_MAKE_TEAM_OPERATE:
		proxyHandleGSMSMakeTeamOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE:
		proxyHandleMSGSMakeTeamOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_DELETE:
		proxyHandleMSGSDelete(pmsg)
	default:
		log.Error("Invalid InnerMsg ID %v", pmsg.Msgid)
	}
}

func proxyHandleGSGSFriendOperate(pmsg *proxymsg.InternalMessage) {
	m := &clientmsg.Req_Friend_Operate{}
	err := proto.Unmarshal(pmsg.Msgdata, m)
	if err != nil {
		log.Error("proxymsg.Req_Friend_Operate Decode Error %v", err)
		return
	}

	player, _ := GetPlayer(m.OperateCharID)
	if m.Action == clientmsg.FriendOperateActionType_FOAT_ADD_FRIEND {
		player.GetPlayerAsset().AssetFriend_AddApplyInfo(pmsg.Charid, m)
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_DEL_FRIEND {
		player.GetPlayerAsset().AssetFriend_DelFriend(m.OperateCharID, pmsg.Charid)
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_ACCEPT {
		player.GetPlayerAsset().AssetFriend_AcceptApplyInfo(m.OperateCharID, pmsg.Charid)
	} else {
		log.Error("Invalid Friend Operate Type %v", m.Action)
	}

}

func proxyHandleGSMSMatch(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_MS_Match{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_GS_MS_Match Decode Error %v", err)
		return
	}
	log.Debug("proxyHandleGSMSMatch CharID %v Action %v", msg.Charid, msg.Action)

	if msg.Action == int32(clientmsg.MatchActionType_MAT_JOIN) {
		JoinTable(msg.Charid, msg.Charname, msg.Matchmode, msg.Mapid, pmsg.Fromid, pmsg.Fromtype)
	} else if msg.Action == int32(clientmsg.MatchActionType_MAT_CANCEL) {
		table := getTableByCharID(msg.Charid)
		if table != nil {
			table.LeaveTable(msg.Charid, msg.Matchmode)
		}
	} else if msg.Action == int32(clientmsg.MatchActionType_MAT_CONFIRM) {
		table := getTableByCharID(msg.Charid)
		if table != nil {
			table.ConfirmTable(msg.Charid, msg.Matchmode)
		}
	} else if msg.Action == int32(clientmsg.MatchActionType_MAT_REJECT) {
		table := getTableByCharID(msg.Charid)
		if table != nil {
			table.RejectTable(msg.Charid, msg.Matchmode)
		}
	} else {
		log.Error("proxyHandleGSMSMatch Invalid Action %v", msg.Action)
	}
}

func proxyHandleGSMSReconnect(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_MS_Reconnect{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_GS_MS_Reconnect Decode Error %v", err)
		return
	}

	table := getTableByCharID(msg.Charid)
	if table != nil {
		rsp := table.ReconnectTable(msg.Charid)
		SendMessageTo(pmsg.Fromid, pmsg.Fromtype, msg.Charid, proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, rsp)
	} else {
		bench := getBenchByCharID(msg.Charid, true)
		if bench != nil {
			//todo
		} else {

		}
	}
}

func proxyHandleGSMSOffline(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_MS_Offline{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_GS_MS_Offline Decode Error %v", err)
		return
	}
	log.Debug("proxyHandleGSMSOffline CharID %v Offline", msg.Charid)

	table := getTableByCharID(msg.Charid)
	if table != nil {
		table.LeaveTable(msg.Charid, 0)
	}
	bench := getBenchByCharID(msg.Charid, true)
	if bench != nil {
		bench.leaveBench(msg.Charid, 0)
	}
}

func proxyHandleMSBSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_BS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_BS_AllocBattleRoom Decode Error %v", err)
		return
	}

	rsp := &proxymsg.Proxy_BS_MS_AllocBattleRoom{}
	err, roomid, battlekey := createRoom(msg)
	if err == nil {
		rsp.Retcode = 0
		rsp.Matchtableid = msg.Matchtableid
		rsp.Battleroomid = roomid
		rsp.Battleroomkey = battlekey
		rsp.Connectaddr = conf.Server.ConnectAddr
		rsp.Battleserverid = int32(conf.Server.ServerID)
	} else {
		rsp.Retcode = 1
		rsp.Matchtableid = msg.Matchtableid
	}

	log.Debug("proxyHandleMSBSAllocBattleRoom TableID %v RoomID %v", msg.Matchtableid, roomid)

	SendMessageTo(pmsg.Fromid, pmsg.Fromtype, 0, proxymsg.ProxyMessageType_PMT_BS_MS_ALLOCBATTLEROOM, rsp)
}

func proxyHandleBSMSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_MS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_BS_MS_AllocBattleRoom Decode Error %v", err)
		return
	}

	log.Debug("proxyHandleBSMSAllocBattleRoom RetCode %v TableID %v RoomID %v BattleServerID %v", msg.Retcode, msg.Matchtableid, msg.Battleroomid, msg.Battleserverid)

	table := getTableByTableID(msg.Matchtableid)
	if table != nil {
		table.ClearTable(msg)
	}
}

func proxyHandleMSGSMatchResult(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Rlt_Match{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Rlt_Match Decode Error %v", err)
		return
	}

	player, _ := GetPlayer(pmsg.Charid)
	if player == nil {
		return
	}

	if player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_MATCH {
		SendMsgToPlayer(pmsg.Charid, msg)
		if msg.RetCode == clientmsg.Type_GameRetCode_GRC_MATCH_ERROR { //匹配失败，返回大厅状态
			player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
			player.MatchServerID = 0
		}
	} else if player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_ONLINE {
		if msg.RetCode == clientmsg.Type_GameRetCode_GRC_MATCH_RECONNECT_OK {
			SendMsgToPlayer(pmsg.Charid, msg)
			player.MatchServerID = int(pmsg.Fromid)
			player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_MATCH)
		} else if msg.RetCode == clientmsg.Type_GameRetCode_GRC_MATCH_RECONNECT_ERROR {
			SendMsgToPlayer(pmsg.Charid, msg)
			player.MatchServerID = 0
		}
	}
}

func proxyHandleGSMSTeamOperate(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Transfer_Team_Operate{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Transfer_Team_Operate Error1 %v", err)
		return
	}

	table := getTableByCharID(pmsg.Charid)
	if table != nil {
		table.TeamOperate(pmsg.Charid, msg)
	}
}

func proxyHandleMSGSTeamOperate(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Transfer_Team_Operate{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Transfer_Team_Operate Error1 %v", err)
		return
	}

	SendMsgToPlayer(pmsg.Charid, msg)
}

func proxyHandleMSGSBeginBattle(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Rlt_NotifyBattleAddress{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("clientmsg.Rlt_NotifyBattleAddress Decode Error %v", err)
		return
	}
	player, err := GetPlayer(pmsg.Charid)
	if player != nil {
		if player.GetGamePlayerStatus() != clientmsg.UserStatus_US_PLAYER_OFFLINE {
			player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_BATTLE)
		}
		player.BattleServerID = int(msg.BattleServerID)
		player.MatchServerID = 0
	}

	log.Debug("proxyHandleMSGSBeginBattle Rlt_NotifyBattleAddress %v", pmsg.Charid)
	SendMsgToPlayer(pmsg.Charid, msg)
}

func proxyHandleGSBSQueryBattleInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_BS_Query_BattleInfo{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("Message Decode Error %v", err)
		return
	}

	rsp := &proxymsg.Proxy_BS_GS_Query_BattleInfo{
		CharID: msg.Charid,
	}

	room := getRoomByCharID(msg.Charid, false)
	if room == nil {
		rsp.InBattle, rsp.BattleKey, rsp.BattleAddr = false, nil, ""
	} else {
		rsp.InBattle, rsp.BattleKey, rsp.BattleAddr = true, room.battlekey, conf.Server.ConnectAddr
	}

	SendMessageTo(pmsg.Fromid, pmsg.Fromtype, 0, proxymsg.ProxyMessageType_PMT_BS_GS_QUERY_BATTLEINFO, rsp)
}

func proxyHandleBSGSQueryBattleInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_GS_Query_BattleInfo{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_BS_GS_Query_BattleInfo Decode Error %v", err)
		return
	}

	player, err := GetPlayer(msg.CharID)
	if err != nil {
		log.Error("proxyHandleBSGSQueryBattleInfo GetPlayer NULL %v", msg.CharID)
		return
	}

	log.Debug("Proxy_BS_GS_Query_BattleInfo %v", msg.CharID)
	if msg.InBattle {
		rsp := &clientmsg.Rlt_NotifyBattleAddress{
			RoomID:         msg.BattleRoomID,
			BattleAddr:     msg.BattleAddr,
			BattleKey:      msg.BattleKey,
			BattleServerID: pmsg.Fromid,
			IsReconnect:    true,
		}
		SendMsgToPlayer(msg.CharID, rsp)
	} else {
		player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
		player.BattleServerID = 0

		SendMsgToPlayer(player.Char.CharID, &clientmsg.Rlt_Re_Enter_Battle{RetCode: clientmsg.Type_BattleRetCode_BRC_ROOM_NOT_EXIST})
	}
}

func proxyHandleBSGSFinishBattle(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_GS_FINISH_BATTLE{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("Message Decode Error %v", err)
		return
	}

	player, err := GetPlayer(msg.CharID)
	if err != nil {
		log.Error("proxyHandleBSGSFinishBattle GetPlayer NULL %v", msg.CharID)
		return
	}

	if player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_BATTLE {
		player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
		player.BattleServerID = 0
	} else {
		player.BattleServerID = 0
	}
}

func proxyHandleBSMSSyncBSInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_MS_SyncBSInfo{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("Message Decode Error %v", err)
		return
	}

	UpdateBSOnlineManager(msg)
}

func proxyHandleGSMSMakeTeamOperate(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_MS_MakeTeamOperate{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("Message Decode Error %v", err)
		return
	}

	if msg.Action == int32(clientmsg.MakeTeamOperateType_MTOT_CREATE) {
		createBench(pmsg.Charid, msg.ActorCharname, msg.Matchmode, msg.Mapid, pmsg.Fromid, pmsg.Fromtype)
	} else if msg.Action == int32(clientmsg.MakeTeamOperateType_MTOT_LEAVE) {
		bench := getBenchByCharID(pmsg.Charid, false)
		if bench != nil {
			bench.leaveBench(pmsg.Charid, msg.Matchmode)
		}
	} else if msg.Action == int32(clientmsg.MakeTeamOperateType_MTOT_START_MATCH) {
		bench := getBenchByCharID(pmsg.Charid, false)
		if bench != nil {
			bench.startMatch(pmsg.Charid)
		}
	} else if msg.Action == int32(clientmsg.MakeTeamOperateType_MTOT_ACCEPT) {
		bench := getBenchByBenchID(msg.Benchid)
		if bench != nil {
			bench.acceptBench(pmsg.Charid, msg.ActorCharname, msg.Benchid, pmsg.Fromid, pmsg.Fromtype)
		}
	} else if msg.Action == int32(clientmsg.MakeTeamOperateType_MTOT_INVITE) {
		bench := getBenchByCharID(pmsg.Charid, false)
		if bench != nil {
			bench.inviteBench(pmsg.Charid, msg.Targetid, msg.Targetgsid)
		}
	} else if msg.Action == int32(clientmsg.MakeTeamOperateType_MTOT_KICK) {
		bench := getBenchByCharID(pmsg.Charid, false)
		if bench != nil {
			bench.kickBench(pmsg.Charid, msg.Targetid)
		}
	} else {
		log.Error("proxyHandleGSMSMakeTeamOperate Invalid Action %v", msg.Action)
	}
}

func proxyHandleMSGSMakeTeamOperate(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Rlt_MakeTeamOperate{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("Message Decode Error %v", err)
		return
	}

	player, _ := GetPlayer(pmsg.Charid)
	if player == nil {
		log.Error("MakeTeamOperate CharID %v Not Found", pmsg.Charid)
		return
	}

	if msg.RetCode == clientmsg.Type_GameRetCode_GRC_OK {
		if msg.Action == clientmsg.MakeTeamOperateType_MTOT_INVITE {
			if player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_ONLINE { //在线状态才通知邀请
				SendMsgToPlayer(pmsg.Charid, msg)
			}
		} else if msg.Action == clientmsg.MakeTeamOperateType_MTOT_ACCEPT {
			player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_BENCH)
			player.MatchServerID = int(pmsg.Fromid)
		} else {
			if msg.Action == clientmsg.MakeTeamOperateType_MTOT_START_MATCH {
				player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_MATCH)
			}

			SendMsgToPlayer(pmsg.Charid, msg)
		}
	} else { // Error , Full, List
		SendMsgToPlayer(pmsg.Charid, msg)
	}
}

func proxyHandleMSGSDelete(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_GS_Delete{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("Message Decode Error %v", err)
		return
	}

	log.Debug("Proxy_MS_GS_Delete CharID %v Reason %v", pmsg.Charid, msg.Reason)
	player, _ := GetPlayer(pmsg.Charid)
	if player == nil {
		log.Error("Proxy_MS_GS_Delete CharID %v Not Found", pmsg.Charid)
		return
	}

	player.MatchServerID = 0
}

func updateFrame(args []interface{}) {

	a := args[0].(time.Time)

	if lastTickTime != 0 && time.Now().UnixNano()-lastTickTime > 100000000 {
		log.Error("Slow FPS, lastframe cost %v s", float64(time.Now().UnixNano()-lastTickTime)/1000000000)
	}
	//log.Debug("Tick %v : Now %v", a, time.Now())

	UpdateBenchManager(&a)
	UpdateTableManager(&a)
	UpdateRoomManager(&a)
	UpdateGamePlayerManager(&a)
	UpdateBattlePlayerManager(&a)

	lastTickTime = a.UnixNano()
}

func rpcNewAgent(args []interface{}) {

	a := args[0].(gate.Agent)

	log.Debug("Connected %v From %v", a.RemoteAddr(), a.LocalAddr())
	_ = a
}

func rpcCloseAgent(args []interface{}) {
	a := args[0].(gate.Agent)

	clientid := a.UserData()
	if clientid != nil {
		RemoveBattlePlayer(clientid.(uint32), a.RemoteAddr().String(), REASON_DISCONNECT)
		RemoveGamePlayer(clientid.(uint32), a.RemoteAddr().String(), REASON_DISCONNECT)
	}

	log.Debug("Disconnected %v From %v", a.RemoteAddr(), a.LocalAddr())
	_ = a

}
