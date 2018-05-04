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

func TestShopList(t *testing.T) { TestingT(t) }
func TestShopBuy(t *testing.T)  { TestingT(t) }

type ShopSuite struct {
	conn net.Conn
	err  error

	charid uint32
}

var _ = Suite(&ShopSuite{})

func (s *ShopSuite) SetUpSuite(c *C) {
}

func (s *ShopSuite) TearDownSuite(c *C) {
}

func (s *ShopSuite) SetUpTest(c *C) {
	s.conn, s.err = net.Dial("tcp", LoginServerAddr)
	if s.err != nil {
		c.Fatal("Connect Server Error ", s.err)
	}

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	s.charid = QuickLogin(c, &s.conn, username, password)
}

func (s *ShopSuite) TearDownTest(c *C) {
	s.conn.Close()
}

func (s *ShopSuite) TestShopList(c *C) {
	reqMsg := &clientmsg.Req_Shop_List{
		Category: clientmsg.Type_Category_TC_HERO,
	}
	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_SHOP_LIST, reqMsg, clientmsg.MessageType_MT_RLT_SHOP_LIST)
	rspMsg := &clientmsg.Rlt_Shop_List{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.Category, Equals, reqMsg.Category)
	c.Assert(len(rspMsg.Goods), Not(Equals), 0)
}

func (s *ShopSuite) TestShopBuy(c *C) {

	reqMsg := &clientmsg.Req_Shop_List{
		Category: clientmsg.Type_Category_TC_HERO,
	}
	msgdata := SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_SHOP_LIST, reqMsg, clientmsg.MessageType_MT_RLT_SHOP_LIST)
	rspMsg := &clientmsg.Rlt_Shop_List{}
	proto.Unmarshal(msgdata, rspMsg)
	c.Assert(rspMsg.Category, Equals, reqMsg.Category)
	c.Assert(len(rspMsg.Goods), Not(Equals), 0)

	time.Sleep(time.Duration(3) * time.Second)

	req := &clientmsg.Req_Shop_Buy{
		ItemID:   rspMsg.Goods[0].ItemID,
		CashType: rspMsg.Goods[0].BuyList[0].CashType,
	}

	msgdata = SendAndRecvUtil(c, &s.conn, clientmsg.MessageType_MT_REQ_SHOP_BUY, req, clientmsg.MessageType_MT_RLT_SHOP_BUY)
	rsp := &clientmsg.Rlt_Shop_Buy{}
	proto.Unmarshal(msgdata, rsp)
	c.Assert(rsp.RetCode, Equals, clientmsg.Type_BuyRetCode_TBR_SYSTEM_ERR)
}
