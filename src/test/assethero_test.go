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

func TestAssetHero(t *testing.T) { TestingT(t) }

type AssetHeroSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&AssetHeroSuite{})

func (s *AssetHeroSuite) SetUpSuite(c *C) {

}

func (s *AssetHeroSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *AssetHeroSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *AssetHeroSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *AssetHeroSuite) TestAssetHero(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_HERO)
	rspMsg := &clientmsg.Rlt_Asset_Hero{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(len(rspMsg.Roles), Not(Equals), 0)
}
