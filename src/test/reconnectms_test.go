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

func TestConnectMS(t *testing.T) { TestingT(t) }

type ConnectMSSuite struct {
	conn net.Conn
	err  error

	username string
	password string

	charid     uint32
	sessionkey []byte
}

var _ = Suite(&ConnectMSSuite{})

func (s *ConnectMSSuite) SetUpSuite(c *C) {
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

func (s *ConnectMSSuite) TearDownSuite(c *C) {
}

func (s *ConnectMSSuite) SetUpTest(c *C) {
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

func (s *ConnectMSSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ConnectMSSuite) TestConnectMS(c *C) {
	msgdata := QuickMatch(c, &s.conn)
	rspMatch := &clientmsg.Rlt_Match{}
	err := proto.Unmarshal(msgdata, rspMatch)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}

	c.Assert(rspMatch.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_OK)

	reqMatch := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType_MAT_CONFIRM,
		Mode:   clientmsg.MatchModeType_MMT_AI,
	}
	msgdata = SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_MATCH, reqMatch, clientmsg.MessageType_MT_RLT_MATCH)
	err = proto.Unmarshal(msgdata, rspMatch)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMatch.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_ALL_CONFIRMED)

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
	err = proto.Unmarshal(msgdata, rspMatch)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMatch.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_RECONNECT_OK)
}
