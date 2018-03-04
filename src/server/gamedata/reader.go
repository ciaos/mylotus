package gamedata

import (
	"reflect"

	"server/gamedata/cfg"

	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/recordfile"
)

func readRf(st interface{}) *recordfile.RecordFile {
	rf, err := recordfile.New(st)
	if err != nil {
		log.Fatal("%v", err)
	}
	fn := reflect.TypeOf(st).Name() + ".csv"
	err = rf.Read("gamedata/csv/" + fn)
	if err != nil {
		log.Fatal("%v: %v", fn, err)
	}

	return rf
}

var RfRecord = readRf(cfg.Record{})