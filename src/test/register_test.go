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
			t.Log("Rlt_Register ", msg.GetRetCode())
			return msg.GetUserID(), msg.GetSessionKey(), nil
		}
	default:
		t.Error("Invalid msgid ", msgid)
	}

	return "", nil, errors.New("register error")
}

func TestRegister(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	reqMsg := &clientmsg.Req_Register{
		UserName:      proto.String(username),
		Password:      proto.String("123456"),
		IsLogin:       proto.Bool(true),
		ClientVersion: proto.Int32(0),
	}

	t.Log("Register Username", username)
	register(t, &conn, reqMsg, clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST)

	reqMsg.IsLogin = proto.Bool(false)
	userid, sessionkey, _ := register(t, &conn, reqMsg, clientmsg.Type_LoginRetCode_LRC_NONE)
	t.Log("Register UserID ", userid, " SessionKey", sessionkey)
	register(t, &conn, reqMsg, clientmsg.Type_LoginRetCode_LRC_ACCOUNT_EXIST)

	reqMsg.IsLogin = proto.Bool(true)
	userid, sessionkey, _ = register(t, &conn, reqMsg, clientmsg.Type_LoginRetCode_LRC_NONE)
	t.Log("Register UserID ", userid, " SessionKey", sessionkey)
	reqMsg.Password = proto.String("1234")
	register(t, &conn, reqMsg, clientmsg.Type_LoginRetCode_LRC_PASSWORD_ERROR)
}
