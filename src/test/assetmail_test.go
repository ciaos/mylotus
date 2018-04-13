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

func TestAssetMail(t *testing.T) { TestingT(t) }

type AssetMailSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&AssetMailSuite{})

func (s *AssetMailSuite) SetUpSuite(c *C) {

}

func (s *AssetMailSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *AssetMailSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *AssetMailSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *AssetMailSuite) TestAssetMail(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_MAIL)
	rspMsg := &clientmsg.Rlt_Asset_Mail{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(len(rspMsg.MailData), Equals, 1)
}
