package internal

import (
	"fmt"
	"server/conf"
	"server/game/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"server/tool"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
)

func init() {
	skeleton.RegisterChanRPC("NewAgent", rpcNewAgent)
	skeleton.RegisterChanRPC("CloseAgent", rpcCloseAgent)
	skeleton.RegisterChanRPC("TickFrame", updateFrame)
	skeleton.RegisterChanRPC("QueueMessage", queueMessage)
}

func queueMessage(args []interface{}) {

	log.Debug("queueMessage Len %v", len(args))
	proxyMsg := &proxymsg.InternalMessage{}
	err := proto.Unmarshal(args[0].([]byte), proxyMsg)
	if err != nil {
		log.Error("queueMessage InnerMsg Decode Error %v", err)
		return
	}
	switch proxymsg.ProxyMessageType(proxyMsg.GetMsgid()) {
	case proxymsg.ProxyMessageType_PMT_GS_MS_MATCH:
		proxyHandleGSMSMatch(proxyMsg)
	case proxymsg.ProxyMessageType_PMT_MS_BS_ALLOCBATTLEROOM:
		proxyHandleMSBSAllocBattleRoom(proxyMsg)
	default:
		log.Error("Invalid InnerMsg ID %v", proxyMsg.GetMsgid())
	}
}

func proxyHandleGSMSMatch(proxyMsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_GS_MS_Match{}
	err := proto.Unmarshal(proxyMsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_GS_MS_Match Decode Error %v", err)
		return
	}
	log.Debug("proxyHandleGSMSMatch %v", msg.GetCharid(), msg.GetAction())

	if msg.GetAction() == int32(clientmsg.MatchActionType_MAT_JOIN) {
		g.JoinTable(msg.GetCharid(), msg.GetMatchmode())
	} else if msg.GetAction() == int32(clientmsg.MatchActionType_MAT_CANCEL) {
		g.LeaveTable(msg.GetCharid(), msg.GetMatchmode())
	} else {
		log.Error("proxyHandleGSMSMatch invalid action %v", msg.GetAction())
	}
}

func proxyHandleMSBSAllocBattleRoom(proxyMsg *proxymsg.InternalMessage) {
	msg := &proxymsg.Proxy_MS_BS_AllocBattleRoom{}
	err := proto.Unmarshal(proxyMsg.GetMsgdata(), msg)
	if err != nil {
		log.Error("proxymsg.Proxy_MS_BS_AllocBattleRoom Decode Error %v", err)
		return
	}
	log.Debug("proxyHandleMSBSAllocBattleRoom %v", msg.GetMatchroomid())

	roomid := g.CreateRoom(msg.GetMatchmode())

	battlekey, _ := tool.DesEncrypt([]byte(fmt.Sprintf("room%d", roomid)), []byte(tool.CRYPT_KEY))

	rsp := &proxymsg.Proxy_BS_MS_AllocBattleRoom{
		Retcode:        proto.Int32(0),
		Matchroomid:    proto.Int32(msg.GetMatchroomid()),
		Battleroomid:   proto.Int32(roomid),
		Battleserverid: proto.Int32(int32(conf.Server.ServerID)),
		Connectaddr:    proto.String(conf.Server.TCPAddr),
		Battleroomkey:  battlekey,
	}

	g.SendMessageTo(proxyMsg.GetFromid(), proxyMsg.GetFromtype(), "", uint32(proxymsg.ProxyMessageType_PMT_BS_MS_ALLOCBATTLEROOM), rsp)
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

	charid := a.UserData()
	log.Debug("Disconnected %v", a.RemoteAddr())
	_ = a

	if charid != nil {
		delete(g.PlayerManager, charid.(string))
		log.Debug("PlayerManager Remove %v", charid)
	}
}
