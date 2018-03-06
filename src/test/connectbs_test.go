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

func TestConnectBS(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	//Login First
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(100))
	password := "123456"
	t.Log("Username ", username)
	charid := QuickLogin(t, &conn, username, password)

	t.Log("CharID ", charid)
	err, roomid, battleaddr, battlekey := QuickMatch(t, &conn)
	if err != nil {
		t.Fatal("Match Error", err)
	}

	t.Log("Match Result ", roomid, battleaddr, battlekey)

	CloseConnection(&conn)
	//connect bs
	conn, err = net.Dial("tcp", battleaddr)
	if err != nil {
		t.Fatal("Connect Battle Error ", err)
	}

	reqMsg := &clientmsg.Req_ConnectBS{
		RoomID:    proto.Int32(roomid),
		BattleKey: battlekey,
		CharID:    proto.String(charid),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed")
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_REQ_CONNECTBS))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := conn.Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_CONNECTBS:
		msg := &clientmsg.Rlt_ConnectBS{}
		proto.Unmarshal(rspbuf[4:len], msg)
		t.Log("Rlt_ConnectBS ", msg.GetRetCode())
	default:
		t.Error("Invalid msgid ", msgid)
	}
}
