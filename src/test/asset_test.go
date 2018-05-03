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

func TestAssetCash(t *testing.T)        { TestingT(t) }
func TestAssetHero(t *testing.T)        { TestingT(t) }
func TestAssetMail(t *testing.T)        { TestingT(t) }
func TestAssetItem(t *testing.T)        { TestingT(t) }
func TestAssetFriend(t *testing.T)      { TestingT(t) }
func TestAssetStatistic(t *testing.T)   { TestingT(t) }
func TestAssetAchievement(t *testing.T) { TestingT(t) }
func TestAssetTask(t *testing.T)        { TestingT(t) }
func TestAssetTutorial(t *testing.T)    { TestingT(t) }

type AssetSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&AssetSuite{})

func (s *AssetSuite) SetUpSuite(c *C) {

}

func (s *AssetSuite) TearDownSuite(c *C) {
	s.conn.Close()
}

func (s *AssetSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *AssetSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *AssetSuite) TestAssetCash(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_CASH)
	rspMsg := &clientmsg.Rlt_Asset_Cash{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.GoldCoin, Not(Equals), 0)
}

func (s *AssetSuite) TestAssetHero(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_HERO)
	rspMsg := &clientmsg.Rlt_Asset_Hero{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(len(rspMsg.Roles), Not(Equals), 0)
}

func (s *AssetSuite) TestAssetMail(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_MAIL)
	rspMsg := &clientmsg.Rlt_Asset_Mail{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(len(rspMsg.MailData), Equals, 1)
}

func (s *AssetSuite) TestAssetItem(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_ITEM)
	rspMsg := &clientmsg.Rlt_Asset_Item{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.CharID, Equals, s.charid)
}

func (s *AssetSuite) TestAssetFriend(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_FRIEND)
	rspMsg := &clientmsg.Rlt_Asset_Friend{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.CharID, Equals, s.charid)
}

func (s *AssetSuite) TestAssetStatistic(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_STATISTIC)
	rspMsg := &clientmsg.Rlt_Asset_Statistic{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.CharID, Equals, s.charid)
}

func (s *AssetSuite) TestAssetAchievement(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_ACHIEVEMENT)
	rspMsg := &clientmsg.Rlt_Asset_Achievement{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.CharID, Equals, s.charid)
}

func (s *AssetSuite) TestAssetTask(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_TASK)
	rspMsg := &clientmsg.Rlt_Asset_Task{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.CharID, Equals, s.charid)
}

func (s *AssetSuite) TestAssetTutorial(c *C) {
	msgdata := RecvUtil(c, &s.conn, clientmsg.MessageType_MT_RLT_ASSET_TUTORIAL)
	rspMsg := &clientmsg.Rlt_Asset_Tutorial{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.CharID, Equals, s.charid)
}
