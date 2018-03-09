package internal

import (
	"server/base"
	"server/conf"

	"github.com/ciaos/leaf/db/mongodb"
	"github.com/ciaos/leaf/module"
)

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
)

var Pmongo *mongodb.DialContext

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton

	Pmongo, _ = mongodb.Dial(conf.Server.MongoDBHost, 10)
}

func (m *Module) OnDestroy() {
	Pmongo.Close()
}
