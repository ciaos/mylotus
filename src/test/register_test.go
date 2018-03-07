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

func TestRegister(t *testing.T) { TestingT(t) }

type RegisterSuite struct {
	conn     net.Conn
	err      error
	username string
	password string
	login    bool
}

var _ = Suite(&RegisterSuite{})

func (s *RegisterSuite) SetUpSuite(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	s.username = fmt.Sprintf("pengjing%d", rand.Intn(10000))
	s.password = "123456"
}

func (s *RegisterSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *RegisterSuite) SetUpTest(c *C) {

}

func (s *RegisterSuite) TearDownTest(c *C) {

}

func (s *RegisterSuite) TestRegister(c *C) {
	retcode, _, _ := Register(c, &s.conn, s.username, s.password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST)

	retcode, _, _ = Register(c, &s.conn, s.username, s.password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_NONE)

	retcode, _, _ = Register(c, &s.conn, s.username, s.password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_ACCOUNT_EXIST)

	retcode, _, _ = Register(c, &s.conn, s.username, s.password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_NONE)

	s.password = "123"
	retcode, _, _ = Register(c, &s.conn, s.username, s.password, true)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_PASSWORD_ERROR)
}
