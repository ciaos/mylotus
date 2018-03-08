package main

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"server/msg/clientmsg"

	"github.com/golang/protobuf/proto"
)

const (
	CLIENT_NUM = 2

	STATUS_NONE = "STATUS_NONE"

	STATUS_LOGIN_CONNECT = "STATUS_LOGIN_CONNECT"
	STATUS_LOGIN         = "STATUS_LOGIN"
	STATUS_LOGIN_LOOP    = "STATUS_LOGIN_LOOP"
	STATUS_LOGIN_CLOSE   = "STATUS_LOGIN_CLOSE"

	STATUS_GAME_CONNECT = "STATUS_GAME_CONNECT"
	STATUS_GAME_LOGIN   = "STATUS_GAME_LOGIN"
	STATUS_GAME_MATCH   = "STATUS_GAME_MATCH"
	STATUS_GAME_LOOP    = "STATUS_GAME_LOOP"
	STATUS_GAME_CLOSE   = "STATUS_GAME_CLOSE"

	STATUS_BATTLE_CONNECT = "STATUS_BATTLE_CONNECT"
	STATUS_BATTLE         = "STATUS_BATTLE"
	STATUS_BATTLE_LOOP    = "STATUS_BATTLE_LOOP"
	STATUS_BATTLE_CLOSE   = "STATUS_BATTLE_CLOSE"
)

var w sync.WaitGroup
var m *sync.Mutex

type Client struct {
	id       int32
	username string
	password string

	sessionkey   []byte
	battlekey    []byte
	battleaddr   string
	battleroomid int32

	userid string
	charid string
	status string

	lconn net.Conn
	gconn net.Conn
	bconn net.Conn

	err error

	nextlogintime int64

	nextpinggstime int64
	nextpingbstime int64

	startbattletime int64
	maxbattletime   int64

	routes map[interface{}]interface{}
}

func (c *Client) ChangeStatus(status string) {
	c.status = status
	fmt.Printf("client %d change to status %s\n", c.id, c.status)
}

func handle_Pong(c *Client, msgdata []byte) {
	rsp := &clientmsg.Pong{}
	proto.Unmarshal(msgdata, rsp)
	fmt.Printf("client %d recv pong %d\n", c.id, rsp.GetID())
}

func handle_Rlt_Register(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_Register{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.GetRetCode() == clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST {
		msg := &clientmsg.Req_Register{
			UserName:      proto.String(c.username),
			Password:      proto.String(c.password),
			IsLogin:       proto.Bool(false),
			ClientVersion: proto.Int32(0),
		}
		go Send(&c.lconn, clientmsg.MessageType_MT_REQ_REGISTER, msg)
	} else if rsp.GetRetCode() == clientmsg.Type_LoginRetCode_LRC_NONE {
		c.userid = rsp.GetUserID()
		c.sessionkey = rsp.GetSessionKey()
		c.ChangeStatus(STATUS_LOGIN_CLOSE)
	} else {
		c.nextlogintime = time.Now().Unix() + 5
		c.ChangeStatus(STATUS_NONE)
		c.lconn.Close()
	}
}

func handle_Rlt_Login(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_Login{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.GetRetCode() == clientmsg.Type_GameRetCode_GRC_NONE {
		c.charid = rsp.GetCharID()
		msg := &clientmsg.Req_Match{
			Action: clientmsg.MatchActionType.Enum(clientmsg.MatchActionType_MAT_JOIN),
			Mode:   clientmsg.MatchModeType.Enum(clientmsg.MatchModeType_MMT_NORMAL),
		}
		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_MATCH, msg)
	} else {
		c.nextlogintime = time.Now().Unix() + 5
		c.ChangeStatus(STATUS_NONE)
		c.gconn.Close()
	}
}

func handle_Rlt_NotifyBattleAddress(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_NotifyBattleAddress{}
	proto.Unmarshal(msgdata, rsp)
	c.battlekey = rsp.GetBattleKey()
	c.battleaddr = rsp.GetBattleAddr()
	c.battleroomid = rsp.GetRoomID()
	c.ChangeStatus(STATUS_BATTLE_CONNECT)
}

func handle_Rlt_ConnectBS(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_ConnectBS{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.GetRetCode() != clientmsg.Type_BattleRetCode_BRC_NONE {
		c.ChangeStatus(STATUS_BATTLE_CLOSE)
	}
}

func handle_Rlt_EndBattle(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_EndBattle{}
	proto.Unmarshal(msgdata, rsp)

	c.ChangeStatus(STATUS_BATTLE_CLOSE)
}

func handle_Transfer_Command(c *Client, msgdata []byte) {
	rsp := &clientmsg.Transfer_Command{}
	proto.Unmarshal(msgdata, rsp)

	fmt.Printf("client %d CharID %s recv transfer command from %s\n", c.id, c.charid, rsp.GetCharID())
}

func (c *Client) updateLogin() {
	if c.status == STATUS_LOGIN_CONNECT {
		c.lconn, c.err = net.Dial("tcp", LoginServerAddr)
		if c.err != nil {
			c.nextlogintime = time.Now().Unix() + 5
			c.ChangeStatus(STATUS_NONE)
		} else {
			c.ChangeStatus(STATUS_LOGIN)
		}
	} else if c.status == STATUS_LOGIN {
		msg := &clientmsg.Req_Register{
			UserName:      proto.String(c.username),
			Password:      proto.String(c.password),
			IsLogin:       proto.Bool(true),
			ClientVersion: proto.Int32(0),
		}
		go Send(&c.lconn, clientmsg.MessageType_MT_REQ_REGISTER, msg)
		c.ChangeStatus(STATUS_LOGIN_LOOP)
	} else if c.status == STATUS_LOGIN_CLOSE {
		c.lconn.Close()
		c.ChangeStatus(STATUS_GAME_CONNECT)
	}
}

func (c *Client) recvLogin() {
	for {
		if c.status == STATUS_LOGIN_LOOP {
			err, msgid, msgbuf := Recv(&c.lconn)
			if err != nil {
				continue
			}
			c.dispatch(msgid, msgbuf)
		}

		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func (c *Client) updateGame() {
	if c.status == STATUS_GAME_CONNECT {
		c.gconn, c.err = net.Dial("tcp", GameServerAddr)
		if c.err != nil {
			c.ChangeStatus(STATUS_NONE)
		} else {
			c.ChangeStatus(STATUS_GAME_LOGIN)
		}
	} else if c.status == STATUS_GAME_LOGIN {
		msg := &clientmsg.Req_Login{
			UserID:     proto.String(c.userid),
			SessionKey: c.sessionkey,
			ServerID:   proto.Int32(GameServerID),
		}

		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_LOGIN, msg)
		c.ChangeStatus(STATUS_GAME_LOOP)
	} else if c.status == STATUS_GAME_MATCH {
		msg := &clientmsg.Req_Match{
			Action: clientmsg.MatchActionType.Enum(clientmsg.MatchActionType_MAT_JOIN),
			Mode:   clientmsg.MatchModeType.Enum(clientmsg.MatchModeType_MMT_NORMAL),
		}
		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_MATCH, msg)
	} else if c.status == STATUS_GAME_CLOSE {
		c.gconn.Close()
		c.ChangeStatus(STATUS_GAME_CONNECT)
	}

	if c.status == STATUS_GAME_LOOP || c.status == STATUS_BATTLE_LOOP {
		if c.nextpinggstime < time.Now().Unix() {
			c.nextpinggstime = time.Now().Unix() + 3

			msg := &clientmsg.Ping{
				ID: proto.Uint32(uint32(rand.Intn(10000))),
			}
			go Send(&c.gconn, clientmsg.MessageType_MT_PING, msg)
		}
	}
}

func (c *Client) recvGame() {
	for {
		if c.status == STATUS_GAME_LOOP || c.status == STATUS_BATTLE_LOOP {
			err, msgid, msgbuf := Recv(&c.gconn)
			if err != nil {
				continue
			}
			c.dispatch(msgid, msgbuf)
		}

		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func (c *Client) updateBattle() {
	if c.status == STATUS_BATTLE_CONNECT {
		c.bconn, c.err = net.Dial("tcp", c.battleaddr)
		if c.err != nil {
			c.ChangeStatus(STATUS_NONE)
		} else {
			c.ChangeStatus(STATUS_BATTLE)
		}
	} else if c.status == STATUS_BATTLE {
		msg := &clientmsg.Req_ConnectBS{
			RoomID:    proto.Int32(c.battleroomid),
			BattleKey: c.battlekey,
			CharID:    proto.String(c.charid),
		}
		go Send(&c.bconn, clientmsg.MessageType_MT_REQ_CONNECTBS, msg)
		c.ChangeStatus(STATUS_BATTLE_LOOP)
		c.startbattletime = time.Now().Unix()
	} else if c.status == STATUS_BATTLE_CLOSE {
		c.bconn.Close()
		c.ChangeStatus(STATUS_GAME_MATCH)
	}

	if c.status == STATUS_BATTLE_LOOP {
		//after battle begin
		if c.startbattletime != 0 && c.nextpingbstime < time.Now().Unix() {
			c.nextpingbstime = time.Now().Unix() + 3

			msg := &clientmsg.Ping{
				ID: proto.Uint32(uint32(rand.Intn(10000))),
			}
			go Send(&c.bconn, clientmsg.MessageType_MT_PING, msg)

			msg2 := &clientmsg.Transfer_Command{
				CharID:    proto.String(c.charid),
				ToCharID:  proto.String("all"),
				CommandID: proto.Int32(0),
			}
			go Send(&c.bconn, clientmsg.MessageType_MT_TRANSFER_COMMAND, msg2)
		}

		if c.startbattletime != 0 && (time.Now().Unix()-c.startbattletime > c.maxbattletime) {
			c.startbattletime = 0
			msg := &clientmsg.Req_EndBattle{
				TypeID: clientmsg.Type_BattleEndTypeID.Enum(clientmsg.Type_BattleEndTypeID_BEC_FINISH),
				CharID: proto.String(c.charid),
			}
			go Send(&c.bconn, clientmsg.MessageType_MT_REQ_ENDBATTLE, msg)
		}
	}
}

func (c *Client) recvBattle() {
	for {
		if c.status == STATUS_BATTLE_LOOP {
			err, msgid, msgbuf := Recv(&c.bconn)
			if err != nil {
				continue
			}
			c.dispatch(msgid, msgbuf)
		}

		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func (c *Client) updateSleep() {
	if c.status == STATUS_NONE {
		if c.nextlogintime < time.Now().Unix() {
			c.status = STATUS_LOGIN_CONNECT
		}
	}
}

func (c *Client) Update() {
	c.updateLogin()
	c.updateGame()
	c.updateBattle()
	c.updateSleep()
}

func (c *Client) Recv() {
	go c.recvLogin()
	go c.recvGame()
	go c.recvBattle()
}

func (c *Client) book(msgid interface{}, handler interface{}) {
	m.Lock()
	defer m.Unlock()
	c.routes[msgid] = handler
}

func (c *Client) dispatch(msgid interface{}, msgdata []byte) {
	m.Lock()
	defer m.Unlock()
	handler, ok := c.routes[msgid]
	if ok {
		handler.(func(c *Client, msgdata []byte))(c, msgdata)
	}
}

func (c *Client) Init(id int32) {
	c.id = id
	c.status = STATUS_NONE

	c.username = fmt.Sprintf("robot_%d", id)
	c.password = "123456"

	c.nextlogintime = time.Now().Unix()
	c.nextpingbstime = time.Now().Unix() + 3
	c.nextpinggstime = time.Now().Unix() + 3
	c.startbattletime = 0
	c.maxbattletime = 10

	c.routes = make(map[interface{}]interface{})
	c.book(clientmsg.MessageType_MT_RLT_REGISTER, handle_Rlt_Register)
	c.book(clientmsg.MessageType_MT_RLT_LOGIN, handle_Rlt_Login)
	c.book(clientmsg.MessageType_MT_RLT_NOTIFYBATTLEADDRESS, handle_Rlt_NotifyBattleAddress)
	c.book(clientmsg.MessageType_MT_RLT_CONNECTBS, handle_Rlt_ConnectBS)
	c.book(clientmsg.MessageType_MT_PONG, handle_Pong)
	c.book(clientmsg.MessageType_MT_TRANSFER_COMMAND, handle_Transfer_Command)
}

func (c *Client) Loop(id int32) {

	c.Init(id)
	c.Recv()
	for {
		select {
		case <-time.After(time.Duration(33) * time.Millisecond):
			c.Update()
		}
	}
}

func main() {
	m = new(sync.Mutex)

	w.Add(CLIENT_NUM)

	i := 1
	for i <= CLIENT_NUM {
		client := &Client{}
		go (*client).Loop(int32(i))
		i += 1
	}

	w.Wait()
}
