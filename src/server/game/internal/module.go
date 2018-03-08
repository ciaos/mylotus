package internal

import (
	"server/base"
	"server/game/g"

	"github.com/name5566/leaf/module"
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
	g.InitTableManager()
	g.InitRoomManager()
	g.InitRedisConnection()

}

func (m *Module) OnDestroy() {
	g.UninitMongoConnection()
	g.UninitRedisConnection()
}
