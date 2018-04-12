package internal

import (
	"server/base"
	"server/conf"
	"server/game"
	"time"

	"github.com/ciaos/leaf/log"
	"github.com/ciaos/leaf/module"
	"github.com/ciaos/leaf/timer"
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

func (m *Module) rotateLog() {
	// cron expr
	d := timer.NewDispatcher(10)
	cronExpr, err := timer.NewCronExpr("0 0 0 * * *")
	if err != nil {
		return
	}
	d.CronFunc(cronExpr, func() {
		log.Rotate()
	})

	go func(chantimer chan *timer.Timer) {
		for {
			(<-chantimer).Cb()
		}
	}(d.ChanTimer)
}

func (m *Module) Run(closeSig chan bool) {

	m.rotateLog()

	for {
		select {
		case <-closeSig:
			return
		case <-time.After(time.Duration(conf.Server.TickInterval) * time.Millisecond):
			game.ChanRPC.Go("TickFrame", time.Now())
		}
	}
}
