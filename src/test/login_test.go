package test

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

func register(t *testing.T, conn *net.Conn, reqMsg *clientmsg.Req_Register, checkCode clientmsg.Type_LoginRetCode) (string, []byte, error) {

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed")
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_REQ_REGISTER))

	copy(reqbuf[4:], data)

	// 发送消息
	(*conn).Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := (*conn).Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_REGISTER:
		msg := &clientmsg.Rlt_Register{}
		proto.Unmarshal(rspbuf[4:len], msg)
		if msg.GetRetCode() != checkCode {
			t.Error("RetCode MisMatch ", msg.GetRetCode(), checkCode)
		} else {
			return msg.GetUserID(), msg.GetSessionKey(), nil
		}
	default:
		t.Error("Invalid msgid ", msgid)
	}

	return "", nil, errors.New("register error")
}

func TestLogin(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	//Register First
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(100))
	registerMsg := &clientmsg.Req_Register{
		UserName:      proto.String(username),
		Password:      proto.String("123456"),
		IsLogin:       proto.Bool(false),
		ClientVersion: proto.Int32(0),
	}

	t.Log("Register Username", username)
	userid, sessionkey, err := register(t, &conn, registerMsg, clientmsg.Type_LoginRetCode_LRC_NONE)
	if err != nil {
		t.Fatal("Register Error")
	}
	reqMsg := &clientmsg.Req_Login{
		UserID:     proto.String(userid),
		SessionKey: sessionkey,
		ServerID:   proto.Int32(GameServerID),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		t.Fatal("Marsha1 failed ", err)
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(clientmsg.MessageType_MT_REQ_LOGIN))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := conn.Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_LOGIN:
		msg := &clientmsg.Rlt_Login{}
		proto.Unmarshal(rspbuf[4:len], msg)
		t.Log("Rlt_Login ", msg.GetRetCode(), msg.GetCharID(), msg.GetIsNewCharacter())
	default:
		t.Error("Invalid msgid ", msgid)
	}
}
