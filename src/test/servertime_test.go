package test

import (
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

func TestServerTime(t *testing.T) { TestingT(t) }

type ServerTimeSuite struct {
	conn net.Conn
	err  error
}

var _ = Suite(&ServerTimeSuite{})

func (s *ServerTimeSuite) SetUpSuite(c *C) {
}

func (s *ServerTimeSuite) TearDownSuite(c *C) {
}

func (s *ServerTimeSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
}

func (s *ServerTimeSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ServerTimeSuite) TestServerTime(c *C) {

	reqMsg := &clientmsg.Req_ServerTime{
		Time: uint32(time.Now().Unix()),
	}
	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_SERVERTIME, reqMsg, clientmsg.MessageType_MT_RLT_SERVERTIME)
	rspMsg := &clientmsg.Rlt_ServerTime{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.Time, Not(Equals), 0)
}
