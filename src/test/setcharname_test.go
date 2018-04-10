package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	. "gopkg.in/check.v1"
)

func TestSetCharName(t *testing.T) { TestingT(t) }

type SetCharNameSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&SetCharNameSuite{})

func (s *SetCharNameSuite) SetUpSuite(c *C) {

}

func (s *SetCharNameSuite) TearDownSuite(c *C) {

}

func (s *SetCharNameSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *SetCharNameSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *SetCharNameSuite) TestSetCharName(c *C) {

	reqMsg := &clientmsg.Req_SetCharName{
		CharName: "player_" + strconv.Itoa(int(s.charid)),
	}

	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_SETCHARNAME, reqMsg, clientmsg.MessageType_MT_RLT_SETCHARNAME)
	rspMsg := &clientmsg.Rlt_SetCharName{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_SetCharName Decode Error ", err)
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
}
