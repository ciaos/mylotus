package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	"github.com/ciaos/leaf/kcp"
	"github.com/golang/protobuf/proto"
	. "gopkg.in/check.v1"
)

func TestConnectBS(t *testing.T) { TestingT(t) }

type ConnectBSSuite struct {
	conn  net.Conn
	err   error
	bconn *kcp.UDPSession

	charid uint32
}

var _ = Suite(&ConnectBSSuite{})

func (s *ConnectBSSuite) SetUpSuite(c *C) {
}

func (s *ConnectBSSuite) TearDownSuite(c *C) {
}

func (s *ConnectBSSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *ConnectBSSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ConnectBSSuite) TestConnectBS(c *C) {
	msgid, msgdata := QuickMatch(c, &s.conn)
	rspMatch := &clientmsg.Rlt_Match{}
	err := proto.Unmarshal(msgdata, rspMatch)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMatch.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_OK)

	reqMatch := &clientmsg.Req_Match{
		Action: clientmsg.MatchActionType_MAT_CONFIRM,
		Mode:   clientmsg.MatchModeType_MMT_NORMAL,
	}
	msgid, msgdata = SendAndRecv(c, &s.conn, clientmsg.MessageType_MT_REQ_MATCH, reqMatch)
	err = proto.Unmarshal(msgdata, rspMatch)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMatch.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_CONFIRM)

	msgid, msgdata = Recv(c, &s.conn)
	err = proto.Unmarshal(msgdata, rspMatch)
	if err != nil {
		c.Fatal("Rlt_Match Decode Error")
	}
	c.Assert(rspMatch.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_MATCH_ALL_CONFIRMED)

	operateMsg := &clientmsg.Transfer_Team_Operate{
		Action: clientmsg.TeamOperateActionType_TOA_SETTLE,
		CharID: s.charid,
	}
	msgid, msgdata = SendAndRecv(c, &s.conn, clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE, operateMsg)

	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_NOTIFYBATTLEADDRESS)
	rspAddress := &clientmsg.Rlt_NotifyBattleAddress{}
	err = proto.Unmarshal(msgdata, rspAddress)
	if err != nil {
		c.Fatal("Rlt_NotifyBattleAddress Decode Error")
	}

	s.bconn, s.err = kcp.Dial(kcp.MODE_FAST, rspAddress.BattleAddr)
	if s.err != nil {
		c.Fatal("Connect BattleServer Error ", s.err)
	}

	reqMsg := &clientmsg.Req_ConnectBS{
		RoomID:    rspAddress.RoomID,
		BattleKey: rspAddress.BattleKey,
		CharID:    s.charid,
	}

	msgid, msgdata = SendAndRecvKCP(c, s.bconn, clientmsg.MessageType_MT_REQ_CONNECTBS, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_CONNECTBS)

	rMsg := &clientmsg.Rlt_ConnectBS{}
	err = proto.Unmarshal(msgdata, rMsg)
	if err != nil {
		c.Fatal("Rlt_ConnectBS Decode Error ", err)
	}
	c.Assert(rMsg.RetCode, Equals, clientmsg.Type_BattleRetCode_BRC_OK)
	s.bconn.Close()
}
