package internal

import (
	"fmt"
	"server/conf"
	"server/msg/proxymsg"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/log"
)

var p *Proxy

type Proxy struct {
	conn redis.Conn
}

func initRedisConnection() {
	p = new(Proxy)
	p.conn, _ = redis.Dial("tcp", conf.Server.RedisHost)
	p.conn.Do("auth", conf.Server.RedisPassWord)
}

func closeRedisConnection() {
	p.conn.Close()
}

func SendMessageTo(toid int32, toserver string, charid uint64, msgid uint32, msgdata interface{}) {

	//EncodeMsgData
	msgbuff, err := proto.Marshal(msgdata.(proto.Message))
	if err != nil {
		log.Error("protobuf Marsha1 error")
		return
	}

	iMsg := &proxymsg.InternalMessage{
		Fromid:   proto.Int32(int32(conf.Server.ServerID)),
		Fromtype: proto.String(conf.Server.ServerType),
		Toid:     proto.Int32(toid),
		Totype:   proto.String(toserver),
		Charid:   proto.Uint64(charid),
		Msgid:    proto.Uint32(msgid),
		Msgdata:  msgbuff,
	}

	//SendToRedis
	queueName := fmt.Sprintf("queue_%v_%v", toserver, toid)
	msgbuff, err = proto.Marshal(iMsg)
	if err != nil {
		log.Error("SendMessageTo Marshal Error %v", msgid)
		return
	}

	skeleton.Go(func() {
		initRedisConnection()
		_, err = redis.DoWithTimeout(p.conn, 1*time.Second, "PUBLISH", queueName, msgbuff)
		if err != nil {
			log.Error("DoWithTimeout Error")
			return
		}
		closeRedisConnection()
	}, func() {
	})
}
