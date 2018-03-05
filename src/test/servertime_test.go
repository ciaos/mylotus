package test

import (
	"encoding/binary"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

func TestServerTime(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	reqMsg := &clientmsg.Req_ServerTime{
		Time: proto.Uint32(uint32(time.Now().Unix())),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed")
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_REQ_SERVERTIME))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := conn.Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_SERVERTIME:
		msg := &clientmsg.Rlt_ServerTime{}
		proto.Unmarshal(rspbuf[4:len], msg)
		t.Log("Rlt_ServerTime ", msg.GetTime())
	default:
		t.Error("Invalid msgid ", msgid)
	}
}
