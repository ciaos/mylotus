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

func TestSearch(t *testing.T) { TestingT(t) }

type SearchSuite struct {
	conn   net.Conn
	err    error
	charid uint32
}

var _ = Suite(&SearchSuite{})

func (s *SearchSuite) SetUpSuite(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	retcode, userid, sessionkey := Register(c, &s.conn, username, password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, charid, isnew := Login(c, &s.conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(isnew, Equals, true)

	s.charid = charid
}

func (s *SearchSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *SearchSuite) SetUpTest(c *C) {

}

func (s *SearchSuite) TearDownTest(c *C) {

}

func (s *SearchSuite) TestSearch(c *C) {
	req := &clientmsg.Req_Friend_Operate{
		Action:     clientmsg.FriendOperateActionType_FOAT_SEARCH,
		SearchName: "robot",
	}

	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_FRIEND_OPERATE, req, clientmsg.MessageType_MT_RLT_FRIEND_OPERATE)
	rspMsg := &clientmsg.Rlt_Friend_Operate{}
	err := proto.Unmarshal(msgdata, rspMsg)
	if err != nil {
		c.Fatal("Rlt_Friend_Operate Decode Error ", err)
	}
	c.Assert(rspMsg.RetCode, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(rspMsg.Action, Equals, req.Action)
	c.Assert(len(rspMsg.SearchedCharIDs), Not(Equals), 0)
}
