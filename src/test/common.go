package test

import (
	"encoding/binary"
	"net"
	"server/msg/clientmsg"

	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

const (
	LoginServerAddr = "127.0.0.1:8888"
	GameServerAddr  = "127.0.0.1:8888"

	GameServerID = 1
)

func SendAndRecv(c *C, conn *net.Conn, msgid clientmsg.MessageType, msgdata interface{}) (clientmsg.MessageType, []byte) {

	//Send
	data, err := proto.Marshal(msgdata.(proto.Message))
	if err != nil {
		c.Fatal("proto.Marshal ", err)
	}
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(msgid))

	copy(reqbuf[4:], data)
	(*conn).Write(reqbuf)

	//Recv
	headdata := make([]byte, 2)
	(*conn).Read(headdata[0:])
	msglen := binary.BigEndian.Uint16(headdata[0:])

	bodydata := make([]byte, msglen)
	bodylen, _ := (*conn).Read(bodydata[0:])
	if msglen == 0 || bodylen == 0 {
		c.Fatal("empty buffer")
	}
	msgid = clientmsg.MessageType(binary.BigEndian.Uint16(bodydata[0:]))

	return msgid, bodydata[2:bodylen]
}

func Register(c *C, conn *net.Conn, username string, password string, islogin bool) (clientmsg.Type_LoginRetCode, string, []byte) {
	reqMsg := &clientmsg.Req_Register{
		UserName:      proto.String(username),
		Password:      proto.String(password),
		IsLogin:       proto.Bool(islogin),
		ClientVersion: proto.Int32(0),
	}

	msgid, msgdata := SendAndRecv(c, conn, clientmsg.MessageType_MT_REQ_REGISTER, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_REGISTER)
	rspMsg := &clientmsg.Rlt_Register{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Register Decode Error ", err)
	}
	return rspMsg.GetRetCode(), rspMsg.GetUserID(), rspMsg.GetSessionKey()
}

func Login(c *C, conn *net.Conn, userid string, sessionkey []byte) (clientmsg.Type_GameRetCode, string, bool) {
	reqMsg := &clientmsg.Req_Login{
		UserID:     proto.String(userid),
		SessionKey: sessionkey,
		ServerID:   proto.Int32(GameServerID),
	}

	msgid, msgdata := SendAndRecv(c, conn, clientmsg.MessageType_MT_REQ_LOGIN, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_LOGIN)
	rspMsg := &clientmsg.Rlt_Login{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Register Decode Error ", err)
	}

	return rspMsg.GetRetCode(), rspMsg.GetCharID(), rspMsg.GetIsNewCharacter()
}

func QuickLogin(c *C, conn *net.Conn, username string, password string) string {
	retcode, userid, sessionkey := Register(c, conn, username, password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_NONE)

	code, charid, _ := Login(c, conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_NONE)
	return charid
}

func QuickMatch(c *C, conn *net.Conn) (clientmsg.MessageType, []byte) {
	reqMsg := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType.Enum(clientmsg.MatchActionType_MAT_JOIN),
		Mode:   clientmsg.MatchModeType.Enum(clientmsg.MatchModeType_MMT_NORMAL),
	}

	msgid, msgdata := SendAndRecv(c, conn, clientmsg.MessageType_MT_REQ_MATCH, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_NOTIFYBATTLEADDRESS)
	return msgid, msgdata
}
