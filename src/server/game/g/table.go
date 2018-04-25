package g

import (
	"fmt"
	"math"
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
	MATCH_CONFIRM           = "match_confirm"           //匹配确认
	MATCH_CLEAR_BADGUY      = "match_clear_badguy"      //清空未点击确认的玩家继续匹配
	MATCH_CHARTYPE_CHOOSING = "match_chartype_choosing" //选择角色中
	MATCH_SOMEBODY_REJECT   = "match_somebody_reject"   //有人拒绝了
	MATCH_CHARTYPE_FIXED    = "match_chartype_fixed"    //角色确定
	MATCH_BEGIN_ALLOCROOM   = "match_begin_allocroom"   //开始申请房间
	MATCH_ALLOCROOM         = "match_allocroom"         //申请房间中
	MATCH_EMPTY             = "match_empty"             //桌子已无人
	MATCH_FINISH            = "match_finish"            //
	MATCH_ERROR             = "match_error"
	MATCH_END               = "match_end"

	SEAT_NONE    = 0
	SEAT_CONFIRM = 1
	SEAT_READY   = 2
	SEAT_REJECT  = 3
)

type Seat struct {
	charid     uint32
	charname   string
	jointime   time.Time
	serverid   int32
	servertype string
	chartype   int32
	ownerid    uint32
	teamid     int32
	status     int32
}

//for match server
type Table struct {
	seats         []*Seat
	createtime    time.Time
	checktime     time.Time
	matchmode     int32
	mapid         int32
	status        string
	tableid       int32
	modeplayercnt int32
}

var TableManager = make(map[int32]*Table, 128)
var PlayerTableIDMap = make(map[uint32]int32, 1024)
var g_tableid int32

//var mTableID *sync.Mutex

func InitTableManager() {
	g_tableid = 0

	//	mTableID = new(sync.Mutex)
}

func UninitTableManager() {
	for tableid, table := range TableManager {
		table.notifyMatchResultToTable(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		table.seats = append([]*Seat{})
		delete(TableManager, tableid)
	}
	for charid := range PlayerTableIDMap {
		delete(PlayerTableIDMap, charid)
	}
}

func allocTableID() {
	//mTableID.Lock()
	//defer mTableID.Unlock()
	g_tableid += 1
	if g_tableid > MAX_TABLE_COUNT {
		g_tableid = 1
	}
}

func (table *Table) getTeamID(teamCnt int) (int32, int) {

	teamIDCnt := make([]int, teamCnt)
	for _, seat := range table.seats {
		teamIDCnt[seat.teamid-1]++
	}

	teamid := 1
	minMemberCnt := math.MaxUint32
	for i, v := range teamIDCnt {
		if v < minMemberCnt {
			teamid = i + 1
			minMemberCnt = v
		}
	}

	return int32(teamid), minMemberCnt
}

func (table *Table) fillRobotToTable() bool {
	r := gamedata.CSVMatchMode.Index((*table).matchmode)
	if r == nil {
		return false
	}
	row := r.(*cfg.MatchMode)

	if len((*table).seats) == 0 {
		return false
	}

	robotnum := row.PlayerCnt * row.TeamCnt - len((*table).seats)
	ownerid := (*table).seats[0].charid

	i := 1
	for i <= robotnum {

		charid := 1000000000 + uint32(rand.Intn(100000000))
		teamid, _ := table.getTeamID(row.TeamCnt)
		seat := &Seat{
			charid:     charid,
			jointime:   time.Now(),
			serverid:   0,
			servertype: "",
			charname:   strconv.Itoa(int(charid)),
			chartype:   0,
			ownerid:    ownerid,
			status:     SEAT_NONE,
			teamid:     teamid,
		}
		(*table).seats = append((*table).seats, seat)
		log.Debug("fillRobotToTable RobotID %v OwnerID %v", (*seat).charid, (*seat).ownerid)
		i++
	}
	return true
}

func (table *Table) autoChooseToTable() {
	for _, seat := range table.seats {
		if (*seat).chartype == 0 {

			(*seat).chartype = 1001

			msg := &clientmsg.Transfer_Team_Operate{
				Action:   clientmsg.TeamOperateActionType_TOA_SETTLE,
				CharID:   seat.charid,
				CharType: (*seat).chartype,
			}
			table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_CHOOSE_OPERATE, msg)
			log.Debug("autoChooseToTable Table %v CharID %v CharName %v CharType %v", table.tableid, seat.charid, seat.charname, seat.chartype)
		}
	}

	r := gamedata.CSVMatchMode.Index((*table).matchmode)
	row := r.(*cfg.MatchMode)
	rsp := &clientmsg.Rlt_Match{
		RetCode:       clientmsg.Type_GameRetCode_GRC_MATCH_ALL_FIXED,
		WaitUntilTime: time.Now().Unix() + int64(row.FixedWaitTimeSec),
	}
	table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, rsp)
}

func (table *Table) broadcast(msgid proxymsg.ProxyMessageType, msgdata interface{}) {
	go func() {
		for _, seat := range table.seats {
			if seat.ownerid == 0 {
				SendMessageTo((*seat).serverid, (*seat).servertype, (*seat).charid, msgid, msgdata)
			}
		}
	}()
}

func (table *Table) notifyMatchResultToTable(retcode clientmsg.Type_GameRetCode) {

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
			member.Status = clientmsg.MemberStatus(seat.status)

			msg.Members = append(msg.Members, member)
		}

		r := gamedata.CSVMatchMode.Index((*table).matchmode)
		row := r.(*cfg.MatchMode)
		msg.WaitUntilTime = time.Now().Unix() + int64(row.ConfirmTimeOutSec)
	}

	table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, msg)
}

func (table *Table) kickBadGuy() {
	//循环删除没有确定和拒绝的玩家
reloop:
	for i, seat := range (*table).seats { //kick badguy and robot
		if seat.status != SEAT_CONFIRM {
			if seat.ownerid == 0 {
				go SendMessageTo(seat.serverid, seat.servertype, seat.charid, proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, &clientmsg.Rlt_Match{RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR})
				log.Debug("Kick BadGuy TableID %v CharID %v RestCount %v", (*table).tableid, seat.charid, len((*table).seats))
				delete(PlayerTableIDMap, seat.charid)
			}
			(*table).seats = append(table.seats[0:i], table.seats[i+1:]...)
			goto reloop
		} else {
			r := gamedata.CSVMatchMode.Index((*table).matchmode)
			row := r.(*cfg.MatchMode)
			rsp := &clientmsg.Rlt_Match{
				RetCode:       clientmsg.Type_GameRetCode_GRC_MATCH_CONTINUE,
				WaitUntilTime: time.Now().Unix() + int64(row.MatchTimeOutSec),
			}
			go SendMessageTo(seat.serverid, seat.servertype, seat.charid, proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, rsp)
		}
	}

	table.changeTableStatus(MATCH_CONTINUE)

	//重置状态
	for _, seat := range (*table).seats {
		seat.status = SEAT_NONE
	}
}

func ReconnectTable(charid uint32, pmsg *proxymsg.InternalMessage) {
	rsp := proxymsg.Proxy_MS_GS_Reconnect{
		Ok: false,
	}

	SendMessageTo(pmsg.Fromid, pmsg.Fromtype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_RECONNECT, rsp)
}

func (table *Table) changeTableStatus(status string) {
	(*table).status = status
	table.checktime = time.Now()
	log.Debug("changeTableStatus Table %v Status %v", (*table).tableid, (*table).status)

	if (*table).status == MATCH_ERROR {
		//notify all member error
		table.notifyMatchResultToTable(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		table.changeTableStatus(MATCH_FINISH)
	} else if (*table).status == MATCH_EMPTY {
		DeleteTable((*table).tableid)
	} else if (*table).status == MATCH_OK {
		//notify all member to choose
		table.notifyMatchResultToTable(clientmsg.Type_GameRetCode_GRC_MATCH_OK)
		table.changeTableStatus(MATCH_CONFIRM)
	} else if (*table).status == MATCH_TIMEOUT {
		if table.matchmode == int32(clientmsg.MatchModeType_MMT_AI) {
			//fill with robot and notify all member to choose
			if table.fillRobotToTable() {
				table.notifyMatchResultToTable(clientmsg.Type_GameRetCode_GRC_MATCH_OK)
				table.changeTableStatus(MATCH_CONFIRM)
			} else {
				table.changeTableStatus(MATCH_ERROR)
			}
		} else {
			table.notifyMatchResultToTable(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
			table.changeTableStatus(MATCH_FINISH)
		}
	} else if (*table).status == MATCH_CHARTYPE_FIXED {
		//notify fixed time

		//auto choose
		table.autoChooseToTable()
	} else if (*table).status == MATCH_BEGIN_ALLOCROOM {
		table.allocBattleRoom()
		table.changeTableStatus(MATCH_ALLOCROOM)
	} else if (*table).status == MATCH_END {
		table.deleteTableSeatInfo()
		table.seats = append([]*Seat{}) //clear seats
		DeleteTable(table.tableid)
	} else if (*table).status == MATCH_CLEAR_BADGUY {
		table.kickBadGuy()
	}
}

func (table *Table) update(now *time.Time) {
	r := gamedata.CSVMatchMode.Index((*table).matchmode)
	if r == nil {
		log.Error("CSVMatchMode ModeID %v Not Found", (*table).matchmode)
		table.changeTableStatus(MATCH_ERROR)
		return
	}
	row := r.(*cfg.MatchMode)

	if (*table).status == MATCH_CONTINUE {
		//匹配超时
		if (*now).Unix()-(*table).checktime.Unix() > int64(row.MatchTimeOutSec) {
			log.Debug("Tableid %v MatchTimeout Createtime %v Now %v", (*table).tableid, (*table).createtime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_TIMEOUT)
			return
		}

		if len((*table).seats) >= row.PlayerCnt * row.TeamCnt {
			table.changeTableStatus(MATCH_OK)
		} else if len((*table).seats) <= 0 {
			table.changeTableStatus(MATCH_EMPTY)
		}
	} else if (*table).status == MATCH_CONFIRM {
		if (*now).Unix()-(*table).checktime.Unix() > int64(row.ConfirmTimeOutSec) {
			log.Debug("Tableid %v ConfirmTimeout checktime %v Now %v", (*table).tableid, (*table).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_CLEAR_BADGUY)
		}
	} else if (*table).status == MATCH_SOMEBODY_REJECT {
		if (*now).Unix()-(*table).checktime.Unix() > int64(row.RejectWaitTime) {
			log.Debug("Tableid %v RejectTimeout checktime %v Now %v", (*table).tableid, (*table).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_CLEAR_BADGUY)
		}
	} else if (*table).status == MATCH_CHARTYPE_CHOOSING {
		if (*now).Unix()-(*table).checktime.Unix() > int64(row.ChooseTimeOutSec) {
			log.Debug("Tableid %v ChooseTimeout checktime %v Now %v", (*table).tableid, (*table).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_CHARTYPE_FIXED)
		}
	} else if (*table).status == MATCH_CHARTYPE_FIXED {
		if (*now).Unix()-(*table).checktime.Unix() > int64(row.FixedWaitTimeSec) {
			log.Debug("Tableid %v FixedWaitTimeout checktime %v Now %v", (*table).tableid, (*table).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_BEGIN_ALLOCROOM)
		}
	} else if (*table).status == MATCH_ALLOCROOM {
		if (*now).Unix()-(*table).checktime.Unix() > 5 { //申请房间超时，解散队伍
			log.Error("Tableid %v Allocroom TimeOut checktime %v Now %v", (*table).tableid, (*table).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_ERROR)
		}
	} else if (*table).status == MATCH_FINISH {
		if (*now).Unix()-(*table).checktime.Unix() > 5 { //房间超时，解散
			log.Debug("Tableid %v Finish TimeOut checktime %v Now %v", (*table).tableid, (*table).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			table.changeTableStatus(MATCH_END)
		}
	}
}

func (table *Table) allocBattleRoom() {

	innerReq := &proxymsg.Proxy_MS_BS_AllocBattleRoom{
		Matchtableid: table.tableid,
		Matchmode:    table.matchmode,
		Mapid:        table.mapid,
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
	go RandSendMessageTo("battleserver", uint32(table.tableid), proxymsg.ProxyMessageType_PMT_MS_BS_ALLOCBATTLEROOM, innerReq)
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

func (table *Table) deleteTableSeatInfo() {
	for _, seat := range table.seats {
		if seat.ownerid == 0 {
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
			if table.status != MATCH_CHARTYPE_CHOOSING {
				log.Error("TeamOperate CharID %v Table %v Status %v Action %v", charid, tableid, table.status, req.Action)
				return
			}

			for _, seat := range table.seats {
				if (*seat).charid == (*req).CharID {
					if (*req).Action == clientmsg.TeamOperateActionType_TOA_CHOOSE {
						(*seat).chartype = (*req).CharType
					}
					if (*req).Action == clientmsg.TeamOperateActionType_TOA_SETTLE {
						(*seat).chartype = (*req).CharType
						(*seat).status = SEAT_READY
					}
				}

				if (*seat).status != SEAT_READY {
					allready = false
				}
			}
			log.Debug("Team_Operate %v %v %v %v", charid, req.Action, req.CharID, req.CharType)
			table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_CHOOSE_OPERATE, req)

			//都准备好了就进入锁定倒计时阶段
			if allready {
				table.changeTableStatus(MATCH_CHARTYPE_FIXED)
			}
		} else {
			log.Error("TeamOperate Table Not Exist", tableid)
		}
	} else {
		log.Error("TeamOperate Error CharID %v Not In Table Action %v", charid, req.Action)
	}
}

func joinTableFromBench(bench *Bench) bool {

	r := gamedata.CSVMatchMode.Index(bench.matchmode)
	if r == nil {
		log.Error("joinTableFromBench CSVMatchMode Not Found %v ", bench.matchmode)
		return false
	}
	row := r.(*cfg.MatchMode)

	var createnew = true
	if bench.matchmode != int32(clientmsg.MatchModeType_MMT_AI) { //打AI都是创建新房间
		for i, table := range TableManager {
			if table.mapid != bench.mapid || table.matchmode != bench.matchmode {
				continue
			}

			if int((*table).modeplayercnt) - len((*table).seats) < len(bench.units) {
				continue
			}

			teamid, teamcnt := table.getTeamID(row.TeamCnt)
			if teamcnt + len(bench.units) > row.PlayerCnt {
				continue
			}

			for _, unit := range bench.units {
				seat := &Seat{
					charid:     unit.charid,
					jointime:   time.Now(),
					serverid:   unit.serverid,
					servertype: unit.servertype,
					chartype:   0,
					ownerid:    0,
					status:     SEAT_NONE,
					charname:   unit.charname,
					teamid:     teamid,
				}
				table.seats = append(table.seats, seat)
				PlayerTableIDMap[unit.charid] = i

				log.Debug("joinTableFromBench TableID %v CharID %v CharName %v", i, unit.charid, unit.charname)
				createnew = false
			}
		}
	}

	if createnew {
		allocTableID()

		_, ok := TableManager[g_tableid]
		if ok {
			log.Error("TableID %v Is Using Current TableCnt %v", g_tableid, len(TableManager))
			return false
		}

		table := &Table{
			tableid:    g_tableid,
			createtime: time.Now(),
			checktime:  time.Now(),
			matchmode:  bench.matchmode,
			mapid:      bench.mapid,
			status:        MATCH_CONTINUE,
			modeplayercnt: int32(row.PlayerCnt * row.TeamCnt),
		}
		TableManager[table.tableid] = table

		teamid, _ := table.getTeamID(row.TeamCnt)
		for _, unit := range bench.units {
			seat := &Seat{
				charid:     unit.charid,
				jointime:   time.Now(),
				serverid:   unit.serverid,
				servertype: unit.servertype,
				chartype:   0,
				ownerid:    0,
				status:     SEAT_NONE,
				charname:   unit.charname,
				teamid:     teamid,
			}
			table.seats = append(table.seats, seat)
			PlayerTableIDMap[unit.charid] = table.tableid

			log.Debug("joinTableFromBench CreateNew TableID %v CharID %v CharName %v", table.tableid, unit.charid, unit.charname)
		}
	}
	
	return true
}

func JoinTable(charid uint32, charname string, matchmode int32, mapid int32, serverid int32, servertype string) {

	//already matching
	_, ok := PlayerTableIDMap[charid]
	if ok {
		return
	}

	r := gamedata.CSVMatchMode.Index(matchmode)
	if r == nil {
		log.Error("JoinTable CSVMatchMode Not Found %v ", matchmode)
		return
	}
	row := r.(*cfg.MatchMode)

	var createnew = true
	if matchmode != int32(clientmsg.MatchModeType_MMT_AI) { //打AI都是创建新房间
		for i, table := range TableManager {
			if table.mapid != mapid || table.matchmode != matchmode {
				continue
			}

			if len((*table).seats) < int((*table).modeplayercnt) {
				teamid, _ := table.getTeamID(row.TeamCnt)
				seat := &Seat{
					charid:     charid,
					jointime:   time.Now(),
					serverid:   serverid,
					servertype: servertype,
					chartype:   0,
					ownerid:    0,
					status:     SEAT_NONE,
					charname:   charname,
					teamid:     teamid,
				}
				table.seats = append(table.seats, seat)
				PlayerTableIDMap[charid] = i

				log.Debug("JoinTable TableID %v CharID %v CharName %v", i, charid, charname)

				createnew = false
				break
			}
		}
	}
	if createnew {
		allocTableID()

		_, ok := TableManager[g_tableid]
		if ok {
			log.Error("TableID %v Is Using Current TableCnt %v", g_tableid, len(TableManager))
			rsp := &clientmsg.Rlt_Match{
				RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
			}
			go SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, rsp)
			return
		}

		table := &Table{
			tableid:    g_tableid,
			createtime: time.Now(),
			checktime:  time.Now(),
			matchmode:  matchmode,
			mapid:      mapid,
			seats: []*Seat{
				&Seat{
					charid:     charid,
					jointime:   time.Now(),
					serverid:   serverid,
					servertype: servertype,
					chartype:   0,
					ownerid:    0,
					status:     SEAT_NONE,
					charname:   charname,
					teamid:     1,
				},
			},
			status:        MATCH_CONTINUE,
			modeplayercnt: int32(row.PlayerCnt * row.TeamCnt),
		}
		TableManager[table.tableid] = table
		PlayerTableIDMap[charid] = table.tableid

		log.Debug("JoinTable CreateTableID %v CharID %v CharName %v", table.tableid, charid, charname)
	}

	rsp := &clientmsg.Rlt_Match{
		RetCode:       clientmsg.Type_GameRetCode_GRC_MATCH_START,
		WaitUntilTime: time.Now().Unix() + int64(row.MatchTimeOutSec),
	}
	go SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, rsp)
}

func LeaveTable(charid uint32, matchmode int32) {
	tableid, ok := PlayerTableIDMap[charid]
	if ok {
		table, ok := TableManager[tableid]
		if ok {
			if table.status != MATCH_CONTINUE { //match already done
				return
			}

			if len(table.seats) <= 1 {
				table.seats = append([]*Seat{})
				log.Debug("LeaveTable TableID %v CharID %v Empty", tableid, charid)
			} else {
				for i, seat := range table.seats {
					if (*seat).charid == charid {
						table.seats = append(table.seats[0:i], table.seats[i+1:]...)

						log.Debug("LeaveTable TableID %v CharID %v RestCount %v", tableid, charid, len(table.seats))
						break
					}
				}
			}
		} else {
			log.Error("LeaveTable TableID %v Not Exist CharID %v", tableid, charid)
		}

		delete(PlayerTableIDMap, charid)
	}
}

func ConfirmTable(charid uint32, matchmode int32) {
	tableid, ok := PlayerTableIDMap[charid]
	if ok {
		table, ok := TableManager[tableid]
		if ok {
			if table.status != MATCH_CONFIRM {
				log.Error("ConfirmTable CharID %v Table %v Status %v", charid, tableid, table.status)
				return
			}

			allconfirmed := true

			msg := &clientmsg.Rlt_Match{}

			for _, seat := range table.seats {
				if (*seat).charid == charid || (*seat).ownerid == charid {
					seat.status = SEAT_CONFIRM
					log.Debug("ConfirmTable TableID %v CharID %v OwnerID %v", tableid, seat.charid, seat.ownerid)

					member := &clientmsg.Rlt_Match_MemberInfo{}
					member.CharID = (*seat).charid
					member.OwnerID = (*seat).ownerid
					member.TeamID = (*seat).teamid
					member.CharName = (*seat).charname
					member.CharType = (*seat).chartype
					member.Status = clientmsg.MemberStatus(seat.status)
					msg.Members = append(msg.Members, member)
				}

				if seat.status != SEAT_CONFIRM {
					allconfirmed = false
				}
			}

			if allconfirmed {
				log.Debug("AllConfirmTable TableID %v", tableid)
				msg.RetCode = clientmsg.Type_GameRetCode_GRC_MATCH_ALL_CONFIRMED
				r := gamedata.CSVMatchMode.Index((*table).matchmode)
				row := r.(*cfg.MatchMode)
				msg.WaitUntilTime = time.Now().Unix() + int64(row.ChooseTimeOutSec)

				table.changeTableStatus(MATCH_CHARTYPE_CHOOSING)
			} else {
				msg.RetCode = clientmsg.Type_GameRetCode_GRC_MATCH_CONFIRM
			}

			table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, msg)
		} else {
			log.Error("ConfirmTable TableID %v Not Exist CharID %v", tableid, charid)
			delete(PlayerTableIDMap, charid)
		}
	} else {
		log.Error("ConfirmTable CharID %v Not Exist", charid)
	}
}

func RejectTable(charid uint32, matchmode int32) {
	tableid, ok := PlayerTableIDMap[charid]
	if ok {
		table, ok := TableManager[tableid]
		if ok {
			if table.status != MATCH_CONFIRM {
				log.Error("RejectTable CharID %v Table %v Status %v", charid, tableid, table.status)
				return
			}

			msg := &clientmsg.Rlt_Match{}

			for _, seat := range table.seats {
				if (*seat).charid == charid {
					seat.status = SEAT_REJECT
					log.Debug("ConfirmTable TableID %v CharID %v OwnerID %v", tableid, seat.charid, seat.ownerid)

					member := &clientmsg.Rlt_Match_MemberInfo{}
					member.CharID = (*seat).charid
					member.OwnerID = (*seat).ownerid
					member.TeamID = (*seat).teamid
					member.CharName = (*seat).charname
					member.CharType = (*seat).chartype
					member.Status = clientmsg.MemberStatus(seat.status)
					msg.Members = append(msg.Members, member)

					break
				}
			}

			table.changeTableStatus(MATCH_SOMEBODY_REJECT)
			msg.RetCode = clientmsg.Type_GameRetCode_GRC_MATCH_CONFIRM

			r := gamedata.CSVMatchMode.Index((*table).matchmode)
			row := r.(*cfg.MatchMode)
			msg.WaitUntilTime = time.Now().Unix() + int64(row.RejectWaitTime)

			table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MATCH_RESULT, msg)
		} else {
			log.Error("ConfirmTable TableID %v Not Exist CharID %v", tableid, charid)
			delete(PlayerTableIDMap, charid)
		}
	} else {
		log.Error("ConfirmTable CharID %v Not Exist", charid)
	}
}

func ClearTable(rlt *proxymsg.Proxy_BS_MS_AllocBattleRoom) {
	table, ok := TableManager[rlt.Matchtableid]
	if ok {
		if rlt.Retcode == 0 {
			msg := &clientmsg.Rlt_NotifyBattleAddress{
				RoomID:         rlt.Battleroomid,
				BattleAddr:     rlt.Connectaddr,
				BattleKey:      rlt.Battleroomkey,
				BattleServerID: rlt.Battleserverid,
			}

			table.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_BEGIN_BATTLE, msg)
			table.checktime = time.Now()
			table.changeTableStatus(MATCH_FINISH)
		} else {
			table.changeTableStatus(MATCH_ERROR)
		}
	} else {
		log.Error("ClearTable TableID %v Not Found , TableCount %v", rlt.Matchtableid, len(TableManager))
	}
}

func FormatTableInfo(tableid int32) string {
	table, ok := TableManager[tableid]
	if ok {
		return fmt.Sprintf("TableID:%v\tMatchMode:%v\tMapID:%v\tPlayerCount:%v\tCTime:%v\tStatus:%v\tSeatCnt:%v", (*table).tableid, (*table).matchmode, (*table).mapid, (*table).modeplayercnt, (*table).createtime.Format(TIME_FORMAT), (*table).status, len((*table).seats))
	}
	return ""
}

func FormatSeatInfo(tableid int32) string {
	output := FormatTableInfo(tableid)
	table, ok := TableManager[tableid]
	if ok {
		for _, seat := range (*table).seats {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%10v\tJoinTime:%v\tCharType:%v\tOwnerID:%v\tTeamID:%v\tStatus:%v\tGSID:%v\tCharName:%v", (*seat).charid, (*seat).jointime.Format(TIME_FORMAT), (*seat).chartype, (*seat).ownerid, (*seat).teamid, (*seat).status, (*seat).serverid, (*seat).charname)}, "\r\n")
		}
	}
	return output
}
