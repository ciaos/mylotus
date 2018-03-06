package g

import (
	"fmt"
	"server/conf"
	"server/msg/proxymsg"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/log"
)

var Predis *RedisProxy

type RedisProxy struct {
	conn redis.Conn
}

func InitRedisConnection() {
	Predis = new(RedisProxy)
	Predis.conn, _ = redis.Dial("tcp", conf.Server.RedisHost)
	Predis.conn.Do("auth", conf.Server.RedisPassWord)
}
func UninitRedisConnection() {
	Predis.conn.Close()
}

func SendMessageTo(toid int32, toserver string, charid string, msgid uint32, msgdata interface{}) bool {

	//EncodeMsgData
	msgbuff, err := proto.Marshal(msgdata.(proto.Message))
	if err != nil {
		log.Error("protobuf Marsha1 error %v", err)
		return false
	}

	iMsg := &proxymsg.InternalMessage{
		Fromid:   proto.Int32(int32(conf.Server.ServerID)),
		Fromtype: proto.String(conf.Server.ServerType),
		Toid:     proto.Int32(toid),
		Totype:   proto.String(toserver),
		Charid:   proto.String(charid),
		Msgid:    proto.Uint32(msgid),
		Msgdata:  msgbuff,
	}

	//SendToRedis
	queueName := fmt.Sprintf("queue_%v_%v", toserver, toid)
	msgbuff, err = proto.Marshal(iMsg)
	if err != nil {
		log.Error("SendMessageTo Marshal Error %v %v", msgid, err)
		return false
	}

	_, err = redis.DoWithTimeout(Predis.conn, 1*time.Second, "PUBLISH", queueName, msgbuff)
	if err != nil {
		log.Error("DoWithTimeout Error %v", err)
		return false
	} else {
		return true
	}
}
