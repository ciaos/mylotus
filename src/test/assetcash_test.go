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

func TestAssetCash(t *testing.T) { TestingT(t) }

type AssetCashSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&AssetCashSuite{})

func (s *AssetCashSuite) SetUpSuite(c *C) {

}

func (s *AssetCashSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *AssetCashSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *AssetCashSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *AssetCashSuite) TestAssetCash(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_CASH)
	rspMsg := &clientmsg.Rlt_Asset_Cash{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.GoldCoin, Not(Equals), 0)
}
