package gamedata

import (
	"reflect"

	"server/gamedata/cfg"

	"github.com/ciaos/leaf/log"
	"github.com/ciaos/leaf/recordfile"
)

func readRf(st interface{}) *recordfile.RecordFile {
	rf, err := recordfile.New(st)
	if err != nil {
		log.Fatal("%v", err)
	}
	rf.Comma = ','
	fn := reflect.TypeOf(st).Name() + ".csv"
	err = rf.Read("gamedata/csv/" + fn)
	if err != nil {
		log.Fatal("%v: %v", fn, err)
	}

	return rf
}

var CSVMatchMode = readRf(cfg.MatchMode{})
var CSVGameServer = readRf(cfg.GameServer{})
var CSVNewPlayer = readRf(cfg.NewPlayer{})
var CSVBenchConfig = readRf(cfg.BenchConfig{})
