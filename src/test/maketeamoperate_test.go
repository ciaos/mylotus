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

func TestMakeTeamOperate(t *testing.T) { TestingT(t) }

var (
	num = 2
)

type MakeTeamOperateSuite struct {
	conn   []net.Conn
	err    []error
	charid []uint32
}

var _ = Suite(&MakeTeamOperateSuite{})

func (s *MakeTeamOperateSuite) SetUpSuite(c *C) {
	s.conn = make([]net.Conn, num, num)
	s.err = make([]error, num, num)
	s.charid = make([]uint32, num, num)
}

func (s *MakeTeamOperateSuite) TearDownSuite(c *C) {
}

func (s *MakeTeamOperateSuite) SetUpTest(c *C) {
	rand.Seed(time.Now().Unix())
	for i := 0; i < len(s.conn); i++ {
		s.conn[i], s.err[i] = net.Dial("tcp", GameServerAddr)
		if s.err[i] != nil {
			c.Fatal("Connect Server Error ", s.err[i])
		}
		username := fmt.Sprintf("robot1_%d", rand.Intn(10000))
		password := "123456"

		retcode, userid, sessionkey := Register(c, &s.conn[i], username, password, false)
		c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

		code, charid, _ := Login(c, &s.conn[i], userid, sessionkey)
		c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)

		s.charid[i] = charid
	}
}

func (s *MakeTeamOperateSuite) TearDownTest(c *C) {
	for i := 0; i < len(s.conn); i++ {
		s.conn[i].Close()
	}
}

func (s *MakeTeamOperateSuite) TestMakeTeamOperate(c *C) {
	reqMsg := &clientmsg.Req_MakeTeamOperate{
		Action: clientmsg.MakeTeamOperateType_MTOT_CREATE,
		Mode:   clientmsg.MatchModeType_MMT_RANK,
		MapID:  10001,
	}

	msgdata := SendAndRecvUtil(c, &s.conn[0], clientmsg.MessageType_MT_REQ_MAKETEAM_OPERATE, reqMsg, clientmsg.MessageType_MT_RLT_MAKETEAM_OPERATE)
	rspMsg := &clientmsg.Rlt_MakeTeamOperate{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)

	time.Sleep(time.Duration(5) * time.Second)

	reqMsg.Action = clientmsg.MakeTeamOperateType_MTOT_INVITE
	reqMsg.TargetID = s.charid[1]
	Send(c, &s.conn[0], clientmsg.MessageType_MT_REQ_MAKETEAM_OPERATE, reqMsg)

	msgdata = RecvUtil(c, &s.conn[1], clientmsg.MessageType_MT_RLT_MAKETEAM_OPERATE)
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.TargetID, Equals, s.charid[1])
	c.Assert(rspMsg.InviterID, Equals, s.charid[0])

	time.Sleep(time.Duration(5) * time.Second)

	reqMsg.Action = clientmsg.MakeTeamOperateType_MTOT_ACCEPT
	reqMsg.BenchID = rspMsg.BenchID
	reqMsg.MatchServerID = rspMsg.MatchServerID
	msgdata = SendAndRecvUtil(c, &s.conn[1], clientmsg.MessageType_MT_REQ_MAKETEAM_OPERATE, reqMsg, clientmsg.MessageType_MT_RLT_MAKETEAM_OPERATE)
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_BENCH_INFO)

	msgdata = RecvUtil(c, &s.conn[0], clientmsg.MessageType_MT_RLT_MAKETEAM_OPERATE)
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_BENCH_INFO)

	time.Sleep(time.Duration(5) * time.Second)

	reqMsg.Action = clientmsg.MakeTeamOperateType_MTOT_START_MATCH
	msgdata = SendAndRecvUtil(c, &s.conn[0], clientmsg.MessageType_MT_REQ_MAKETEAM_OPERATE, reqMsg, clientmsg.MessageType_MT_RLT_MAKETEAM_OPERATE)
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.Action, Equals, clientmsg.MakeTeamOperateType_MTOT_START_MATCH)

	msgdata = RecvUtil(c, &s.conn[1], clientmsg.MessageType_MT_RLT_MAKETEAM_OPERATE)
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.Action, Equals, clientmsg.MakeTeamOperateType_MTOT_START_MATCH)

	time.Sleep(time.Duration(15) * time.Second)
}
