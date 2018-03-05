package test

import (
	"encoding/binary"
	"net"
	"server/msg/clientmsg"
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestServerList(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	reqMsg := &clientmsg.Req_ServerList{
		Channel: proto.Int32(0),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed")
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_REQ_SERVERLIST))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := conn.Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_SERVERLIST:
		msg := &clientmsg.Rlt_ServerList{}
		proto.Unmarshal(rspbuf[4:len], msg)
		t.Log("Rlt_ServerList ", msg.GetServerCount())
		if msg.GetServerCount() > 0 {
			serverInfo := msg.GetServerList()[0]
			t.Log("Rlt_ServerList ", serverInfo.GetServerID(), serverInfo.GetServerName(), serverInfo.GetConnectAddr(), serverInfo.GetStatus())
		}
	default:
		t.Error("Invalid msgid ", msgid)
	}
}
