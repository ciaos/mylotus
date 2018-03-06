package test

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

func TestMatch(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	//Login First
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(100))
	password := "123456"
	QuickLogin(t, &conn, username, password)

	reqMsg := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType.Enum(clientmsg.MatchActionType_MAT_JOIN),
		Mode:   clientmsg.MatchModeType.Enum(clientmsg.MatchModeType_MMT_NORMAL),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed")
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_REQ_MATCH))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := conn.Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_MATCH:
		msg := &clientmsg.Rlt_Match{}
		proto.Unmarshal(rspbuf[4:len], msg)
		t.Log("Rlt_Match ", msg.GetRetCode())
	default:
		t.Error("Invalid msgid ", msgid)
	}
}
