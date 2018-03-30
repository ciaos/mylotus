package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func TestMultiLogin(t *testing.T) { TestingT(t) }

type MultiLoginSuite struct {
	conn_1 net.Conn
	conn_2 net.Conn
	err    error
}

var _ = Suite(&MultiLoginSuite{})

func (s *MultiLoginSuite) SetUpSuite(c *C) {
	s.conn_1, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
	s.conn_2, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
}

func (s *MultiLoginSuite) TearDownSuite(c *C) {
	s.conn_1.Close()
	s.conn_2.Close()
}

func (s *MultiLoginSuite) SetUpTest(c *C) {

}

func (s *MultiLoginSuite) TearDownTest(c *C) {

}

func (s *MultiLoginSuite) TestMultiLogin(c *C) {
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	retcode, userid, sessionkey := Register(c, &s.conn_1, username, password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_OK)

	code, _, isnew := Login(c, &s.conn_1, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(isnew, Equals, true)

	code, _, isnew = Login(c, &s.conn_2, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_OK)
	c.Assert(isnew, Equals, true)
}
