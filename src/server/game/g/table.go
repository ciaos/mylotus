package g

import (
	"fmt"
	"math/rand"
	"server/gamedata"
	"server/gamedata/cfg"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"strconv"
	"strings"
	//	"sync"
	"time"

	"github.com/ciaos/leaf/log"
)

const (
	MATCH_OK                = "match_ok"
	MATCH_CONTINUE          = "match_continue"          //匹配中
	MATCH_TIMEOUT           = "match_timeout"           //匹配超时(补充AI)
	MATCH_CHARTYPE_CHOOSING = "match_chartype_choosing" //选择角色中
	MATCH_CHARTYPE_FIXED    = "match_chartype_fixed"    //角色确定
	MATCH_BEGIN_ALLOCROOM   = "match_begin_allocroom"   //开始申请房间
	MATCH_ALLOCROOM         = "match_allocroom"         //申请房间中
	MATCH_EMPTY             = "match_empty"             //桌子已无人
	MATCH_FINISH            = "match_finish"            //
	MATCH_ERROR             = "match_error"

	SEAT_NONE  = "seat_none"
	SEAT_READY = "seat_ready"
)

type Seat struct {
	charid     uint32
	charname   string
	jointime   int64
	serverid   int32
	servertype string
	chartype   int32
	ownerid    uint32
	teamid     int32
	status     string
}

//for match server
type Table struct {
	seats         []*Seat
	createtime    int64
	checktime     int64
	matchmode     int32
	status        string
	tableid       int32
	modeplayercnt int32
}

var TableManager = make(map[int32]*Table)
var PlayerTableIDMap = make(map[uint32]int32)
var tableid int32

//var mTableID *sync.Mutex

func InitTableManager() {
	tableid = 0

	//	mTableID = new(sync.Mutex)
}

func allocTableID() {
	//mTableID.Lock()
	//defer mTableID.Unlock()
	tableid += 1
	if tableid > MAX_TABLE_COUNT {
		tableid = 1
	}
}

func fillRobotToTable(table *Table) {
	r := gamedata.CSVMatchMode.Index((*table).matchmode)
	if r == nil {
		return
	}
	row := r.(*cfg.MatchMode)

	robotnum := row.PlayerCnt - len((*table).seats)
	ownerid := (*table).seats[0].charid

	i := 1
	for i <= robotnum {

		charid := uint32(rand.Intn(100000))
		seat := &Seat{
			charid:     charid,
			jointime:   0,
			serverid:   0,
			servertype: "",
			charname:   strconv.Itoa(int(charid)),
			chartype:   0,
			ownerid:    ownerid,
			status:     SEAT_NONE,
			teamid:     int32(len((*table).seats) % 2),
		}
		(*table).seats = append((*table).seats, seat)
		log.Debug("fillRobotToTable RobotID %v OwnerID %v", (*seat).charid, (*seat).ownerid)
		i++
	}
}

func notifyMatchResultToTable(table *Table, retcode clientmsg.Type_GameRetCode) {

	msg := &clientmsg.Rlt_Match{
		RetCode: retcode,
		Members: []*clientmsg.Rlt_Match_MemberInfo{},
	}

	if retcode == clientmsg.Type_GameRetCode_GRC_MATCH_OK {
		for _, seat := range table.seats {
			member := &clientmsg.Rlt_Match_MemberInfo{}
			member.CharID = (*seat).charid
			member.OwnerID = (*seat).ownerid
			member.TeamID = (*seat).teamid
			member.CharName = (*seat).charname
			member.CharType = (*seat).chartype
			msg.Members = append(msg.Members, member)
		}
	}
	for _, seat := range table.seats {
		if (*seat).ownerid == 0 {
			log.Debug("notify MatchResult CharID %v Result %v MemberCount %v", (*seat).charid, retcode, len(msg.Members))
			go SendMessageTo((*seat).serverid, (*seat).servertype, (*seat).charid, uint32(proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT), msg)
		}
	}
}

func changeTableStatus(table *Table, status string) {
	(*table).status = status
	log.Debug("changeTableStatus Table %v Status %v", (*table).tableid, (*table).status)

	if (*table).status == MATCH_ERROR {
		//notify all member error
		notifyMatchResultToTable(table, clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		deleteTableSeatInfo((*table).tableid)
		DeleteTable((*table).tableid)
	} else if (*table).status == MATCH_EMPTY {
		DeleteTable((*table).tableid)
	} else if (*table).status == MATCH_OK {
		//notify all member to choose
		notifyMatchResultToTable(table, clientmsg.Type_GameRetCode_GRC_MATCH_OK)
		(*table).checktime = time.Now().Unix()
		changeTableStatus(table, MATCH_CHARTYPE_CHOOSING)
	} else if (*table).status == MATCH_TIMEOUT {
		fillRobotToTable(table)
		notifyMatchResultToTable(table, clientmsg.Type_GameRetCode_GRC_MATCH_OK)
		(*table).checktime = time.Now().Unix()
		changeTableStatus(table, MATCH_CHARTYPE_CHOOSING)
	} else if (*table).status == MATCH_BEGIN_ALLOCROOM {
		table.allocBattleRoom()
		(*table).checktime = time.Now().Unix()
		changeTableStatus(table, MATCH_ALLOCROOM)
	} else if (*table).status == MATCH_FINISH {
		deleteTableSeatInfo(tableid)
		table.seats = append([]*Seat{}) //clear seats
		DeleteTable(tableid)
	}
}

func (table *Table) update(now *time.Time) {
	r := gamedata.CSVMatchMode.Index((*table).matchmode)
	if r == nil {
		log.Error("CSVMatchMode ModeID %v Not Found", (*table).matchmode)
		changeTableStatus(table, MATCH_ERROR)
		return
	}
	row := r.(*cfg.MatchMode)

	if (*table).status == MATCH_CONTINUE {
		//匹配超时
		if (*now).Unix()-(*table).createtime > int64(row.MatchTimeOutSec) {
			log.Debug("Tableid %v MatchTimeout Createtime %v Now %v", (*table).tableid, (*table).createtime, (*now).Unix())
			changeTableStatus(table, MATCH_TIMEOUT)
			return
		}

		if len((*table).seats) >= row.PlayerCnt {
			changeTableStatus(table, MATCH_OK)
		} else if len((*table).seats) <= 0 {
			changeTableStatus(table, MATCH_EMPTY)
		}
	} else if (*table).status == MATCH_CHARTYPE_CHOOSING {
		if (*now).Unix()-(*table).checktime > int64(row.ChooseTimeOutSec) {
			log.Debug("Tableid %v ChooseTimeout checktime %v Now %v", (*table).tableid, (*table).checktime, (*now).Unix())
			(*table).checktime = (*now).Unix()
			changeTableStatus(table, MATCH_CHARTYPE_FIXED)
		}
	} else if (*table).status == MATCH_CHARTYPE_FIXED {
		if (*now).Unix()-(*table).checktime > int64(row.FixedWaitTimeSec) {
			log.Debug("Tableid %v FixedWaitTimeout checktime %v Now %v", (*table).tableid, (*table).checktime, (*now).Unix())
			(*table).checktime = (*now).Unix()
			changeTableStatus(table, MATCH_BEGIN_ALLOCROOM)
		}
	} else if (*table).status == MATCH_ALLOCROOM {
		if (*now).Unix()-(*table).checktime > 5 { //申请房间超时，解散队伍
			log.Error("Tableid %v Allocroom TimeOut checktime %v Now %v", (*table).tableid, (*table).checktime, (*now).Unix())
			(*table).checktime = (*now).Unix()
			changeTableStatus(table, MATCH_ERROR)
		}
	}
}

func (table *Table) allocBattleRoom() {

	innerReq := &proxymsg.Proxy_MS_BS_AllocBattleRoom{
		Matchtableid: table.tableid,
		Matchmode:    table.matchmode,
	}

	for _, seat := range table.seats {
		member := &proxymsg.Proxy_MS_BS_AllocBattleRoom_MemberInfo{
			CharID:       seat.charid,
			CharName:     seat.charname,
			CharType:     seat.chartype,
			TeamID:       seat.teamid,
			OwnerID:      seat.ownerid,
			GameServerID: seat.serverid,
		}
		innerReq.Members = append(innerReq.Members, member)
	}

	//todo 固定路由到指定的BattleServer
	go RandSendMessageTo("battleserver", uint32(tableid), uint32(proxymsg.ProxyMessageType_PMT_MS_BS_ALLOCBATTLEROOM), innerReq)
}

func UpdateTableManager(now *time.Time) {
	for _, table := range TableManager {
		(*table).update(now)
	}
}

func DeleteTable(tableid int32) {
	log.Debug("DeleteTable TableID %v", tableid)
	delete(TableManager, tableid)
}

func deleteTableSeatInfo(tableid int32) {
	table, ok := TableManager[tableid]
	if ok {
		for _, seat := range table.seats {
			delete(PlayerTableIDMap, seat.charid)
		}
	}
}

func TeamOperate(charid uint32, req *clientmsg.Transfer_Team_Operate) {

	allready := true

	tableid, ok := PlayerTableIDMap[charid]
	if ok {
		table, ok := TableManager[tableid]
		if ok {
			for _, seat := range table.seats {
				if (*seat).charid == (*req).CharID {
					if (*req).Action == clientmsg.TeamOperateActionType_TOA_CHOOSE {
						(*seat).chartype = (*req).CharType
					}
					if (*req).Action == clientmsg.TeamOperateActionType_TOA_SETTLE {
						(*seat).status = SEAT_READY
					}
				}

				if (*seat).status != SEAT_READY {
					allready = false
				}
			}

			//都准备好了就进入锁定倒计时阶段
			if allready {
				(*table).checktime = time.Now().Unix()
				changeTableStatus(table, MATCH_CHARTYPE_FIXED)
			}
		}
	}
}

func JoinTable(charid uint32, charname string, matchmode int32, serverid int32, servertype string) {

	var createnew = true
	for i, table := range TableManager {
		if len((*table).seats) < int((*table).modeplayercnt) {
			seat := &Seat{
				charid:     charid,
				jointime:   time.Now().Unix(),
				serverid:   serverid,
				servertype: servertype,
				chartype:   0,
				ownerid:    0,
				status:     SEAT_NONE,
				charname:   charname,
				teamid:     int32(len((*table).seats) % 2),
			}
			TableManager[i].seats = append(TableManager[i].seats, seat)
			PlayerTableIDMap[charid] = i

			log.Debug("JoinTable TableID %v CharID %v", i, charid)

			createnew = false
			break
		}
	}
	if createnew {
		allocTableID()

		r := gamedata.CSVMatchMode.Index(matchmode)
		if r == nil {
			log.Error("JoinTable CSVMatchMode Not Found %v ", matchmode)
			return
		}
		row := r.(*cfg.MatchMode)

		table := &Table{
			tableid:    tableid,
			createtime: time.Now().Unix(),
			checktime:  0,
			matchmode:  matchmode,
			seats: []*Seat{
				&Seat{
					charid:     charid,
					jointime:   time.Now().Unix(),
					serverid:   serverid,
					servertype: servertype,
					chartype:   0,
					ownerid:    0,
					status:     SEAT_NONE,
					charname:   charname,
					teamid:     0,
				},
			},
			status:        MATCH_CONTINUE,
			modeplayercnt: int32(row.PlayerCnt),
		}
		TableManager[tableid] = table
		PlayerTableIDMap[charid] = tableid

		log.Debug("JoinTable CreateTableID %v CharID %v", tableid, charid)
	}
}

func LeaveTable(charid uint32, matchmode int32) {
	tableid, ok := PlayerTableIDMap[charid]
	if ok {
		table, ok := TableManager[tableid]
		if ok {
			for i, seat := range table.seats {
				if (*seat).charid == charid {
					TableManager[tableid].seats = append(table.seats[0:i], table.seats[i+1:]...)

					log.Debug("LeaveTable TableID %v CharID %v RestCount %v", tableid, charid, len(table.seats))
				}
			}
		} else {
			log.Error("LeaveTable TableID %v Not Exist CharID %v", tableid, charid)
		}

		delete(PlayerTableIDMap, charid)
	} else {
		log.Error("LeaveTable CharID %v Not Exist", charid)
	}
}

func ClearTable(rlt *proxymsg.Proxy_BS_MS_AllocBattleRoom) {
	table, ok := TableManager[rlt.Matchtableid]
	if ok {
		msg := &clientmsg.Rlt_NotifyBattleAddress{
			RoomID:     rlt.Battleroomid,
			BattleAddr: rlt.Connectaddr,
			BattleKey:  rlt.Battleroomkey,
		}

		for _, seat := range table.seats {
			if seat.ownerid == 0 {
				log.Debug("NotifyConnectBS CharID %v BSID %v RoomID %v", (*seat).charid, rlt.Battleserverid, rlt.Battleroomid)
				go SendMessageTo((*seat).serverid, (*seat).servertype, (*seat).charid, uint32(proxymsg.ProxyMessageType_PMT_MS_GS_BEGIN_BATTLE), msg)
			}
		}

		changeTableStatus(table, MATCH_FINISH)
	} else {
		log.Error("ClearTable TableID %v Not Found , TableCount %v", tableid, len(TableManager))
	}
}

func FormatTableInfo(tableid int32) string {
	table, ok := TableManager[tableid]
	if ok {
		return fmt.Sprintf("TableID:%v\tCTime:%v\tStatus:%v\tSeatCnt:%v", (*table).tableid, (*table).createtime, (*table).status, len((*table).seats))
	}
	return ""
}

func FormatSeatInfo(tableid int32) string {
	output := fmt.Sprintf("TableID:%v", tableid)
	table, ok := TableManager[tableid]
	if ok {
		for _, seat := range (*table).seats {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tJoinTime:%v\tServerID:%v\tServerType:%v", (*seat).charid, (*seat).jointime, (*seat).serverid, (*seat).servertype)}, "\r\n")
		}
	}
	return output
}
