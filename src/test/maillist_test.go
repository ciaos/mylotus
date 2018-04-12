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

func TestMailList(t *testing.T) { TestingT(t) }

type MailListSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&MailListSuite{})

func (s *MailListSuite) SetUpSuite(c *C) {
}

func (s *MailListSuite) TearDownSuite(c *C) {
}

func (s *MailListSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *MailListSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *MailListSuite) TestPing(c *C) {

	rand.Seed(time.Now().UnixNano())
	reqMsg := &clientmsg.Req_Mail_Action{
		Action: clientmsg.MailActionType_MAT_LIST_MAIL,
	}

	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_MAIL_ACTION, reqMsg, clientmsg.MessageType_MT_RLT_MAIL_ACTION)
	rspMsg := &clientmsg.Rlt_Mail_Action{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.Action, Equals, reqMsg.Action)
}
