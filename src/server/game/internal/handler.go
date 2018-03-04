package internal

import (
	"reflect"
	//"server/conf"
	"server/msg/clientmsg"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/gate"
)

func init() {
	handler(&clientmsg.Ping{}, handlePing)
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
