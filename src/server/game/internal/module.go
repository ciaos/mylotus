package internal

import (
	"server/base"
	"server/gamedata"

	"github.com/ciaos/leaf/module"
)

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton

	gamedata.LoadCSV()

	InitMongoConnection()
	InitRedisConnection()

	InitBenchManager()
	InitTableManager()
	InitRoomManager()
}

func (m *Module) OnDestroy() {
	UninitRoomManager()
	UninitTableManager()
	UninitBenchManager()
	UninitGamePlayerManager()

	UninitRedisConnection()
	UninitMongoConnection()
}
