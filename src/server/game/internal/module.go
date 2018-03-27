package internal

import (
	"server/base"
	"server/game/g"

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

	g.InitMongoConnection()
	g.InitRedisConnection()

	g.InitTableManager()
	g.InitRoomManager()

}

func (m *Module) OnDestroy() {
	g.UninitRoomManager()
	g.UninitTableManager()

	g.UninitRedisConnection()
	g.UninitMongoConnection()
}
