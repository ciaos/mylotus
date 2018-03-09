package internal

import (
	"fmt"
	"server/base"
	"server/conf"
	"server/game"

	"github.com/ciaos/leaf/log"
	"github.com/ciaos/leaf/module"
	"github.com/garyburd/redigo/redis"
)

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
)

type Module struct {
	*module.Skeleton
	conn      redis.Conn
	queueName string
	psc       redis.PubSubConn
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	m.conn, _ = redis.Dial("tcp", conf.Server.RedisHost)
	m.conn.Do("auth", conf.Server.RedisPassWord)
	m.psc = redis.PubSubConn{m.conn}
	m.queueName = fmt.Sprintf("queue_%v_%v", conf.Server.ServerType, conf.Server.ServerID)

	go (*m).update()
}

func (m *Module) OnDestroy() {
	m.conn.Close()
}

func (m *Module) update() {
	m.psc.Subscribe(m.queueName)
	for {
		switch v := m.psc.Receive().(type) {
		case redis.Message:
			game.ChanRPC.Go("QueueMessage", v.Data)
		case redis.Subscription:
			log.Debug("SubScribe Queue:%s Channel:%s Kind:%s Count:%d", m.queueName, v.Channel, v.Kind, v.Count)
		case error:
			log.Error("SubScribe Queue Error:%s", v.Error())
		}
	}
}
