package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

func TestGMCommand_Echo(t *testing.T)    { TestingT(t) }
func TestGMCommand_AddCash(t *testing.T) { TestingT(t) }

type GMCommandSuite struct {
	conn   net.Conn
	err    error
	charid uint32
}

var _ = Suite(&GMCommandSuite{})

func (s *GMCommandSuite) SetUpSuite(c *C) {

}

func (s *GMCommandSuite) TearDownSuite(c *C) {

}

func (s *GMCommandSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	retcode, userid, sessionkey := Register(c, &s.conn, username, password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, charid, isnew := Login(c, &s.conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(isnew, Equals, true)

	s.charid = charid
}

func (s *GMCommandSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *GMCommandSuite) TestGMCommand_Echo(c *C) {

	req := &clientmsg.Req_GM_Command{
		Command: "echo aaa",
	}

	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_GM_COMMAND, req, clientmsg.MessageType_MT_RLT_GM_COMMAND)
	rspMsg := &clientmsg.Rlt_GM_Command{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_GM_Command Decode Error ", err)
	}
	c.Assert(rspMsg.Result, Equals, "aaa")
}

func (s *GMCommandSuite) TestGMCommand_AddCash(c *C) {

	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_CASH)
	rspMsg := &clientmsg.Rlt_Asset_Cash{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.GoldCoin, Not(Equals), 0)

	goldcoin := rspMsg.GoldCoin
	req := &clientmsg.Req_GM_Command{
		Command: fmt.Sprintf("addcash %v 1 100", s.charid),
	}

	msgdata = SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_GM_COMMAND, req, clientmsg.MessageType_MT_RLT_GM_COMMAND)
	rsp := &clientmsg.Rlt_GM_Command{}
	err := proto.Unmarshal(msgdata, rsp)
	if err != nil {
		c.Fatal("Rlt_GM_Command Decode Error ", err)
	}
	msgdata = RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_CASH)
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.GoldCoin, Equals, uint32(goldcoin+100))

}
