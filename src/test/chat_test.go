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

func TestChat(t *testing.T) { TestingT(t) }

type ChatSuite struct {
	conn   net.Conn
	err    error
	charid uint32
}

var _ = Suite(&ChatSuite{})

func (s *ChatSuite) SetUpSuite(c *C) {
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

func (s *ChatSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *ChatSuite) SetUpTest(c *C) {

}

func (s *ChatSuite) TearDownTest(c *C) {

}

func (s *ChatSuite) TestChat(c *C) {
	req := &clientmsg.Req_Chat{}
	req.Channel = clientmsg.ChatChannelType_CCT_WORLD
	req.MessageType = uint32(0)
	req.TargetID = 0
	req.MessageData = "Hello"

	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_CHAT, req, clientmsg.MessageType_MT_RLT_CHAT)
	rspMsg := &clientmsg.Rlt_Chat{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Chat Decode Error ", err)
	}
	c.Assert(rspMsg.Channel, Equals, clientmsg.ChatChannelType_CCT_WORLD)
	c.Assert(rspMsg.SenderID, Equals, s.charid)
	c.Assert(rspMsg.MessageData, Equals, "Hello")
}
