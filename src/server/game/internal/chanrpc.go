package internal

import (
	"fmt"
	"server/conf"
	"server/game/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
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
	switch proxymsg.ProxyMessageType(pmsg.GetMsgid()) {
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
		log.Error("Invalid InnerMsg ID %v", pmsg.GetMsgid())
	}
}

func proxyHandleGSMSMatch(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_MS_Match{}
	err := proto.Unmarshal(pmsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_GS_MS_Match Decode Error %v", err)
		return
	}
	log.Debug("proxyHandleGSMSMatch %v", msg.GetCharid(), msg.GetAction())

	if msg.GetAction() == int32(clientmsg.MatchActionType_MAT_JOIN) {
		g.JoinTable(msg.GetCharid(), msg.GetMatchmode(), *pmsg.Fromid, *pmsg.Fromtype)
	} else if msg.GetAction() == int32(clientmsg.MatchActionType_MAT_CANCEL) {
		g.LeaveTable(msg.GetCharid(), msg.GetMatchmode())
	} else {
		log.Error("proxyHandleGSMSMatch invalid action %v", msg.GetAction())
	}
}

func proxyHandleMSBSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_BS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_BS_AllocBattleRoom Decode Error %v", err)
		return
	}
	log.Debug("proxyHandleMSBSAllocBattleRoom %v", msg.GetMatchroomid())

	roomid := g.CreateRoom(msg.GetMatchmode())

	rsp := &proxymsg.Proxy_BS_MS_AllocBattleRoom{
		Retcode:          proto.Int32(0),
		Matchroomid:      proto.Int32(msg.GetMatchroomid()),
		Battleroomid:     proto.Int32(roomid),
		Battleserverid:   proto.Int32(int32(conf.Server.ServerID)),
		Battleservername: proto.String(conf.Server.ServerType),
	}

	g.SendMessageTo(pmsg.GetFromid(), pmsg.GetFromtype(), "", uint32(proxymsg.ProxyMessageType_PMT_BS_MS_ALLOCBATTLEROOM), rsp)
}

func proxyHandleBSMSAllocBattleRoom(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_MS_AllocBattleRoom{}
	err := proto.Unmarshal(pmsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_BS_MS_AllocBattleRoom Decode Error %v", err)
		return
	}

	g.ClearTable(msg.GetMatchroomid(), msg.GetBattleroomid(), msg.GetBattleserverid(), msg.GetBattleservername())
}

func proxyHandleMSGSMatchResult(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_GS_MatchResult{}
	err := proto.Unmarshal(pmsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_GS_MatchResult Decode Error %v", err)
		return
	}

	_, ok := g.GamePlayerManager[pmsg.GetCharid()]
	if !ok {
		log.Error("proxyHandleMSGSMatchResult g.GamePlayerManager Not Found %v", pmsg.GetCharid())
		return
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB("game").C(fmt.Sprintf("userinfo_%d", conf.Server.ServerID))

	result := g.UserInfo{}
	err = c.Find(bson.M{"charid": pmsg.GetCharid()}).One(&result)
	if err != nil {
		log.Error("userinfo not found %v", pmsg.GetCharid())
		return
	}

	req := &proxymsg.Proxy_GS_BS_SyncPlayerInfo{
		Charid:       proto.String(pmsg.GetCharid()),
		Charname:     proto.String(result.CharName),
		Chartype:     proto.Int32(0),
		Teamtype:     proto.Int32(0),
		Battleroomid: proto.Int32(msg.GetBattleroomid()),
	}
	g.SendMessageTo(msg.GetBattleserverid(), msg.GetBattleservername(), "", uint32(proxymsg.ProxyMessageType_PMT_GS_BS_SYNCPLAYERINFO), req)
}

func proxyHandleGSBSSyncPlayerInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_BS_SyncPlayerInfo{}
	err := proto.Unmarshal(pmsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_GS_MatchResult Decode Error %v", err)
		return
	}

	battlekey := g.JoinRoom(msg.GetCharid(), msg.GetBattleroomid(), msg.GetCharname(), msg.GetChartype())
	if battlekey != nil {
		rsp := &proxymsg.Proxy_BS_GS_SyncPlayerInfo{
			Retcode:       proto.Int32(0),
			Battleroomid:  proto.Int32(msg.GetBattleroomid()),
			Battleroomkey: battlekey,
			Connectaddr:   proto.String(conf.Server.TCPAddr),
		}

		g.SendMessageTo((*pmsg).GetFromid(), (*pmsg).GetFromtype(), msg.GetCharid(), uint32(proxymsg.ProxyMessageType_PMT_BS_GS_SYNCPLAYERINFO), rsp)
	} else {
		log.Error("proxyHandleGSBSSyncPlayerInfo JoinRoom Error")
	}
}

func proxyHandleBSGSSyncPlayerInfo(pmsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_BS_GS_SyncPlayerInfo{}
	err := proto.Unmarshal(pmsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_GS_MatchResult Decode Error %v", err)
		return
	}

	if msg.GetRetcode() == 0 {
		agent, ok := g.GamePlayerManager[pmsg.GetCharid()]
		if ok {
			rsp := &clientmsg.Rlt_NotifyBattleAddress{
				RoomID:     proto.Int32(msg.GetBattleroomid()),
				BattleAddr: proto.String(msg.GetConnectaddr()),
				BattleKey:  msg.GetBattleroomkey(),
			}

			(*agent).WriteMsg(rsp)
		} else {
			log.Error("proxyHandleBSGSSyncPlayerInfo %v CharID Not Found", pmsg.GetCharid())
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
		g.RemoveBattlePlayer(clientid.(string), a.RemoteAddr().String())
		g.RemoveGamePlayer(clientid.(string), a.RemoteAddr().String())
	}
}
