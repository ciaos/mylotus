package internal

import (
	"reflect"
	"time"
	//"server/conf"
	"server/msg/clientmsg"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/gate"
)

func init() {
	handler(&clientmsg.Ping{}, handlePing)
	handler(&clientmsg.Req_ServerTime{}, handleReqServerTime)
	handler(&clientmsg.Req_Login{}, handleReqLogin)
	handler(&clientmsg.Req_Match{}, handleReqMatch)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handlePing(args []interface{}) {
	m := args[0].(*clientmsg.Ping)
	a := args[1].(gate.Agent)

	a.WriteMsg(&clientmsg.Pong{ID: proto.Uint32(m.GetID())})

	//SendMessageTo(int32(conf.Server.ServerID), conf.Server.ServerType, uint64(1), uint32(0), m)
}

func handleReqServerTime(args []interface{}) {
	//	m := args[0].(*clientmsg.Req_ServerTime)
	a := args[1].(gate.Agent)

	a.WriteMsg(&clientmsg.Rlt_ServerTime{Time: proto.Uint32(uint32(time.Now().Unix()))})
}

func handleReqLogin(args []interface{}) {

}

func handleReqMatch(args []interface{}) {

}
