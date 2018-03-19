package test

import (
	"net"
	"server/msg/clientmsg"
	"testing"

	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

func TestServerList(t *testing.T) { TestingT(t) }

type ServerListSuite struct {
	conn net.Conn
	err  error
}

var _ = Suite(&ServerListSuite{})

func (s *ServerListSuite) SetUpSuite(c *C) {
}

func (s *ServerListSuite) TearDownSuite(c *C) {
}

func (s *ServerListSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
}

func (s *ServerListSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ServerListSuite) TestServerList(c *C) {

	reqMsg := &clientmsg.Req_ServerList{
		Channel: 0,
	}

	msgid, msgdata := SendAndRecv(c, &s.conn, clientmsg.MessageType_MT_REQ_SERVERLIST, reqMsg)

	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_SERVERLIST)
	rspMsg := &clientmsg.Rlt_ServerList{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.ServerCount, Not(Equals), 0)
	c.Assert(len(rspMsg.GetServerList()), Not(Equals), 0)
}
