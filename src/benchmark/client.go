package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"server/msg/clientmsg"

	"github.com/ciaos/leaf/kcp"
	"github.com/golang/protobuf/proto"
	"github.com/op/go-logging"
)

var tlog = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} > %{level:.4s} %{color:reset} %{message}`,
)

const (
	CLIENT_NUM        = 2
	BATTLE_BASIC_TIME = 10

	STATUS_NONE = "STATUS_NONE"

	STATUS_LOGIN_CONNECT = "connect_login_server"
	STATUS_LOGIN         = "start_register"
	STATUS_LOGIN_LOOP    = "loop_login"
	STATUS_LOGIN_CLOSE   = "disconnect_login_server"

	STATUS_GAME_CONNECT      = "connect_game_server"
	STATUS_GAME_LOGIN        = "start_signin"
	STATUS_GAME_MATCH        = "start_match"
	STATUS_GAME_TEAM_OPERATE = "team_operate"
	STATUS_GAME_LOOP         = "loop_game"
	STATUS_GAME_CLOSE        = "disconnect_game_server"

	STATUS_BATTLE_CONNECT  = "connect_battle_server"
	STATUS_BATTLE_PROGRESS = "loading_progress"
	STATUS_BATTLE          = "join_battle_room"
	STATUS_BATTLE_WAITEND  = "wait_battle_end_rsp"
	STATUS_BATTLE_LOOP     = "loop_battle"
	STATUS_BATTLE_CLOSE    = "disconnect_battle_server"
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

	userid uint32
	charid uint32
	status string

	lconn net.Conn
	gconn net.Conn
	bconn *kcp.UDPSession

	err error

	nextlogintime    int64
	nextregistertime int64
	nextmatchtime    int64

	nextpinggstime int64
	nextpingbstime int64

	startbattletime int64
	maxbattletime   int64

	startbattle bool

	routes map[interface{}]interface{}
}

func (c *Client) ChangeStatus(status string) {
	c.status = status
	tlog.Debugf("client %d %s\n", c.id, c.status)
}

func handle_Pong(c *Client, msgdata []byte) {
	rsp := &clientmsg.Pong{}
	proto.Unmarshal(msgdata, rsp)
	//fmt.Printf("client %d recv pong %d\n", c.id, rsp.GetID())
}

func handle_Rlt_Register(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_Register{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.RetCode == clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST {
		msg := &clientmsg.Req_Register{
			UserName:      c.username,
			Password:      c.password,
			IsLogin:       false,
			ClientVersion: 0,
		}
		go Send(&c.lconn, clientmsg.MessageType_MT_REQ_REGISTER, msg)
	} else if rsp.RetCode == clientmsg.Type_LoginRetCode_LRC_NONE {
		c.userid = rsp.UserID
		c.sessionkey = rsp.SessionKey
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
	if rsp.RetCode == clientmsg.Type_GameRetCode_GRC_NONE {
		c.charid = rsp.CharID
		c.nextmatchtime = time.Now().Unix() + randInt(1, 5)
		c.ChangeStatus(STATUS_GAME_MATCH)
	} else {
		c.nextlogintime = time.Now().Unix() + 5
		c.ChangeStatus(STATUS_NONE)
		c.gconn.Close()
	}
}

func handle_Rlt_Match(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_Match{}
	proto.Unmarshal(msgdata, rsp)

	c.ChangeStatus(STATUS_GAME_TEAM_OPERATE)
	msg := &clientmsg.Transfer_Team_Operate{
		Action:   clientmsg.TeamOperateActionType_TOA_CHOOSE,
		CharID:   c.charid,
		CharType: 1001,
	}
	go Send(&c.gconn, clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE, msg)
	msg = &clientmsg.Transfer_Team_Operate{
		Action: clientmsg.TeamOperateActionType_TOA_SETTLE,
		CharID: c.charid,
	}
	go Send(&c.gconn, clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE, msg)
	c.ChangeStatus(STATUS_GAME_LOOP)
}

func handle_Rlt_NotifyBattleAddress(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_NotifyBattleAddress{}
	proto.Unmarshal(msgdata, rsp)
	c.battlekey = rsp.BattleKey
	c.battleaddr = rsp.BattleAddr
	c.battleroomid = rsp.RoomID
	c.ChangeStatus(STATUS_BATTLE_CONNECT)

	c.maxbattletime = BATTLE_BASIC_TIME + randInt(1, 2)
}

func handle_Rlt_ConnectBS(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_ConnectBS{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.RetCode != clientmsg.Type_BattleRetCode_BRC_NONE {
		c.ChangeStatus(STATUS_BATTLE_CLOSE)
	}

	c.ChangeStatus(STATUS_BATTLE_PROGRESS)
	msg := &clientmsg.Transfer_Loading_Progress{
		CharID:   c.charid,
		Progress: 100,
	}
	go SendKCP(c.bconn, clientmsg.MessageType_MT_TRANSFER_LOADING_PROGRESS, msg)
	c.ChangeStatus(STATUS_BATTLE_LOOP)
}

func handle_Rlt_StartBattle(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_StartBattle{}
	proto.Unmarshal(msgdata, rsp)
	c.startbattle = true

	tlog.Debugf("startbattle %d\n", c.charid)
}

func handle_Rlt_EndBattle(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_EndBattle{}
	proto.Unmarshal(msgdata, rsp)

	c.ChangeStatus(STATUS_BATTLE_CLOSE)
}

func handle_Transfer_Command(c *Client, msgdata []byte) {
	rsp := &clientmsg.Transfer_Command{}
	proto.Unmarshal(msgdata, rsp)

	//	fmt.Printf("client %d CharID %s recv transfer command from %s\n", c.id, c.charid, rsp.GetCharID())
}

func (c *Client) updateLogin() {
	if c.status == STATUS_LOGIN_CONNECT {
		if c.nextregistertime < time.Now().Unix() {
			c.nextregistertime = 0
			c.lconn, c.err = net.Dial("tcp", LoginServerAddr)
			if c.err != nil {
				c.ChangeStatus(STATUS_NONE)
			} else {
				c.ChangeStatus(STATUS_LOGIN)
			}
		}
	} else if c.status == STATUS_LOGIN {
		msg := &clientmsg.Req_Register{
			UserName:      c.username,
			Password:      c.password,
			IsLogin:       true,
			ClientVersion: 0,
		}
		go Send(&c.lconn, clientmsg.MessageType_MT_REQ_REGISTER, msg)
		c.ChangeStatus(STATUS_LOGIN_LOOP)
	} else if c.status == STATUS_LOGIN_CLOSE {
		c.lconn.Close()
		c.nextlogintime = time.Now().Unix() + randInt(1, 3)
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

		time.Sleep(time.Duration(1) * time.Microsecond)
	}
}

func (c *Client) updateGame() {
	if c.status == STATUS_GAME_CONNECT {
		if c.nextlogintime < time.Now().Unix() {
			c.nextlogintime = 0
			c.gconn, c.err = net.Dial("tcp", GameServerAddr)
			if c.err != nil {
				c.ChangeStatus(STATUS_NONE)
			} else {
				c.ChangeStatus(STATUS_GAME_LOGIN)
			}
		}
	} else if c.status == STATUS_GAME_LOGIN {
		time.Sleep(time.Duration(1) * time.Second)
		msg := &clientmsg.Req_Login{
			UserID:     c.userid,
			SessionKey: c.sessionkey,
			ServerID:   GameServerID,
		}

		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_LOGIN, msg)
		c.ChangeStatus(STATUS_GAME_LOOP)
	} else if c.status == STATUS_GAME_MATCH {
		if c.nextmatchtime < time.Now().Unix() {
			c.nextmatchtime = 0
			msg := &clientmsg.Req_Match{
				Action: clientmsg.MatchActionType_MAT_JOIN,
				Mode:   clientmsg.MatchModeType_MMT_NORMAL,
			}
			go Send(&c.gconn, clientmsg.MessageType_MT_REQ_MATCH, msg)
			c.ChangeStatus(STATUS_GAME_LOOP)
		}
	} else if c.status == STATUS_GAME_CLOSE {
		if c.gconn != nil {
			c.gconn.Close()
		}
		c.ChangeStatus(STATUS_GAME_CONNECT)
	}

	if c.status == STATUS_GAME_LOOP || c.status == STATUS_BATTLE_LOOP {
		if c.nextpinggstime < time.Now().Unix() {
			c.nextpinggstime = time.Now().Unix() + 3

			msg := &clientmsg.Ping{
				ID: uint32(rand.Intn(10000)),
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
				c.ChangeStatus(STATUS_GAME_CLOSE)
				continue
			}
			c.dispatch(msgid, msgbuf)
		}

		time.Sleep(time.Duration(1) * time.Microsecond)
	}
}

func (c *Client) updateBattle() {
	if c.status == STATUS_BATTLE_CONNECT {
		c.bconn, c.err = kcp.Dial(kcp.MODE_FAST, c.battleaddr)
		if c.err != nil {
			c.ChangeStatus(STATUS_NONE)
		} else {
			c.ChangeStatus(STATUS_BATTLE)
		}
		c.startbattle = false
	} else if c.status == STATUS_BATTLE {
		msg := &clientmsg.Req_ConnectBS{
			RoomID:    c.battleroomid,
			BattleKey: c.battlekey,
			CharID:    c.charid,
		}
		go SendKCP(c.bconn, clientmsg.MessageType_MT_REQ_CONNECTBS, msg)

		c.ChangeStatus(STATUS_BATTLE_LOOP)
		c.startbattletime = time.Now().Unix()
	} else if c.status == STATUS_BATTLE_CLOSE {
		c.bconn.Close()
		c.nextmatchtime = time.Now().Unix() + randInt(1, 5)
		c.ChangeStatus(STATUS_GAME_MATCH)
	}

	if c.status == STATUS_BATTLE_LOOP {
		//after battle begin
		if c.startbattletime != 0 && c.nextpingbstime < time.Now().Unix() {
			c.nextpingbstime = time.Now().Unix() + 3

			msg := &clientmsg.Ping{
				ID: uint32(rand.Intn(10000)),
			}
			go SendKCP(c.bconn, clientmsg.MessageType_MT_PING, msg)

		}

		if c.startbattle {
			i := 1
			for i < 3 {
				msg := &clientmsg.Transfer_Command{
					CharID: c.charid,
				}
				go SendKCP(c.bconn, clientmsg.MessageType_MT_TRANSFER_COMMAND, msg)

				i += 1
			}
		}

		if c.startbattletime != 0 && (time.Now().Unix()-c.startbattletime > c.maxbattletime) {
			c.startbattletime = 0
			msg := &clientmsg.Req_EndBattle{
				TypeID: clientmsg.Type_BattleEndTypeID_BEC_FINISH,
				CharID: c.charid,
			}
			c.ChangeStatus(STATUS_BATTLE_WAITEND)
			go SendKCP(c.bconn, clientmsg.MessageType_MT_REQ_ENDBATTLE, msg)
		}
	}
}

func (c *Client) recvBattle() {
	for {
		if c.status == STATUS_BATTLE_LOOP || c.status == STATUS_BATTLE_WAITEND {
			err, msgid, msgbuf := RecvKCP(c.bconn)
			if err != nil {
				c.ChangeStatus(STATUS_BATTLE_CLOSE)
				continue
			}
			c.dispatch(msgid, msgbuf)
		}

		time.Sleep(time.Duration(1) * time.Microsecond)
	}
}

func (c *Client) Update() {
	c.updateLogin()
	c.updateGame()
	c.updateBattle()
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
		if msgid != clientmsg.MessageType_MT_PONG && msgid != clientmsg.MessageType_MT_TRANSFER_COMMAND {
			tlog.Debugf("clientid %d msgid %d", c.id, msgid)
		}
		handler.(func(c *Client, msgdata []byte))(c, msgdata)
	}
}

func (c *Client) Init(id int32) {
	c.id = id
	c.nextregistertime = time.Now().Unix() + randInt(1, 5)
	c.status = STATUS_LOGIN_CONNECT

	c.username = fmt.Sprintf("robot_%d", id)
	c.password = "123456"

	c.nextlogintime = time.Now().Unix()
	c.nextpingbstime = time.Now().Unix() + 3
	c.nextpinggstime = time.Now().Unix() + 3
	c.startbattletime = 0
	c.maxbattletime = 10
	c.startbattle = false

	c.routes = make(map[interface{}]interface{})
	c.book(clientmsg.MessageType_MT_RLT_REGISTER, handle_Rlt_Register)
	c.book(clientmsg.MessageType_MT_RLT_LOGIN, handle_Rlt_Login)
	c.book(clientmsg.MessageType_MT_RLT_NOTIFYBATTLEADDRESS, handle_Rlt_NotifyBattleAddress)
	c.book(clientmsg.MessageType_MT_RLT_CONNECTBS, handle_Rlt_ConnectBS)
	c.book(clientmsg.MessageType_MT_PONG, handle_Pong)
	c.book(clientmsg.MessageType_MT_RLT_ENDBATTLE, handle_Rlt_EndBattle)
	c.book(clientmsg.MessageType_MT_TRANSFER_COMMAND, handle_Transfer_Command)
	c.book(clientmsg.MessageType_MT_RLT_MATCH, handle_Rlt_Match)
	c.book(clientmsg.MessageType_MT_RLT_STARTBATTLE, handle_Rlt_StartBattle)
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

func randInt(min, max int) int64 {
	rand.Seed(time.Now().Unix())
	randNum := rand.Intn(max-min) + min
	return int64(randNum)
}

func main() {
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)

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
