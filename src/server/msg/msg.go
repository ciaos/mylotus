package msg

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/network/protobuf"
)

var Processor = protobuf.NewProcessor()

func init() {
	//0
	Processor.Register(&clientmsg.Ping{})

	//1 - 10
	Processor.Register(&clientmsg.Pong{})
	Processor.Register(&clientmsg.Req_ServerTime{})
	Processor.Register(&clientmsg.Rlt_ServerTime{})
	Processor.Register(&clientmsg.Req_Register{})
	Processor.Register(&clientmsg.Rlt_Register{})
	Processor.Register(&clientmsg.Req_ServerList{})
	Processor.Register(&clientmsg.Rlt_ServerList{})
	Processor.Register(&clientmsg.Req_Login{})
	Processor.Register(&clientmsg.Rlt_Login{})
	Processor.Register(&clientmsg.Req_SetCharName{})

	//11 - 20
	Processor.Register(&clientmsg.Rlt_SetCharName{})
	Processor.Register(&clientmsg.Req_Match{})
	Processor.Register(&clientmsg.Rlt_Match{})
	Processor.Register(&clientmsg.Rlt_NotifyBattleAddress{})
	Processor.Register(&clientmsg.Req_ConnectBS{})
	Processor.Register(&clientmsg.Rlt_ConnectBS{})
	Processor.Register(&clientmsg.Rlt_StartBattle{})
	Processor.Register(&clientmsg.Req_EndBattle{})
	Processor.Register(&clientmsg.Rlt_EndBattle{})
	Processor.Register(&clientmsg.Transfer_Command{})

	//21 - 30
	Processor.Register(&clientmsg.Transfer_Loading_Progress{})
	Processor.Register(&clientmsg.Transfer_Team_Operate{})
}
