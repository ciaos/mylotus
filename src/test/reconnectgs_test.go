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

func TestReconnectGS_Normal(t *testing.T) { TestingT(t) }
func TestReconnectGS_Match(t *testing.T)  { TestingT(t) }

type ReconnectGSSuite struct {
	conn net.Conn
	err  error

	username string
	password string

	charid     uint32
	sessionkey []byte
}

var _ = Suite(&ReconnectGSSuite{})

func (s *ReconnectGSSuite) SetUpSuite(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	s.username = fmt.Sprintf("pengjing%d", rand.Intn(10000))
	s.password = "123456"

	retcode, _, _ := Register(c, &s.conn, s.username, s.password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)
	s.conn.Close()
}

func (s *ReconnectGSSuite) TearDownSuite(c *C) {
}

func (s *ReconnectGSSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
	retcode, userid, sessionkey := Register(c, &s.conn, s.username, s.password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	s.sessionkey = sessionkey
	code, charid, isnew := Login(c, &s.conn, userid, s.sessionkey)
	s.charid = charid
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(isnew, Equals, true)
}

func (s *ReconnectGSSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ReconnectGSSuite) TestReconnectGS_Normal(c *C) {
	s.conn.Close()

	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	req := &clientmsg.Req_Re_ConnectGS{
		CharID:     s.charid,
		SessionKey: s.sessionkey,
	}
	rlt := &clientmsg.Rlt_Re_ConnectGS{}
	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_RE_CONNECTGS, req, clientmsg.MessageType_MT_RLT_RE_CONNECTGS)
	err := proto.Unmarshal(msgdata, rlt)
	if err != nil {
		c.Fatal("Rlt_Re_ConnectGS Decode Error")
	}
	c.Assert(rlt.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)

	reqMsg := &clientmsg.Ping{
		ID: uint32(rand.Intn(10000)),
	}

	msgdata = SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_PING, reqMsg, clientmsg.MessageType_MT_PONG)
	rspMsg := &clientmsg.Pong{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.ID, Equals, reqMsg.ID)

}

func (s *ReconnectGSSuite) TestReconnectGS_Match(c *C) {
	reqMsg := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType_MAT_JOIN,
		Mode:   clientmsg.MatchModeType_MMT_AI,
	}

	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_MATCH, reqMsg, clientmsg.MessageType_MT_RLT_MATCH)
	rspMsg := &clientmsg.Rlt_Match{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_START)

	time.Sleep(time.Duration(5) * time.Second)

	s.conn.Close()

	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	req := &clientmsg.Req_Re_ConnectGS{
		CharID:     s.charid,
		SessionKey: s.sessionkey,
	}
	rlt := &clientmsg.Rlt_Re_ConnectGS{}
	msgdata = SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_RE_CONNECTGS, req, clientmsg.MessageType_MT_RLT_RE_CONNECTGS)
	err = proto.Unmarshal(msgdata, rlt)
	if err != nil {
		c.Fatal("Rlt_Re_ConnectGS Decode Error")
	}
	c.Assert(rlt.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)

	msgdata = RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_MATCH)
	err = proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_RECONNECT_OK)
}
