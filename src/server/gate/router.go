package gate

import (
	"server/game"
	"server/login"
	"server/msg"
	"server/msg/clientmsg"
)

func init() {

	//login server
	msg.Processor.SetRouter(&clientmsg.Req_Register{}, login.ChanRPC)
	msg.Processor.SetRouter(&clientmsg.Req_ServerList{}, login.ChanRPC)

	//game server
	msg.Processor.SetRouter(&clientmsg.Ping{}, game.ChanRPC)
	msg.Processor.SetRouter(&clientmsg.Req_ServerTime{}, game.ChanRPC)
	msg.Processor.SetRouter(&clientmsg.Req_Login{}, game.ChanRPC)
	msg.Processor.SetRouter(&clientmsg.Req_Match{}, game.ChanRPC)

	//battle server
	msg.Processor.SetRouter(&clientmsg.Req_ConnectBS{}, game.ChanRPC)
}
