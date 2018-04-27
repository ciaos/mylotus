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

func TestReConnectBS(t *testing.T) { TestingT(t) }

type ReConnectBSSuite struct {
	conn  net.Conn
	err   error
	bconn net.Conn

	charid    uint32
	battlekey []byte
	bsaddr    string
}

var _ = Suite(&ReConnectBSSuite{})

func (s *ReConnectBSSuite) SetUpSuite(c *C) {
}

func (s *ReConnectBSSuite) TearDownSuite(c *C) {
}

func (s *ReConnectBSSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)

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
			SkinID:   100001,
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
	s.battlekey = rspAddress.BattleKey
	s.bsaddr = rspAddress.BattleAddr

	reqMsg := &clientmsg.Req_ConnectBS{
		RoomID:    rspAddress.RoomID,
		BattleKey: rspAddress.BattleKey,
		CharID:    s.charid,
	}

	msgdata = SendAndRecvUtil(c, &s.bconn, clientmsg.MessageType_MT_REQ_CONNECTBS, reqMsg, clientmsg.MessageType_MT_RLT_CONNECTBS)

	rMsg := &clientmsg.Rlt_ConnectBS{}
	err = proto.Unmarshal(msgdata, rMsg)
	if err != nil {
		c.Fatal("Rlt_ConnectBS Decode Error ", err)
	}
	c.Assert(rMsg.RetCode, Equals, clientmsg.Type_BattleRetCode_BRC_OK)
	c.Assert(rMsg.MapID, Equals, int32(100))

	for _, member := range rMsg.Member {
		c.Assert(member.CharType, Equals, int32(1001))
		c.Assert(member.SkinID, Equals, int32(100001))
	}

	for _, member := range rMsg.Member {
		req := &clientmsg.Transfer_Loading_Progress{
			CharID:   member.CharID,
			Progress: 100,
		}
		Send(c, &s.bconn, clientmsg.MessageType_MT_TRANSFER_LOADING_PROGRESS, req)
	}
	msgdata = RecvUtil(c, &s.bconn, clientmsg.MessageType_MT_RLT_STARTBATTLE)
	rsp := &clientmsg.Rlt_StartBattle{}
	err = proto.Unmarshal(msgdata, rsp)
	if err != nil {
		c.Fatal("Rlt_StartBattle Decode Error ", err)
	}
	c.Assert(rsp.RandSeed, Not(Equals), 0)
}

func (s *ReConnectBSSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ReConnectBSSuite) TestReConnectBS(c *C) {
	s.bconn.Close()
	time.Sleep(time.Duration(3) * time.Second)

	s.bconn, s.err = kcp.Dial(s.bsaddr)
	if s.err != nil {
		c.Fatal("Connect BattleServer Error ", s.err)
	}

	req := &clientmsg.Req_Re_ConnectBS{
		BattleKey: s.battlekey,
		CharID:    s.charid,
		FrameID:   0,
	}

	msgdata := SendAndRecvUtil(c, &s.bconn, clientmsg.MessageType_MT_REQ_RE_CONNECTBS, req, clientmsg.MessageType_MT_RLT_CONNECTBS)
	rsp := &clientmsg.Rlt_ConnectBS{}
	err := proto.Unmarshal(msgdata, rsp)
	if err != nil {
		c.Fatal("Rlt_ConnectBS Decode Error ", err)
	}
	c.Assert(rsp.RetCode, Equals, clientmsg.Type_BattleRetCode_BRC_OK)
	c.Assert(rsp.IsReconnect, Equals, true)

	msgdata = RecvUtil(c, &s.bconn, clientmsg.MessageType_MT_TRANSFER_COMMAND)
	cmd := &clientmsg.Transfer_Command{}
	err = proto.Unmarshal(msgdata, cmd)
	if err != nil {
		c.Fatal("Transfer_Command Decode Error ", err)
	}
	c.Assert(cmd.FrameID, Equals, uint32(1))
}
