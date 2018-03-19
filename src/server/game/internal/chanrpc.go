package internal

import (
	"fmt"
	"server/conf"
	"server/game/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"github.com/golang/protobuf/proto"
	"gopkg.in/mgo.v2/bson"
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
	case proxymsg.ProxyMessageType_PMT_GS_BS_SYNCPLAYERINFO:
		proxyHandleGSBSSyncPlayerInfo(pmsg)
	case proxymsg.ProxyMessageType_PMT_BS_GS_SYNCPLAYERINFO:
		proxyHandleBSGSSyncPlayerInfo(pmsg)
	case proxymsg.ProxyMessageType_PMT_MS_GS_MATCHRESULT:
		proxyHandleMSGSMatchResult(pmsg)
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
		g.JoinTable(msg.Charid, msg.Matchmode, pmsg.Fromid, pmsg.Fromtype)
	} else if msg.Action == int32(clientmsg.MatchActionType_MAT_CANCEL) {
		g.LeaveTable(msg.Charid, msg.Matchmode)
	} else {
		log.Error("proxyHandleGSMSMatch Invalid Action %v", msg.Action)
	}
}

func proxyHandleMSBSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_BS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_BS_AllocBattleRoom Decode Error %v", err)
		return
	}

	roomid := g.CreateRoom(msg.Matchmode, msg.Membercnt)

	rsp := &proxymsg.Proxy_BS_MS_AllocBattleRoom{
		Retcode:          0,
		Matchroomid:      msg.Matchroomid,
		Battleroomid:     roomid,
		Battleserverid:   int32(conf.Server.ServerID),
		Battleservername: conf.Server.ServerType,
	}

	log.Debug("proxyHandleMSBSAllocBattleRoom TableID %v RoomID %v", msg.Matchroomid, roomid)

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

	log.Debug("proxyHandleBSMSAllocBattleRoom RetCode %v TableID %v RoomID %v BattleServerID %v", msg.Retcode, msg.Matchroomid, msg.Battleroomid, msg.Battleserverid)

	g.ClearTable(msg.Matchroomid, msg.Battleroomid, msg.Battleserverid, msg.Battleservername)
}

func proxyHandleMSGSMatchResult(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_GS_MatchResult{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_GS_MatchResult Decode Error %v", err)
		return
	}

	_, ok := g.GamePlayerManager[pmsg.Charid]
	if !ok {
		log.Error("proxyHandleMSGSMatchResult g.GamePlayerManager Not Found %v", pmsg.Charid)
		return
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB("game").C(fmt.Sprintf("userinfo_%d", conf.Server.ServerID))

	result := g.UserInfo{}
	err = c.Find(bson.M{"charid": pmsg.Charid}).One(&result)
	if err != nil {
		log.Error("userinfo not found %v", pmsg.Charid)
		return
	}

	req := &proxymsg.Proxy_GS_BS_SyncPlayerInfo{
		Charid:       pmsg.Charid,
		Charname:     result.CharName,
		Chartype:     0,
		Teamtype:     0,
		Battleroomid: msg.Battleroomid,
	}

	log.Debug("proxyHandleMSGSMatchResult SyncPlayerInfo CharID %v RoomID %v", pmsg.Charid, msg.Battleroomid)

	skeleton.Go(func() {
		g.SendMessageTo(msg.Battleserverid, msg.Battleservername, 0, uint32(proxymsg.ProxyMessageType_PMT_GS_BS_SYNCPLAYERINFO), req)
	}, func() {})
}

func proxyHandleGSBSSyncPlayerInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_BS_SyncPlayerInfo{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_GS_MatchResult Decode Error %v", err)
		return
	}

	log.Debug("proxyHandleGSBSSyncPlayerInfo SyncPlayerInfo CharID %v RoomID %v GameServerID %v", msg.Charid, msg.Battleroomid, pmsg.Fromid)

	battlekey := g.JoinRoom(msg.Charid, msg.Battleroomid, msg.Charname, msg.Chartype, pmsg.Fromid)
	if battlekey != nil {

		rsp := &proxymsg.Proxy_BS_GS_SyncPlayerInfo{
			Retcode:       0,
			Battleroomid:  msg.Battleroomid,
			Battleroomkey: battlekey,
			Connectaddr:   conf.Server.ConnectAddr,
		}

		skeleton.Go(func() {
			g.SendMessageTo((*pmsg).Fromid, (*pmsg).Fromtype, msg.Charid, uint32(proxymsg.ProxyMessageType_PMT_BS_GS_SYNCPLAYERINFO), rsp)
		}, func() {})
	} else {
		log.Error("proxyHandleGSBSSyncPlayerInfo JoinRoom Error")
	}
}

func proxyHandleBSGSSyncPlayerInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_GS_SyncPlayerInfo{}
	err := proto.Unmarshal(pmsg.Msgdata, msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_GS_MatchResult Decode Error %v", err)
		return
	}

	log.Debug("proxyHandleBSGSSyncPlayerInfo Notify CharID %v RoomID %v RoomAddr %v", pmsg.Charid, msg.Battleroomid, msg.Connectaddr)

	if msg.Retcode == 0 {
		agent, ok := g.GamePlayerManager[pmsg.Charid]
		if ok {
			rsp := &clientmsg.Rlt_NotifyBattleAddress{
				RoomID:     msg.Battleroomid,
				BattleAddr: msg.Connectaddr,
				BattleKey:  msg.Battleroomkey,
			}

			(*agent).WriteMsg(rsp)
		} else {
			log.Error("proxyHandleBSGSSyncPlayerInfo %v CharID Not Found", pmsg.Charid)
		}
	}
}

func updateFrame(args []interface{}) {

	a := args[0].(time.Time)
	//log.Debug("Tick %v : %v : %v", time.Now().Unix(), time.Now().UnixNano(), a)

	g.UpdateTableManager(&a)
	g.UpdateRoomManager(&a)
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
		g.RemoveGamePlayer(clientid.(uint32), a.RemoteAddr().String())
	}
}
