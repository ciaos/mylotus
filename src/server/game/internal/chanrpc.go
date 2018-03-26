package internal

import (
	"server/conf"
	"server/game/g"
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

func queueMessage(args []interface{}) {
	pmsg := &proxymsg.InternalMessage{}
	err := proto.Unmarshal(args[0].([]byte), pmsg)
	if err != nil {
		log.Error("queueMessage InnerMsg Decode Error %v", err)
		return
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
	case proxymsg.ProxyMessageType_PMT_GS_MS_TEAM_OPERATE:
		proxyHandleGSMSTeamOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_TEAM_OPERATE:
		proxyHandleMSGSTeamOperate(pmsg)
	case proxymsg.ProxyMessageType_PMT_GS_MS_OFFLINE:
		proxyHandleGSMSOffline(pmsg)
	default:
		log.Error("Invalid InnerMsg ID %v", pmsg.Msgid)
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
		g.JoinTable(msg.Charid, msg.Charname, msg.Matchmode, msg.Mapid, pmsg.Fromid, pmsg.Fromtype)
	} else if msg.Action == int32(clientmsg.MatchActionType_MAT_CANCEL) {
		g.LeaveTable(msg.Charid, msg.Matchmode)
	} else if msg.Action == int32(clientmsg.MatchActionType_MAT_CONFIRM) {
		g.ConfirmTable(msg.Charid, msg.Matchmode)
	} else {
		log.Error("proxyHandleGSMSMatch Invalid Action %v", msg.Action)
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
	g.LeaveTable(msg.Charid, 0)
}

func proxyHandleMSBSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_BS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_BS_AllocBattleRoom Decode Error %v", err)
		return
	}

	roomid, battlekey := g.CreateRoom(msg)

	rsp := &proxymsg.Proxy_BS_MS_AllocBattleRoom{
		Retcode:        0,
		Matchtableid:   msg.Matchtableid,
		Battleroomid:   roomid,
		Battleroomkey:  battlekey,
		Connectaddr:    conf.Server.ConnectAddr,
		Battleserverid: int32(conf.Server.ServerID),
	}

	log.Debug("proxyHandleMSBSAllocBattleRoom TableID %v RoomID %v", msg.Matchtableid, roomid)

	skeleton.Go(func() {
		g.SendMessageTo(pmsg.Fromid, pmsg.Fromtype, 0, uint32(proxymsg.ProxyMessageType_PMT_BS_MS_ALLOCBATTLEROOM), rsp)
	}, func() {})
}

func proxyHandleBSMSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_MS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_BS_MS_AllocBattleRoom Decode Error %v", err)
		return
	}

	log.Debug("proxyHandleBSMSAllocBattleRoom RetCode %v TableID %v RoomID %v BattleServerID %v", msg.Retcode, msg.Matchtableid, msg.Battleroomid, msg.Battleserverid)

	g.ClearTable(msg)
}

func proxyHandleMSGSMatchResult(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Rlt_Match{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Rlt_Match Decode Error %v", err)
		return
	}

	g.SendMsgToPlayer(pmsg.Charid, msg)
}

func proxyHandleGSMSTeamOperate(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Transfer_Team_Operate{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Transfer_Team_Operate Error1 %v", err)
		return
	}
	//log.Debug("proxyHandleGSMSTeamOperate %v %v %v %v", pmsg.Charid, msg.Action, msg.CharID, msg.CharType)

	g.TeamOperate(pmsg.Charid, msg)
}

func proxyHandleMSGSTeamOperate(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Transfer_Team_Operate{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Transfer_Team_Operate Error1 %v", err)
		return
	}

	//log.Debug("Transfer_Team_Operate %v %v %v %v", pmsg.Charid, msg.Action, msg.CharID, msg.CharType)
	g.SendMsgToPlayer(pmsg.Charid, msg)
}

func proxyHandleMSGSBeginBattle(pmsg *proxymsg.InternalMessage) {
	msg := &clientmsg.Rlt_NotifyBattleAddress{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Rlt_NotifyBattleAddress Decode Error %v", err)
		return
	}
	player, err := g.GetPlayer(pmsg.Charid)
	if player != nil {
		player.BattleServerID = int(msg.BattleServerID)
		player.MatchServerID = 0
	}

	g.SendMsgToPlayer(pmsg.Charid, msg)
}

func updateFrame(args []interface{}) {

	a := args[0].(time.Time)
	//log.Debug("Tick %v : %v : %v", time.Now().Unix(), time.Now().UnixNano(), a)

	g.UpdateTableManager(&a)
	g.UpdateRoomManager(&a)
	g.UpdatePlayerManager(&a)
}

func rpcNewAgent(args []interface{}) {

	a := args[0].(gate.Agent)

	log.Debug("Connected %v", a.RemoteAddr())
	_ = a
}

func rpcCloseAgent(args []interface{}) {
	a := args[0].(gate.Agent)

	clientid := a.UserData()
	log.Debug("Disconnected %v", a.RemoteAddr())
	_ = a

	if clientid != nil {
		g.RemoveBattlePlayer(clientid.(uint32), a.RemoteAddr().String())
		g.RemoveGamePlayer(clientid.(uint32), a.RemoteAddr().String(), false)
	}
}
