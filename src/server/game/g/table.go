package g

import (
	"server/conf"
	"server/msg/proxymsg"
	"time"

	"github.com/name5566/leaf/log"

	"github.com/golang/protobuf/proto"
)

const (
	MATCH_OK        = 1
	MATCH_CONTINUE  = 2
	MATCH_TIMEOUT   = 3
	MATCH_ALLOCROOM = 4
	MATCH_EMPTY     = 5
	MATCH_FINISH    = 6

	MATCH_OK_COUNT = 2
)

type Seat struct {
	charid   string
	jointime int64
}

//for match server
type Table struct {
	seats      []*Seat
	createtime int64
	matchmode  int32
	status     int32
	tableid    int32
}

var TableManager = make(map[int32]*Table)
var PlayerTableIDMap = make(map[string]int32)
var tableid int32

func InitTableManager() {
	tableid = 0
}

func (table *Table) update(now *time.Time) int32 {
	if (*now).Unix()-(*table).createtime > MATCH_TIMEOUT {
		log.Debug("tableid %v timeout createtime %v now %v", (*table).tableid, (*table).createtime, (*now).Unix())
		return MATCH_TIMEOUT
	}
	if len((*table).seats) >= MATCH_OK_COUNT {
		return MATCH_OK
	}
	if len((*table).seats) <= 0 {
		log.Debug("tableid %v empty", (*table).tableid)
		return MATCH_EMPTY
	}
	return MATCH_CONTINUE
}

func allocBattleRoom(tableid int32) {

	innerReq := &proxymsg.Proxy_MS_BS_AllocBattleRoom{
		Matchroomid: proto.Int32(tableid),
		Matchmode:   proto.Int32(TableManager[tableid].matchmode),
	}

	//todo 固定路由到指定的BattleServer
	if len(conf.Server.BattleServerList) > 0 {
		battleServer := conf.Server.BattleServerList[0]
		log.Debug("Alloc BattleRoom For Table %v", tableid)

		SendMessageTo(int32(battleServer.ServerID), battleServer.ServerType, "", uint32(proxymsg.ProxyMessageType_PMT_MS_BS_ALLOCBATTLEROOM), &innerReq)
	}
}

func UpdateTableManager(now *time.Time) {
	log.Debug("UpdateTableManager %v", len(TableManager))
	for i, table := range TableManager {
		(*table).status = (*table).update(now)
		if (*table).status == MATCH_OK || (*table).status == MATCH_TIMEOUT {

			(*table).status = MATCH_ALLOCROOM
			allocBattleRoom(i)
		}
		if (*table).status == MATCH_EMPTY {
			delete(TableManager, i)
		}
	}
}

func JoinTable(charid string, matchmode int32) {

	var createnew = true
	for i, table := range TableManager {
		if len((*table).seats) < MATCH_OK_COUNT {
			seat := &Seat{
				charid:   charid,
				jointime: time.Now().Unix(),
			}
			TableManager[i].seats = append(TableManager[i].seats, seat)
			PlayerTableIDMap[charid] = i

			log.Debug("JoinTable %v %v", i, charid)

			createnew = false
			break
		}
	}
	if createnew {
		tableid += 1
		if tableid > MAX_TABLE_COUNT {
			tableid = 0
		}

		table := &Table{
			tableid:    tableid,
			createtime: time.Now().Unix(),
			matchmode:  matchmode,
			seats: []*Seat{
				&Seat{
					charid:   charid,
					jointime: time.Now().Unix(),
				},
			},
			status: MATCH_CONTINUE,
		}
		TableManager[tableid] = table
		PlayerTableIDMap[charid] = tableid

		log.Debug("CreateNewTable %v %v", tableid, charid)
	}
}

func LeaveTable(charid string, matchmode int32) {
	tableid, ok := PlayerTableIDMap[charid]
	if ok {
		for i, seat := range TableManager[tableid].seats {
			if (*seat).charid == charid {
				ii := i + 1
				TableManager[tableid].seats = append(TableManager[tableid].seats[0:i], TableManager[tableid].seats[ii:]...)

				log.Debug("LeaveTable %v TableID %v Count %v", charid, tableid, len(TableManager[tableid].seats))
			}
		}

		delete(PlayerTableIDMap, charid)
	}
}
