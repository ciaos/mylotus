package test

import (
	"encoding/binary"
	"net"
	"server/msg/clientmsg"

	"time"

	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

const (
	LoginServerAddr = "127.0.0.1:8888"
	GameServerAddr  = LoginServerAddr

	GameServerID = 1
)

func SendAndRecvUtil(c *C, conn *net.Conn, msgid clientmsg.MessageType, msgdata interface{}, waitmsgid clientmsg.MessageType) []byte {
	Send(c, conn, msgid, msgdata)
	ch := make(chan []byte, 1)
	go func() {
		for {
			msgid, msgdata := Recv(c, conn)
			if msgid == waitmsgid {
				ch <- msgdata
				break
			}
		}
	}()

	select {
	case msgdata := <-ch:
		return msgdata
	case <-time.After(time.Second * 20):
		c.Fatal("Wait TimeOut")
	}
	return nil
}

func RecvUtil(c *C, conn *net.Conn, waitmsgid clientmsg.MessageType) []byte {
	ch := make(chan []byte, 1)
	go func() {
		for {
			msgid, msgdata := Recv(c, conn)
			if msgid == waitmsgid {
				ch <- msgdata
				break
			}
		}
	}()

	select {
	case msgdata := <-ch:
		return msgdata
	case <-time.After(time.Second * 20):
		c.Fatal("Wait TimeOut")
	}
	return nil
}

func Send(c *C, conn *net.Conn, msgid clientmsg.MessageType, msgdata interface{}) {
	data, err := proto.Marshal(msgdata.(proto.Message))
	if err != nil {
		c.Fatal("proto.Marshal ", err)
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(msgid))

	copy(reqbuf[4:], data)
	(*conn).Write(reqbuf)
}

func Recv(c *C, conn *net.Conn) (clientmsg.MessageType, []byte) {
	//Recv
	headdata := make([]byte, 2)
	(*conn).Read(headdata[0:])
	msglen := binary.BigEndian.Uint16(headdata[0:])

	bodydata := make([]byte, msglen)
	bodylen, _ := (*conn).Read(bodydata[0:])
	if msglen == 0 || bodylen == 0 {
		c.Fatal("empty buffer")
	}
	msgid := clientmsg.MessageType(binary.BigEndian.Uint16(bodydata[0:]))

	return msgid, bodydata[2:bodylen]
}

func Register(c *C, conn *net.Conn, username string, password string, islogin bool) (clientmsg.Type_LoginRetCode, uint32, []byte) {
	reqMsg := &clientmsg.Req_Register{
		UserName:      username,
		Password:      password,
		IsLogin:       islogin,
		ClientVersion: 0,
	}

	msgdata := SendAndRecvUtil(c, conn, clientmsg.MessageType_MT_REQ_REGISTER, reqMsg, clientmsg.MessageType_MT_RLT_REGISTER)
	rspMsg := &clientmsg.Rlt_Register{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Register Decode Error ", err)
	}
	return rspMsg.RetCode, rspMsg.UserID, rspMsg.SessionKey
}

func Login(c *C, conn *net.Conn, userid uint32, sessionkey []byte) (clientmsg.Type_GameRetCode, uint32, bool) {
	reqMsg := &clientmsg.Req_Login{
		UserID:     userid,
		SessionKey: sessionkey,
		ServerID:   GameServerID,
	}

	msgdata := SendAndRecvUtil(c, conn, clientmsg.MessageType_MT_REQ_LOGIN, reqMsg, clientmsg.MessageType_MT_RLT_LOGIN)
	rspMsg := &clientmsg.Rlt_Login{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Register Decode Error ", err)
	}

	return rspMsg.RetCode, rspMsg.CharID, rspMsg.IsNewCharacter
}

func QuickLogin(c *C, conn *net.Conn, username string, password string) uint32 {
	retcode, userid, sessionkey := Register(c, conn, username, password, false)
	if retcode == clientmsg.Type_LoginRetCode_LRC_ACCOUNT_EXIST {
		retcode, userid, sessionkey = Register(c, conn, username, password, true)
	}
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, charid, _ := Login(c, conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	return charid
}

func QuickMatch(c *C, conn *net.Conn) []byte {
	reqMsg := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType_MAT_JOIN,
		Mode:   clientmsg.MatchModeType_MMT_AI,
		MapID:  100,
	}

	msgdata := SendAndRecvUtil(c, conn, clientmsg.MessageType_MT_REQ_MATCH, reqMsg, clientmsg.MessageType_MT_RLT_MATCH)

	msgdata = RecvUtil(c, conn, clientmsg.MessageType_MT_RLT_MATCH)
	return msgdata
}
