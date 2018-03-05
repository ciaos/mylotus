package internal

import (
	"server/game/internal/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
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
	switch proxyMsg.GetMsgid() {
	case 0:
		innerMsg := &clientmsg.Ping{}
		err = proto.Unmarshal(proxyMsg.GetMsgdata(), innerMsg)
		if err != nil {
			log.Error("queueMessage Hello Decode Error %v", err)
			return
		}
		log.Debug("Recv %v", innerMsg.GetID())
	default:
		log.Error("Invalid InnerMsg ID %v", proxyMsg.GetMsgid())
	}
}

func updateFrame(args []interface{}) {

	a := args[0].(time.Time)
	log.Debug("Tick %v : %v : %v", time.Now().Unix(), time.Now().UnixNano(), a)

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
