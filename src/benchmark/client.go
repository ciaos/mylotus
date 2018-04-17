package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"benchmark/testpb"

	"server/msg/clientmsg"

	"github.com/ciaos/leaf/kcp"
	"github.com/golang/protobuf/proto"
	"github.com/op/go-logging"
)

var tlog = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} > %{level:.4s} %{color:reset} %{message}`,
)

var w sync.WaitGroup
var m *sync.Mutex

type Client struct {
	id       int32
	username string
	password string

	gameserveraddr string
	sessionkey     []byte
	battlekey      []byte
	battleaddr     string
	battleroomid   int32

	userid uint32
	charid uint32

	prevstatus       testpb.ClientStatusType
	status           testpb.ClientStatusType
	changeStatusTime time.Time

	lconn net.Conn
	gconn net.Conn
	bconn net.Conn

	err error

	checktimeout time.Time

	frameid uint32

	nextpinggstime      int64
	nextpingbstime      int64
	lastgsheartbeattime int64
	lastbsheartbeattime int64

	startbattletime int64
	maxbattletime   int64

	startbattle bool

	routes map[interface{}]interface{}
}

func (c *Client) ChangeStatus(status testpb.ClientStatusType) {
	c.prevstatus = c.status
	c.status = status
	c.changeStatusTime = time.Now()
	tlog.Debugf("client %d CharID %v, Status %v\n", c.id, c.charid, c.status)

	switch c.status {
	case testpb.ClientStatusType_None:
		c.checktimeout = time.Now().Add(time.Second * time.Duration(randInt(1, 20)))
		c.ChangeStatus(testpb.ClientStatusType_Sleep_Before_Connect_LoginServer)

	//login server
	case testpb.ClientStatusType_Connect_LoginServer:
		c.lconn, c.err = net.Dial("tcp", LoginServerAddr)
		if c.err != nil {
			c.ChangeStatus(testpb.ClientStatusType_None)
		} else {
			c.ChangeStatus(testpb.ClientStatusType_Request_Register)
		}
	case testpb.ClientStatusType_Request_Register:
		msg := &clientmsg.Req_Register{
			UserName:      c.username,
			Password:      c.password,
			IsLogin:       true,
			ClientVersion: 0,
		}
		go Send(&c.lconn, clientmsg.MessageType_MT_REQ_REGISTER, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_LoginServer_Response)
	case testpb.ClientStatusType_Request_GameServer_List:
		msg := &clientmsg.Req_ServerList{
			Channel: 1,
		}
		go Send(&c.lconn, clientmsg.MessageType_MT_REQ_SERVERLIST, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_LoginServer_Response)
	case testpb.ClientStatusType_Wait_LoginServer_Response:
		c.checktimeout = time.Now().Add(time.Second * time.Duration(5))
	case testpb.ClientStatusType_Disconnect_LoginServer:
		if c.lconn != nil {
			c.lconn.Close()
		}
		c.checktimeout = time.Now().Add(time.Second * time.Duration(randInt(5, 20)))
		c.ChangeStatus(testpb.ClientStatusType_Sleep_Before_Connect_GameServer)

	//game server
	case testpb.ClientStatusType_Connect_GameServer:
		c.lastgsheartbeattime = time.Now().Unix()
		c.gconn, c.err = net.Dial("tcp", c.gameserveraddr)
		if c.err != nil {
			c.ChangeStatus(testpb.ClientStatusType_None)
		} else {
			c.ChangeStatus(testpb.ClientStatusType_Request_Login)
		}
	case testpb.ClientStatusType_Request_Login:
		msg := &clientmsg.Req_Login{
			UserID:     c.userid,
			SessionKey: c.sessionkey,
			ServerID:   GameServerID,
		}
		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_LOGIN, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_GameServer_Response)
	case testpb.ClientStatusType_Request_Match:
		msg := &clientmsg.Req_Match{
			Action: clientmsg.MatchActionType_MAT_JOIN,
			Mode:   clientmsg.MatchModeType_MMT_NORMAL,
		}
		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_MATCH, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_GameServer_Response)
	case testpb.ClientStatusType_Request_TeamOperate:
		msg := &clientmsg.Transfer_Team_Operate{
			Action:   clientmsg.TeamOperateActionType_TOA_SETTLE,
			CharID:   c.charid,
			CharType: 1001,
		}
		go Send(&c.gconn, clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_GameServer_Response)
	case testpb.ClientStatusType_Wait_GameServer_Response:
		c.checktimeout = time.Now().Add(time.Second * time.Duration(60))
	case testpb.ClientStatusType_Disconnect_GameServer:
		if c.gconn != nil {
			c.gconn.Close()
		}
		c.checktimeout = time.Now().Add(time.Second * time.Duration(randInt(5, 20)))
		c.ChangeStatus(testpb.ClientStatusType_Sleep_Before_Connect_GameServer)

	//battle server
	case testpb.ClientStatusType_Connect_BattleServer:
		c.frameid = 0
		c.lastbsheartbeattime = time.Now().Unix()
		c.bconn, c.err = kcp.Dial(c.battleaddr)
		if c.err != nil {
			c.ChangeStatus(testpb.ClientStatusType_Disconnect_GameServer)
		} else {
			c.ChangeStatus(testpb.ClientStatusType_Request_ConnectBS)
		}
	case testpb.ClientStatusType_Request_ConnectBS:
		msg := &clientmsg.Req_ConnectBS{
			RoomID:    c.battleroomid,
			BattleKey: c.battlekey,
			CharID:    c.charid,
		}
		go Send(&c.bconn, clientmsg.MessageType_MT_REQ_CONNECTBS, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_BattleServer_Response)
	case testpb.ClientStatusType_Request_Progress:
		msg := &clientmsg.Transfer_Loading_Progress{
			CharID:   c.charid,
			Progress: 100,
		}
		go Send(&c.bconn, clientmsg.MessageType_MT_TRANSFER_LOADING_PROGRESS, msg)
		c.ChangeStatus(testpb.ClientStatusType_Wait_BattleServer_Response)
	case testpb.ClientStatusType_Request_EndBattle:
		msg := &clientmsg.Req_EndBattle{
			TypeID: clientmsg.Type_BattleEndTypeID_BEC_FINISH,
			CharID: c.charid,
		}
		go Send(&c.bconn, clientmsg.MessageType_MT_REQ_ENDBATTLE, msg)
	case testpb.ClientStatusType_Disconnect_BattleServer:
		c.startbattle = false
		if c.bconn != nil {
			c.bconn.Close()
		}
		c.checktimeout = time.Now().Add(time.Second * time.Duration(randInt(5, 20)))
		c.ChangeStatus(testpb.ClientStatusType_Sleep_Before_Request_Match)
	}
}

func handle_Pong(c *Client, msgdata []byte) {
	rsp := &clientmsg.Pong{}
	proto.Unmarshal(msgdata, rsp)
	c.lastgsheartbeattime = time.Now().Unix()
	//tlog.Debugf("client %d recv pong %d\n", c.id, rsp.GetID())
}

func handle_Transfer_HeartBeat(c *Client, msgdata []byte) {
	rsp := &clientmsg.Transfer_Battle_Heartbeat{}
	proto.Unmarshal(msgdata, rsp)
	c.lastbsheartbeattime = time.Now().UnixNano()

	//tlog.Debugf("client %d CharID %v, ping %v\n", c.id, c.charid, (uint64(time.Now().UnixNano())-rsp.TickTime)/(1000*1000))
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
	} else if rsp.RetCode == clientmsg.Type_LoginRetCode_LRC_OK {
		c.userid = rsp.UserID
		c.sessionkey = rsp.SessionKey
		c.ChangeStatus(testpb.ClientStatusType_Request_GameServer_List)
	} else {
		c.lconn.Close()
		c.ChangeStatus(testpb.ClientStatusType_None)
	}
}

func handle_Rlt_ServerList(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_ServerList{}
	proto.Unmarshal(msgdata, rsp)
	for _, server := range rsp.ServerList {
		if server.ServerID == GameServerID {
			c.gameserveraddr = server.ConnectAddr
		}
	}
	c.ChangeStatus(testpb.ClientStatusType_Disconnect_LoginServer)
}

func handle_Rlt_Login(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_Login{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.RetCode == clientmsg.Type_GameRetCode_GRC_OK {

		if rsp.IsNewCharacter {
			msg := &clientmsg.Req_SetCharName{
				CharName: "robot_" + strconv.Itoa(int(rsp.CharID)),
			}
			go Send(&c.gconn, clientmsg.MessageType_MT_REQ_SETCHARNAME, msg)
		}

		c.charid = rsp.CharID
		c.checktimeout = time.Now().Add(time.Second * time.Duration(randInt(5, 20)))
		c.ChangeStatus(testpb.ClientStatusType_Sleep_Before_Request_Match)
	} else {
		c.gconn.Close()
		c.ChangeStatus(testpb.ClientStatusType_None)
	}
}

func handle_Rlt_Match(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_Match{}
	proto.Unmarshal(msgdata, rsp)

	if rsp.RetCode == clientmsg.Type_GameRetCode_GRC_MATCH_ERROR {
		c.ChangeStatus(testpb.ClientStatusType_Request_Match)
		return
	} else if rsp.RetCode == clientmsg.Type_GameRetCode_GRC_MATCH_OK {
		msg := &clientmsg.Req_Match{
			Action: clientmsg.MatchActionType_MAT_CONFIRM,
			Mode:   clientmsg.MatchModeType_MMT_RANK,
		}
		go Send(&c.gconn, clientmsg.MessageType_MT_REQ_MATCH, msg)
	} else if rsp.RetCode == clientmsg.Type_GameRetCode_GRC_MATCH_ALL_CONFIRMED {
		c.ChangeStatus(testpb.ClientStatusType_Request_TeamOperate)
	}
}

func handle_Rlt_TeamOperate(c *Client, msgdata []byte) {
	//	rsp := &clientmsg.Transfer_Team_Operate{}
	//	proto.Unmarshal(msgdata, rsp)
	//	tlog.Debug("TeamOperate %d %d %d \n", rsp.Action, rsp.CharID, rsp.CharType)
}

func handle_Rlt_NotifyBattleAddress(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_NotifyBattleAddress{}
	proto.Unmarshal(msgdata, rsp)
	c.battlekey = rsp.BattleKey
	c.battleaddr = rsp.BattleAddr
	c.battleroomid = rsp.RoomID

	c.checktimeout = time.Now().Add(time.Second * time.Duration(randInt(1, 5)))
	c.ChangeStatus(testpb.ClientStatusType_Sleep_Before_Connect_BattleServer)

	c.maxbattletime = OneBattleTime + randInt(1, 2)
}

func handle_Rlt_ConnectBS(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_ConnectBS{}
	proto.Unmarshal(msgdata, rsp)
	if rsp.RetCode != clientmsg.Type_BattleRetCode_BRC_OK {
		c.ChangeStatus(testpb.ClientStatusType_Disconnect_BattleServer)
		return
	}
	c.ChangeStatus(testpb.ClientStatusType_Request_Progress)
}

func handle_Rlt_StartBattle(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_StartBattle{}
	proto.Unmarshal(msgdata, rsp)
	c.startbattle = true
	c.startbattletime = time.Now().Unix()
}

func handle_Rlt_EndBattle(c *Client, msgdata []byte) {
	rsp := &clientmsg.Rlt_EndBattle{}
	proto.Unmarshal(msgdata, rsp)

	c.ChangeStatus(testpb.ClientStatusType_Disconnect_BattleServer)
}

func handle_Transfer_Command(c *Client, msgdata []byte) {
	if c.status != testpb.ClientStatusType_Wait_BattleServer_Response || c.startbattletime == 0 {
		return
	}

	rsp := &clientmsg.Transfer_Command{}
	proto.Unmarshal(msgdata, rsp)

	if len(rsp.Messages) > 0 {
		ping := &clientmsg.Ping{}
		err := proto.Unmarshal(rsp.Messages[0].Msgdata, ping)
		if err != nil {
			tlog.Errorf("Unmartial Error")
			return
		}
		//	tlog.Debugf("client %d recv tranfer_cmd from %d, frame %d PingID %d Total %d\n", c.charid, rsp.Messages[0].CharID, rsp.FrameID, ping.ID, len(rsp.Messages))
	} else {
		//	tlog.Debugf("client %d recv tranfer_cmd from server, frame %d Total %d\n", c.charid, rsp.FrameID, len(rsp.Messages))
	}
	if rsp.FrameID != c.frameid+1 {
		tlog.Fatalf("rsp.frameid %v client.frameid %v client.id %v client.charid %v startbattletime %v", rsp.FrameID, c.frameid, c.id, c.charid, c.startbattletime)
	}
	c.frameid = rsp.FrameID
	//fmt.Printf("client %d frame %v CharID %v recv transfer command from %v\n", c.id, rsp.FrameID, c.charid, rsp.CharID)
}

func (c *Client) updateLogin() {
	if c.status == testpb.ClientStatusType_Sleep_Before_Connect_LoginServer {
		if c.checktimeout.Unix() < time.Now().Unix() {
			c.ChangeStatus(testpb.ClientStatusType_Connect_LoginServer)
		}
	} else if c.status == testpb.ClientStatusType_Wait_LoginServer_Response {
		if c.checktimeout.Unix() < time.Now().Unix() {
			c.lconn.Close()
			c.ChangeStatus(testpb.ClientStatusType_None)
		}
	}
}

func (c *Client) recvLogin() {
	for {
		if c.status == testpb.ClientStatusType_Wait_LoginServer_Response {
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
	if c.status == testpb.ClientStatusType_Sleep_Before_Connect_GameServer {
		if c.checktimeout.Unix() < time.Now().Unix() {
			c.ChangeStatus(testpb.ClientStatusType_Connect_GameServer)
		}
	} else if c.status == testpb.ClientStatusType_Sleep_Before_Request_Match {
		if c.checktimeout.Unix() < time.Now().Unix() {
			c.ChangeStatus(testpb.ClientStatusType_Request_Match)
		}
	} else if c.status == testpb.ClientStatusType_Wait_GameServer_Response || c.status == testpb.ClientStatusType_Wait_BattleServer_Response {
		if c.status == testpb.ClientStatusType_Wait_GameServer_Response {
			if c.checktimeout.Unix() < time.Now().Unix() {
				tlog.Errorf("client %d status %v checktimeout\n", c.id, c.status)
				c.ChangeStatus(testpb.ClientStatusType_Disconnect_GameServer)
				return
			}
		}

		if c.nextpinggstime < time.Now().Unix() {
			c.nextpinggstime = time.Now().Unix() + 3

			msg := &clientmsg.Ping{
				ID: uint32(rand.Intn(10000)),
			}
			go Send(&c.gconn, clientmsg.MessageType_MT_PING, msg)

			if time.Now().Unix()-c.lastgsheartbeattime > 20 {

				tlog.Errorf("client %d gs ping timeout\n", c.id)
				c.ChangeStatus(testpb.ClientStatusType_Disconnect_GameServer)
			}
		}
	}
}

func (c *Client) recvGame() {
	for {
		if c.status == testpb.ClientStatusType_Wait_GameServer_Response || c.status == testpb.ClientStatusType_Wait_BattleServer_Response {
			err, msgid, msgbuf := Recv(&c.gconn)
			if err != nil {
				c.ChangeStatus(testpb.ClientStatusType_Disconnect_GameServer)
				continue
			}
			c.dispatch(msgid, msgbuf)
		}

		time.Sleep(time.Duration(1) * time.Microsecond)
	}
}

func (c *Client) updateBattle() {
	if c.status == testpb.ClientStatusType_Sleep_Before_Connect_BattleServer {
		if c.checktimeout.Unix() < time.Now().Unix() {
			c.ChangeStatus(testpb.ClientStatusType_Connect_BattleServer)
		}
	} else if c.status == testpb.ClientStatusType_Wait_BattleServer_Response {
		//after battle begin
		if c.startbattle {
			//send heartbeat
			if c.startbattletime != 0 && c.nextpingbstime < time.Now().Unix() {
				c.nextpingbstime = time.Now().Unix() + 1

				msg := &clientmsg.Transfer_Battle_Heartbeat{}
				msg.TickTime = uint64(time.Now().UnixNano())
				go Send(&c.bconn, clientmsg.MessageType_MT_TRANSFER_BATTLE_HEARTBEAT, msg)
			}
			if time.Now().Unix()-c.lastbsheartbeattime > 20 {
				tlog.Errorf("client %d bs heartbeat timeout\n", c.id)
				c.ChangeStatus(testpb.ClientStatusType_Disconnect_BattleServer)
				return
			}

			//send transfer cmd
			/*	i := 0
				for i < 1 {

					ping := &clientmsg.Ping{
						ID: c.charid,
					}
					msgbuff, _ := proto.Marshal(ping)
					cdata := &clientmsg.Transfer_Command_CommandData{
						Msgdata: msgbuff,
					}

					msg := &clientmsg.Transfer_Command{}
					msg.Messages = append(msg.Messages, cdata)
					go Send(&c.bconn, clientmsg.MessageType_MT_TRANSFER_COMMAND, msg)

					i += 1
				}*/
		}

		if c.startbattletime != 0 && (time.Now().Unix()-c.startbattletime > c.maxbattletime) {
			c.startbattletime = 0
			c.ChangeStatus(testpb.ClientStatusType_Request_EndBattle)
			c.ChangeStatus(testpb.ClientStatusType_Wait_BattleServer_Response)
		}
	}
}

func (c *Client) recvBattle() {
	for {
		if c.status == testpb.ClientStatusType_Wait_BattleServer_Response {
			err, msgid, msgbuf := Recv(&c.bconn)
			if err != nil {
				c.ChangeStatus(testpb.ClientStatusType_Disconnect_BattleServer)
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

	//状态卡60s则重启
	if time.Now().Unix()-c.changeStatusTime.Unix() > c.maxbattletime+30 {
		tlog.Errorf("charid %d status %v timeout prevstatus %v changestatustime %v checktime %v now %v maxbattle %v c.startbattletime %v\n", c.charid, c.status, c.prevstatus, c.changeStatusTime.Unix(), c.checktimeout.Unix(), time.Now().Unix(), c.maxbattletime, c.startbattletime)
		if c.lconn != nil {
			c.lconn.Close()
		}
		if c.gconn != nil {
			c.gconn.Close()
		}
		if c.bconn != nil {
			c.bconn.Close()
		}

		c.ChangeStatus(testpb.ClientStatusType_None)
	}
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
		//if msgid != clientmsg.MessageType_MT_PONG && msgid != clientmsg.MessageType_MT_TRANSFER_COMMAND {
		//	tlog.Debugf("clientid %d msgid %d", c.id, msgid)
		//}
		handler.(func(c *Client, msgdata []byte))(c, msgdata)
	}
}

func (c *Client) Init(id int32) {
	c.id = id

	c.username = fmt.Sprintf("robot_%d", id)
	c.password = "123456"

	//	c.username = fmt.Sprintf("%s", "gaojiangshan")
	//	c.password = fmt.Sprintf("%d", 123456)

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
	c.book(clientmsg.MessageType_MT_RLT_SERVERLIST, handle_Rlt_ServerList)
	c.book(clientmsg.MessageType_MT_TRANSFER_TEAMOPERATE, handle_Rlt_TeamOperate)
	c.book(clientmsg.MessageType_MT_TRANSFER_BATTLE_HEARTBEAT, handle_Transfer_HeartBeat)

	c.ChangeStatus(testpb.ClientStatusType_None)
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
	randNum := rand.Intn(max-min) + min
	return int64(randNum)
}

var m_client = make(map[int]*Client, ClientNum)

func stat(fin chan int) {
	for {
		select {
		case _ = <-fin:
			return
		case <-time.After(time.Second * 2):
			m_stat := make(map[testpb.ClientStatusType]int)
			for _, m_client := range m_client {
				m_stat[m_client.status] += 1
			}
			for k, v := range m_stat {
				tlog.Infof("Status:%v\t Count:%v\t\n", k, v)
			}
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendleveled := logging.AddModuleLevel(backendFormatter)
	backendleveled.SetLevel(logging.INFO, "")

	logging.SetBackend(backendleveled)

	m = new(sync.Mutex)

	w.Add(ClientNum)

	i := 1
	for i <= ClientNum {
		client := &Client{}
		go (*client).Loop(int32(i))
		m_client[i] = client
		i += 1
	}

	fin := make(chan int, 1)
	go stat(fin)
	w.Wait()

	fin <- 1
}
