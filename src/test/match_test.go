package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func TestMatch(t *testing.T) { TestingT(t) }

type MatchSuite struct {
	conn net.Conn
	err  error

	username string
	password string
}

var _ = Suite(&MatchSuite{})

func (s *MatchSuite) SetUpSuite(c *C) {
	//register first
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	s.username = fmt.Sprintf("pengjing%d", rand.Intn(10000))
	s.password = "123456"

	retcode, _, _ := Register(c, &s.conn, s.username, s.password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_NONE)

	defer s.conn.Close()
}

func (s *MatchSuite) TearDownSuite(c *C) {
}

func (s *MatchSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
	retcode, userid, sessionkey := Register(c, &s.conn, s.username, s.password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_NONE)
	s.conn.Close()

	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
	code, _, isnew := Login(c, &s.conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_NONE)
	c.Assert(isnew, Equals, true)
}

func (s *MatchSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *MatchSuite) TestMatch(c *C) {
	reqMsg := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType_MAT_JOIN,
		Mode:   clientmsg.MatchModeType_MMT_NORMAL,
	}

	msgid, _ := SendAndRecv(c, &s.conn, clientmsg.MessageType_MT_REQ_MATCH, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_MATCH)
}
