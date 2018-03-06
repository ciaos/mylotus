package test

import (
	"encoding/binary"
	"errors"
	"net"
	"server/msg/clientmsg"
	"testing"

	"github.com/golang/protobuf/proto"
)

const (
	TestServerAddr = "127.0.0.1:8888"

	GameServerID = 1
)

func Register(t *testing.T, conn *net.Conn, username string, password string, islogin bool, checkCode clientmsg.Type_LoginRetCode) (string, []byte, error) {

	reqMsg := &clientmsg.Req_Register{
		UserName:      proto.String(username),
		Password:      proto.String(password),
		IsLogin:       proto.Bool(islogin),
		ClientVersion: proto.Int32(0),
	}

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
			t.Error("RetCode MisMatch ", msg.GetRetCode())
			t.Error("RetCode MisMatch ", checkCode)
		} else {
			return msg.GetUserID(), msg.GetSessionKey(), nil
		}
	default:
		t.Error("Invalid msgid ", msgid)
	}

	return "", nil, errors.New("register error")
}

func Login(t *testing.T, conn *net.Conn, userid string, sessionkey []byte, checkCode clientmsg.Type_GameRetCode) (string, error) {
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
	(*conn).Write(reqbuf)
	rspbuf := make([]byte, 2014)
	len, _ := (*conn).Read(rspbuf[0:])

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch clientmsg.MessageType(msgid) {
	case clientmsg.MessageType_MT_RLT_LOGIN:
		msg := &clientmsg.Rlt_Login{}
		proto.Unmarshal(rspbuf[4:len], msg)
		if msg.GetRetCode() == checkCode {
			return msg.GetCharID(), nil
		}
	default:
		t.Error("Invalid msgid ", msgid)
	}

	return "", errors.New("login error")
}

func QuickLogin(t *testing.T, conn *net.Conn, username string, password string) string {
	userid, sessionkey, err := Register(t, conn, username, password, false, clientmsg.Type_LoginRetCode_LRC_NONE)
	if err != nil {
		t.Fatal("Register Error", err)
	}

	t.Log("Login UserID", userid)
	charid, err := Login(t, conn, userid, sessionkey, clientmsg.Type_GameRetCode_GRC_NONE)
	if err != nil {
		t.Fatal("Login Error", err)
	}
	t.Log("Login OK CharID", charid)
	return charid
}
