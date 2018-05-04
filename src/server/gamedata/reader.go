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

var (
	CSVMatchMode   *recordfile.RecordFile
	CSVGameServer  *recordfile.RecordFile
	CSVNewPlayer   *recordfile.RecordFile
	CSVBenchConfig *recordfile.RecordFile
	CSVShopItem    *recordfile.RecordFile
)

func LoadCSV() {
	CSVMatchMode = readRf(cfg.MatchMode{})
	CSVGameServer = readRf(cfg.GameServer{})
	CSVNewPlayer = readRf(cfg.NewPlayer{})
	CSVBenchConfig = readRf(cfg.BenchConfig{})
	CSVShopItem = readRf(cfg.ShopItem{})
}
