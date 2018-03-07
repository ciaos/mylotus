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

func TestLogin(t *testing.T) { TestingT(t) }

type LoginSuite struct {
	conn net.Conn
	err  error
}

var _ = Suite(&LoginSuite{})

func (s *LoginSuite) SetUpSuite(c *C) {
	s.conn, s.err = net.Dial("tcp", GameServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}
}

func (s *LoginSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *LoginSuite) SetUpTest(c *C) {

}

func (s *LoginSuite) TearDownTest(c *C) {

}

func (s *LoginSuite) TestLogin(c *C) {
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	retcode, userid, sessionkey := Register(c, &s.conn, username, password, false)
	c.Assert(retcode, Equals, clientmsg.Type_LoginRetCode_LRC_NONE)

	code, _, isnew := Login(c, &s.conn, userid, sessionkey)
	c.Assert(code, Equals, clientmsg.Type_GameRetCode_GRC_NONE)
	c.Assert(isnew, Equals, true)
}
