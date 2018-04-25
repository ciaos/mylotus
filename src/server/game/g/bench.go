package g

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

func DeleteBench(benchid int32) {
	log.Debug("DeleteBench BenchID %v", benchid)
	delete(BenchManager, benchid)
}

func (bench *Bench) deleteBenchUnitInfo() {
	for _, unit := range bench.units {
		delete(PlayerBenchIDMap, unit.charid)
	}
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
		DeleteBench((*bench).benchid)
	} else if (*bench).status == BENCH_TIMEOUT {
		bench.notifyResultToBench(clientmsg.Type_GameRetCode_GRC_MATCH_ERROR)
		bench.changeBenchStatus(BENCH_FINISH)
	} else if (*bench).status == BENCH_END {
		bench.deleteBenchUnitInfo()
		bench.units = append([]*Unit{}) //clear seats
		DeleteBench(bench.benchid)
	}
}

func (bench *Bench) broadcast(msgid proxymsg.ProxyMessageType, msgdata interface{}) {
	go func() {
		for _, unit := range bench.units {
			SendMessageTo(unit.serverid, unit.servertype, unit.charid, msgid, msgdata)
		}
	}()
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

func CreateBench(charid uint32, charname string, matchmode int32, mapid int32, serverid int32, servertype string) {
	allocBenchID()

	_, ok := BenchManager[g_benchid]
	if ok {
		log.Error("BenchID %v Is Using Current BenchCnt %v", g_benchid, len(BenchManager))
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_BENCH_ERROR,
		}
		go SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
		return
	}

	r := gamedata.CSVBenchConfig.Index(matchmode)
	if r == nil {
		log.Error("CSVBenchConfig ModeID %v Not Found", matchmode)
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_BENCH_ERROR,
		}
		go SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
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

	log.Debug("CreateBenchID %v CharID %v CharName %v", bench.benchid, charid, charname)

	rsp := &clientmsg.Rlt_MakeTeamOperate{
		RetCode: clientmsg.Type_GameRetCode_GRC_OK,
		Action : clientmsg.MakeTeamOperateType_MTOT_CREATE,
	}
	go SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
}

func InviteBench(charid uint32, targetid uint32, targetgsid int32) {
	benchid, ok := PlayerBenchIDMap[charid]
	if ok {
		bench, ok := BenchManager[benchid]
		if ok {
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
			go SendMessageTo(targetgsid, conf.Server.GameServerRename, targetid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
		}
	}
}

func AcceptBench(charid uint32, charname string, benchid int32, serverid int32, servertype string) {
	bench, ok := BenchManager[benchid]
	if !ok {
		log.Error("BenchID %v NotExist CharID %v", benchid, charid)
		return
	}

	log.Debug("AcceptBench %v %v",charid, benchid)
	if len(bench.units) >= int(bench.maxunitcnt) {
		rsp := &clientmsg.Rlt_MakeTeamOperate{
			RetCode: clientmsg.Type_GameRetCode_GRC_BENCH_FULL,
		}
		go SendMessageTo(serverid, servertype, charid, proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)
	} else {
		unit := &Unit{
			charid:     charid,
			jointime:   time.Now(),
			serverid:   serverid,
			servertype: servertype,
			charname:   charname,
		}
		bench.units = append(bench.units, unit)
		bench.notifyResultToBench(clientmsg.Type_GameRetCode_GRC_BENCH_INFO)
	}
}

func StartMatch(charid uint32) {
	benchid, ok := PlayerBenchIDMap[charid]
	if ok {
		bench, ok := BenchManager[benchid]
		if ok {
			if bench.status != BENCH_WAIT || len(bench.units) <= 0 {
				return
			}

			if bench.units[0].charid != charid {
				log.Error("StartMatch Invalid CharID %v != %v", bench.units[0].charid, charid)
				return
			}

			if joinTableFromBench(bench) == true {
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
	}
}

func KickBench(charid uint32, targetid uint32) {
	benchid, ok := PlayerBenchIDMap[charid]
	if ok {
		bench, ok := BenchManager[benchid]
		if ok {
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
					bench.units = append(bench.units[0:i], bench.units[i+1:]...)

					log.Debug("KickBench BenchID %v CharID %v RestCount %v", benchid, charid, len(bench.units))
					break
				}
			}
		} else {
			log.Error("KickBench BenchID %v Not Exist CharID %v", benchid, charid)
		}
	} else {
		log.Error("KickBench CharID %v Not Exist", charid)
	}
}

func LeaveBench(charid uint32, matchmode int32) {
	benchid, ok := PlayerBenchIDMap[charid]
	if ok {
		bench, ok := BenchManager[benchid]
		if ok {
			if bench.status != BENCH_WAIT { //match already done
				return
			}

			rsp := &clientmsg.Rlt_MakeTeamOperate{
				RetCode:  clientmsg.Type_GameRetCode_GRC_OK,
				Action:   clientmsg.MakeTeamOperateType_MTOT_LEAVE,
				TargetID: charid,
			}
			bench.broadcast(proxymsg.ProxyMessageType_PMT_MS_GS_MAKE_TEAM_OPERATE, rsp)

			if len(bench.units) <= 1 {
				bench.units = append([]*Unit{})
				log.Debug("LeaveBench BenchID %v CharID %v Empty", benchid, charid)
			} else {
				for i, unit := range bench.units {
					if unit.charid == charid {
						bench.units = append(bench.units[0:i], bench.units[i+1:]...)

						log.Debug("LeaveBench BenchID %v CharID %v RestCount %v", benchid, charid, len(bench.units))
						break
					}
				}
			}
		} else {
			log.Error("LeaveBench BenchID %v Not Exist CharID %v", benchid, charid)
		}

		delete(PlayerBenchIDMap, charid)
	} 
}

func FormatBenchInfo(benchid int32) string {
	bench, ok := BenchManager[benchid]
	if ok {
		return fmt.Sprintf("BenchID:%v\tMatchMode:%v\tMapID:%v\tMaxUnitCount:%v\tCTime:%v\tStatus:%v\tUnitCnt:%v", bench.benchid, bench.matchmode, bench.mapid, bench.maxunitcnt, bench.createtime.Format(TIME_FORMAT), bench.status, len(bench.units))
	}
	return ""
}

func FormatUnitInfo(benchid int32) string {
	output := FormatBenchInfo(benchid)
	bench, ok := BenchManager[benchid]
	if ok {
		for _, unit := range bench.units {
			output = strings.Join([]string{output, fmt.Sprintf("CharID:%10v\tJoinTime:%v\tGSID:%v\tCharName:%v", unit.charid, unit.jointime.Format(TIME_FORMAT), unit.serverid, unit.charname)}, "\r\n")
		}
	}
	return output
}
