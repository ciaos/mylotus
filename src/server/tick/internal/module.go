package internal

import (
	"server/base"
	"server/conf"
	"server/game"
	"time"

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
}

func (m *Module) OnDestroy() {

}

func (m *Module) Run(closeSig chan bool) {
	for {
		select {
		case <-closeSig:
			return
		case <-time.After(time.Duration(conf.Server.TickInterval) * time.Millisecond):
			game.ChanRPC.Go("TickFrame", time.Now())
		}
	}
}
