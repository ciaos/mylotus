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

func TestConnectBS(t *testing.T) { TestingT(t) }

type ConnectBSSuite struct {
	conn net.Conn
	err  error

	charid string
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
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_NOTIFYBATTLEADDRESS)
	rspMsg := &clientmsg.Rlt_NotifyBattleAddress{}
	err := proto.Unmarshal(msgdata, rspMsg)

	s.conn.Close()
	s.conn, s.err = net.Dial("tcp", rspMsg.GetBattleAddr())
	if s.err != nil {
		c.Fatal("Connect BattleServer Error ", s.err)
	}

	reqMsg := &clientmsg.Req_ConnectBS{
		RoomID:    proto.Int32(rspMsg.GetRoomID()),
		BattleKey: rspMsg.GetBattleKey(),
		CharID:    proto.String(s.charid),
	}
	msgid, msgdata = SendAndRecv(c, &s.conn, clientmsg.MessageType_MT_REQ_CONNECTBS, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_CONNECTBS)

	rMsg := &clientmsg.Rlt_ConnectBS{}
	err = proto.Unmarshal(msgdata, rMsg)
	if err != nil {
		c.Fatal("Rlt_ConnectBS Decode Error ", err)
	}
	c.Assert(rMsg.GetRetCode(), Equals, clientmsg.Type_BattleRetCode_BRC_NONE)
}
