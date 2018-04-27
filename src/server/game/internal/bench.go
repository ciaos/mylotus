package internal

import (
	"fmt"
	"server/conf"
	"server/gamedata"
	"server/gamedata/cfg"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"strings"
	//	"sync"
	"time"

	"github.com/ciaos/leaf/log"
)

const (
	BENCH_OK         = "bench_ok"
	BENCH_WAIT       = "bench_wait"       //开黑中
	BENCH_TIMEOUT    = "bench_timeout"    //
	BENCH_EMPTY      = "bench_empty"      //bench已无人
	BENCH_FINISH     = "bench_finish"     //
	BENCH_ERROR      = "bench_error"
	BENCH_END        = "bench_end"
)

type Unit struct {
	charid     uint32
	charname   string
	jointime   time.Time
	serverid   int32
	servertype string
}

//for match server
type Bench struct {
	units       []*Unit
	createtime  time.Time
	checktime   time.Time
	matchmode   int32
	mapid       int32
	status      string
	benchid     int32
	maxunitcnt  int32
	maxwaittime int64
}

var BenchManager = make(map[int32]*Bench, 128)
var PlayerBenchIDMap = make(map[uint32]int32, 1024)
var g_benchid int32

func InitBenchManager() {
	g_benchid = 0
}

func UninitBenchManager() {
	for benchid, bench := range BenchManager {
		bench.notifyResultToBench(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		bench.units = append([]*Unit{})
		delete(BenchManager, benchid)
	}
	for charid := range PlayerBenchIDMap {
		delete(PlayerBenchIDMap, charid)
	}
}

func UpdateBenchManager(now *time.Time) {
	for _, bench := range BenchManager {
		(*bench).update(now)
	}
}

func getBenchByCharID(charid uint32, nolog bool) *Bench {
	benchid, ok := PlayerBenchIDMap[charid]
	if ok {
		bench, ok := BenchManager[benchid]
		if ok {
			return bench
		} else {
			delete(PlayerBenchIDMap, charid)
		}
	}
	if nolog == false {
		log.Error("getBenchByCharID nil Charid %v", charid)
	}
	return nil
}

func getBenchByBenchID(benchid int32) *Bench {
	bench, ok := BenchManager[benchid]
	if ok {
		return bench
	} 
	log.Error("getBenchByBenchID nil Benchid %v", benchid)
	return nil
}

func (bench *Bench) update(now *time.Time) {

	if (*bench).status == BENCH_WAIT {
		//匹配超时
		if (*now).Unix()-(*bench).checktime.Unix() > bench.maxwaittime {
			log.Debug("Benchid %v WaitTimeout Createtime %v Now %v", bench.benchid, bench.createtime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			bench.changeBenchStatus(BENCH_TIMEOUT)
			return
		}

		if len(bench.units) <= 0 {
			bench.changeBenchStatus(BENCH_EMPTY)
		}
	} else if (*bench).status == BENCH_FINISH {
		if (*now).Unix()-(*bench).checktime.Unix() > 5 { //超时，解散
			log.Debug("Benchid %v Finish TimeOut checktime %v Now %v", (*bench).benchid, (*bench).checktime.Format(TIME_FORMAT), (*now).Format(TIME_FORMAT))
			bench.changeBenchStatus(BENCH_END)
		}
	}
}

func allocBenchID() {
	g_benchid += 1
	if g_benchid > MAX_BENCH_COUNT {
		g_benchid = 1
	}
}

func (bench *Bench)deleteBench() {
	log.Debug("DeleteBench BenchID %v", bench.benchid)
	delete(BenchManager, bench.benchid)
}

func (bench *Bench) deleteBenchUnit() {
	for _, unit := range bench.units {
		delete(PlayerBenchIDMap, unit.charid)
	}
	bench.units = append([]*Unit{}) //clear seats
}

func (bench *Bench) changeBenchStatus(status string) {
	(*bench).status = status
	bench.checktime = time.Now()
	log.Debug("changeBenchStatus Bench %v Status %v", bench.benchid, bench.status)

	if (*bench).status == BENCH_ERROR {
		//notify all member error
		bench.notifyResultToBench(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		bench.changeBenchStatus(BENCH_FINISH)
	} else if (*bench).status == BENCH_EMPTY {
		bench.deleteBench()
	} else if (*bench).status == BENCH_TIMEOUT {
		bench.notifyResultToBench(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		bench.changeBenchStatus(BENCH_FINISH)
	} else if (*bench).status == BENCH_END {
		bench.deleteBenchUnit()
		bench.deleteBench()
	}
}

func (bench *Bench) broadcast(msgid proxymsg.ProxyMessageType, msgdata interface{}) {
	for _, unit := range bench.units {
		SendMessageTo(unit.serverid, unit.servertype, unit.charid, msgid, msgdata)
	}
}

func (bench *Bench) notifyResultToBench(retcode clientmsg.Type_GameRetCode) {
	rsp := &clientmsg.Rlt_MakeTeamOperate{
		RetCode: retcode,
		Mode:          clientmsg.MatchModeType(bench.matchmode),
		MapID:         bench.mapid,
	}

	if retcode == clientmsg.Type_GameRetCode_GRC_BENCH_INFO {
		for _, unit := range bench.units {
			member := &clientmsg.Rlt_MakeTeamOperate_TeamMemberInfo {
				CharID : unit.charid,
				CharName : unit.charname, 
			}
			rsp.Members = append(rsp.Members, member)
		}
	}

	bench.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
}

func createBench(charid uint32, charname string, matchmode int32, mapid int32, serverid int32, servertype string) {
	allocBenchID()

	_, ok := BenchManager[g_benchid]
	if ok {
		log.Error("BenchID %v Is Using Current BenchCnt %v", g_benchid, len(BenchManager))
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_BENCH_ERROR,
		}
		SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
		return
	}

	r := gamedata.CSVBenchConfig.Index(matchmode)
	if r == nil {
		log.Error("CSVBenchConfig ModeID %v Not Found", matchmode)
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_BENCH_ERROR,
		}
		SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
		return
	}
	row := r.(*cfg.BenchConfig)

	bench := &Bench{
		benchid:    g_benchid,
		createtime: time.Now(),
		checktime:  time.Now(),
		matchmode:  matchmode,
		mapid:      mapid,
		units: []*Unit{
			&Unit{
				charid:     charid,
				jointime:   time.Now(),
				serverid:   serverid,
				servertype: servertype,
				charname:   charname,
			},
		},
		status:      BENCH_WAIT,
		maxunitcnt:  row.MaxUnitCnt,
		maxwaittime: row.MaxWaitTime,
	}
	BenchManager[bench.benchid] = bench
	PlayerBenchIDMap[charid] = bench.benchid

	log.Release("CreateBench BenchID %v CharID %v", bench.benchid, charid)

	rsp := &clientmsg.Rlt_MakeTeamOperate{
		RetCode: clientmsg.Type_GameRetCode_GRC_OK,
		Action : clientmsg.MakeTeamOperateType_MTOT_CREATE,
	}
	SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
}

func (bench *Bench)inviteBench(charid uint32, targetid uint32, targetgsid int32) {
	log.Debug("InviteBench %v %v to BenchID %v",charid, targetid, bench.benchid)
	rsp := &clientmsg.Rlt_MakeTeamOperate{
		RetCode:       clientmsg.Type_GameRetCode_GRC_OK,
		Action:        clientmsg.MakeTeamOperateType_MTOT_INVITE,
		Mode:          clientmsg.MatchModeType(bench.matchmode),
		MapID:         bench.mapid,
		TargetID:      targetid,
		BenchID:       bench.benchid,
		MatchServerID: int32(conf.Server.ServerID),
		InviterID:     charid,
	}
	SendMessageTo(targetgsid, conf.Server.GameServerRename, targetid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
}

func (bench *Bench)acceptBench(charid uint32, charname string, benchid int32, serverid int32, servertype string) {
	log.Debug("AcceptBench %v %v",charid, benchid)
	if len(bench.units) >= int(bench.maxunitcnt) {
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_BENCH_FULL,
		}
		SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
	} else {
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_OK,
			Action: clientmsg.MakeTeamOperateType_MTOT_ACCEPT,
		}
		SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)

		unit := &Unit{
			charid:     charid,
			jointime:   time.Now(),
			serverid:   serverid,
			servertype: servertype,
			charname:   charname,
		}
		bench.units = append(bench.units, unit)
		PlayerBenchIDMap[charid] = bench.benchid
		bench.notifyResultToBench(clientmsg.Type_GameRetCode_GRC_BENCH_INFO)
	}
}

func (bench *Bench)startMatch(charid uint32) {
	if bench.status != BENCH_WAIT || len(bench.units) <= 0 {
		return
	}

	if bench.units[0].charid != charid {
		log.Error("StartMatch Invalid CharID %v != %v", bench.units[0].charid, charid)
		return
	}

	if joinTableFromBench(bench) == true {

		log.Release("StartMatch BenchID %v CharID %v", bench.benchid, charid)
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_OK,
			Action : clientmsg.MakeTeamOperateType_MTOT_START_MATCH,
		}
		bench.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
		bench.changeBenchStatus(BENCH_FINISH)
	} else {
		bench.changeBenchStatus(BENCH_ERROR)
	}
}

func (bench *Bench)kickBench(charid uint32, targetid uint32) {
	if bench.status != BENCH_WAIT || len(bench.units) <= 1 { //match already done
		return
	}

	if bench.units[0].charid != charid {
		log.Error("KickBench Invalid CharID %v != %v kick %v", bench.units[0].charid, charid, targetid)
		return
	}

	rsp := &clientmsg.Rlt_MakeTeamOperate{
		RetCode:  clientmsg.Type_GameRetCode_GRC_OK,
		Action:   clientmsg.MakeTeamOperateType_MTOT_KICK,
		TargetID: targetid,
	}
	bench.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
	for i, unit := range bench.units {
		if unit.charid == targetid {
			msg := &proxymsg.Proxy_MS_GS_Delete{
				Reason : 2,
			}
			SendMessageTo(unit.serverid, conf.Server.GameServerRename, targetid, proxymsg.ProxyMessageType_PMT_MS_GS_DELETE, msg)

			bench.units = append(bench.units[0:i], bench.units[i+1:]...)

			log.Debug("KickBench BenchID %v CharID %v RestCount %v", bench.benchid, charid, len(bench.units))
			break
		}
	}
}

func (bench *Bench)leaveBench(charid uint32, matchmode int32) {
	if bench.status != BENCH_WAIT { //match already done
		return
	}

	rsp := &clientmsg.Rlt_MakeTeamOperate{
		RetCode:  clientmsg.Type_GameRetCode_GRC_OK,
		Action:   clientmsg.MakeTeamOperateType_MTOT_LEAVE,
		TargetID: charid,
	}
	bench.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)

	gsid := int32(0)
	if len(bench.units) == 1 {
		gsid = bench.units[0].serverid
		bench.units = append([]*Unit{})
		log.Debug("LeaveBench BenchID %v CharID %v Empty", bench.benchid, charid)
	} else {
		for i, unit := range bench.units {
			if unit.charid == charid {
				gsid = unit.serverid
				bench.units = append(bench.units[0:i], bench.units[i+1:]...)

				log.Debug("LeaveBench BenchID %v CharID %v RestCount %v", bench.benchid, charid, len(bench.units))
				break
			}
		}
	}

	delete(PlayerBenchIDMap, charid)
	if gsid > 0 {
		rsp := &proxymsg.Proxy_MS_GS_Delete{
			Reason : 3,
		}
		SendMessageTo(gsid, conf.Server.GameServerRename, charid, proxymsg.ProxyMessageType_PMT_MS_GS_DELETE, rsp)
	}
}

func (bench *Bench)FormatBenchInfo() string {
	return fmt.Sprintf("BenchID:%v\tMatchMode:%v\tMapID:%v\tMaxUnitCount:%v\tCTime:%v\tStatus:%v\tUnitCnt:%v", bench.benchid, bench.matchmode, bench.mapid, bench.maxunitcnt, bench.createtime.Format(TIME_FORMAT), bench.status, len(bench.units))
}

func (bench *Bench)FormatUnitInfo() string {
	output := bench.FormatBenchInfo()
	for _, unit := range bench.units {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%10v\tJoinTime:%v\tGSID:%v\tCharName:%v", unit.charid, unit.jointime.Format(TIME_FORMAT), unit.serverid, unit.charname)}, "\r\n")
	}
	return output
}
