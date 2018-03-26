package test

import (
	"fmt"
	"net"
	"server/msg/clientmsg"
	"testing"

	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

func TestFriend(t *testing.T) { TestingT(t) }

type FriendSuite struct {
	conn_1   net.Conn
	err_1    error
	charid_1 uint32

	conn_2   net.Conn
	err_2    error
	charid_2 uint32
}

var _ = Suite(&FriendSuite{})

func (s *FriendSuite) SetUpSuite(c *C) {
	s.conn_1, s.err_1 = net.Dial("tcp", GameServerAddr)
	if s.err_1 != nil {
		c.Fatal("Connect Server Error ", s.err_1)
	}
	username := fmt.Sprintf("robot_%d", 1)
	password := "123456"

	retcode, userid, sessionkey := Register(c, &s.conn_1, username, password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, charid, _ := Login(c, &s.conn_1, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)

	s.charid_1 = charid

	s.conn_2, s.err_2 = net.Dial("tcp", GameServerAddr)
	if s.err_2 != nil {
		c.Fatal("Connect Server Error ", s.err_2)
	}
	username = fmt.Sprintf("robot_%d", 2)
	password = "123456"

	retcode, userid, sessionkey = Register(c, &s.conn_2, username, password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, charid, _ = Login(c, &s.conn_2, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)

	s.charid_2 = charid
}

func (s *FriendSuite) TearDownSuite(c *C) {
	s.conn_1.Close()
	s.conn_2.Close()
}

func (s *FriendSuite) SetUpTest(c *C) {

}

func (s *FriendSuite) TearDownTest(c *C) {

}

func (s *FriendSuite) TestFriendAdd(c *C) {

	req := &clientmsg.Req_Friend_Operate{}
	rspMsg := &clientmsg.Rlt_Friend_Operate{}

	req.Action = clientmsg.FriendOperateActionType_FOAT_ADD_FRIEND
	req.OperateCharID = s.charid_1
	req.Message = "Hello"
	msgid, msgdata := SendAndRecv(c, &s.conn_2, clientmsg.MessageType_MT_REQ_FRIEND_OPERATE, req)
	msgid, msgdata = SendAndRecv(c, &s.conn_2, clientmsg.MessageType_MT_REQ_FRIEND_OPERATE, req)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_FRIEND_OPERATE)
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Friend_Operate Decode Error ", err)
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.Action, Equals, req.Action)

	req.Action = clientmsg.FriendOperateActionType_FOAT_ACCEPT
	req.OperateCharID = s.charid_2
	msgid, msgdata = SendAndRecv(c, &s.conn_1, clientmsg.MessageType_MT_REQ_FRIEND_OPERATE, req)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_FRIEND_OPERATE)
	err = proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Friend_Operate Decode Error ", err)
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.Action, Equals, req.Action)

	req.Action = clientmsg.FriendOperateActionType_FOAT_DEL_FRIEND
	req.OperateCharID = s.charid_2
	msgid, msgdata = SendAndRecv(c, &s.conn_1, clientmsg.MessageType_MT_REQ_FRIEND_OPERATE, req)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_FRIEND_OPERATE)
	err = proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Friend_Operate Decode Error ", err)
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.Action, Equals, req.Action)

}
