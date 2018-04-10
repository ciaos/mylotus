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

func TestServerTime(t *testing.T) { TestingT(t) }

type ServerTimeSuite struct {
	conn   net.Conn
	err    error
	charid uint32
}

var _ = Suite(&ServerTimeSuite{})

func (s *ServerTimeSuite) SetUpSuite(c *C) {

}

func (s *ServerTimeSuite) TearDownSuite(c *C) {
}

func (s *ServerTimeSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *ServerTimeSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ServerTimeSuite) TestServerTime(c *C) {

	reqMsg := &clientmsg.Req_ServerTime{
		Time: uint64(time.Now().Unix()),
	}
	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_SERVERTIME, reqMsg, clientmsg.MessageType_MT_RLT_SERVERTIME)
	rspMsg := &clientmsg.Rlt_ServerTime{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.Time, Not(Equals), 0)
}
