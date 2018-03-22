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
}

var _ = Suite(&SetCharNameSuite{})

func (s *SetCharNameSuite) SetUpSuite(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
}

func (s *SetCharNameSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *SetCharNameSuite) SetUpTest(c *C) {

}

func (s *SetCharNameSuite) TearDownTest(c *C) {

}

func (s *SetCharNameSuite) TestSetCharName(c *C) {
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	retcode, userid, sessionkey := Register(c, &s.conn, username, password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, charid, isnew := Login(c, &s.conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(isnew, Equals, true)

	reqMsg := &clientmsg.Req_SetCharName{
		CharName: "player_" + strconv.Itoa(int(charid)),
	}

	msgid, msgdata := SendAndRecv(c, &s.conn, clientmsg.MessageType_MT_REQ_SETCHARNAME, reqMsg)
	c.Assert(msgid, Equals, clientmsg.MessageType_MT_RLT_SETCHARNAME)
	rspMsg := &clientmsg.Rlt_SetCharName{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_SetCharName Decode Error ", err)
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
}
