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

func TestHeartBeat(t *testing.T) { TestingT(t) }

type HeartBeatSuite struct {
	conn  net.Conn
	err   error
	bconn net.Conn

	charid uint32
}

var _ = Suite(&HeartBeatSuite{})

func (s *HeartBeatSuite) SetUpSuite(c *C) {
}

func (s *HeartBeatSuite) TearDownSuite(c *C) {
}

func (s *HeartBeatSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *HeartBeatSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *HeartBeatSuite) TestConnectBS(c *C) {
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

	for _, member := range rspMatch.Members {
		operateMsg := &clientmsg.Transfer_Team_Operate{
			Action:   clientmsg.TeamOperateActionType_TOA_SETTLE,
			CharID:   member.CharID,
			CharType: 1001,
		}
		msgdata = SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE, operateMsg, clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE)
		err = proto.Unmarshal(msgdata, operateMsg)
		if err != nil {
			c.Fatal("Transfer_Team_Operate Decode Error")
		}
		c.Assert(operateMsg.CharType, Equals, int32(1001))
	}
	msgdata = RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_NOTIFYBATTLEADDRESS)
	rspAddress := &clientmsg.Rlt_NotifyBattleAddress{}
	err = proto.Unmarshal(msgdata, rspAddress)
	if err != nil {
		c.Fatal("Rlt_NotifyBattleAddress Decode Error")
	}

	s.bconn, s.err = kcp.Dial(rspAddress.BattleAddr)
	if s.err != nil {
		c.Fatal("Connect BattleServer Error ", s.err)
	}

	reqMsg := &clientmsg.Transfer_Battle_Heartbeat{
		TickTime: uint64(time.Now().Unix()),
	}

	Send(c, &s.bconn, clientmsg.MessageType_MT_REQ_CONNECTBS, reqMsg)
}
