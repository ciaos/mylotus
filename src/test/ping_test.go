package test

import (
	"encoding/binary"
	"net"
	"server/msg/clientmsg"
	"testing"

	"github.com/golang/protobuf/proto"
)

const (
	address = "127.0.0.1:8888"
)

func TestPing(t *testing.T) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	reqMsg := &clientmsg.Ping{
		ID: proto.Uint32(11),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed")
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_PING))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := conn.Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_PONG:
		msg := &clientmsg.Pong{}
		proto.Unmarshal(rspbuf[4:len], msg)
		t.Log("Recv 0 ", msg.GetID())
	default:
		t.Error("Invalid msgid ", msgid)
	}
}
