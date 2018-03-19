package g

import (
	"fmt"
	"hash/crc32"
	"server/conf"
	"server/msg/proxymsg"
	"sync"
	"time"

	"github.com/ciaos/leaf/log"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
)

var Predis *RedisProxy
var m *sync.Mutex

type RedisProxy struct {
	conn redis.Conn
}

func InitRedisConnection() {
	Predis = new(RedisProxy)
	var err error
	Predis.conn, err = redis.Dial("tcp", conf.Server.RedisHost)
	if err != nil {
		log.Fatal("InitRedisConnection Error %v", err)
	}
	Predis.conn.Do("auth", conf.Server.RedisPassWord)

	m = new(sync.Mutex)
}
func UninitRedisConnection() {
	Predis.conn.Close()
}

func RandSendMessageTo(toserver string, charid string, msgid uint32, msgdata interface{}) bool {

	crc32ID := crc32.ChecksumIEEE([]byte(charid))

	switch toserver {
	case "matchserver":
		if len(conf.Server.MatchServerList) > 0 {
			idx := int(crc32ID) % len(conf.Server.MatchServerList)
			matchserver := &conf.Server.MatchServerList[idx]
			return SendMessageTo(int32((*matchserver).ServerID), (*matchserver).ServerType, charid, msgid, msgdata)
		} else {
			return false
		}
	case "battleserver":
		if len(conf.Server.BattleServerList) > 0 {
			idx := int(crc32ID) % len(conf.Server.BattleServerList)
			battleserver := &conf.Server.BattleServerList[idx]
			return SendMessageTo(int32((*battleserver).ServerID), (*battleserver).ServerType, charid, msgid, msgdata)
		} else {
			return false
		}
	}

	return false
}

func SendMessageTo(toid int32, toserver string, charid string, msgid uint32, msgdata interface{}) bool {

	//EncodeMsgData
	msgbuff, err := proto.Marshal(msgdata.(proto.Message))
	if err != nil {
		log.Error("protobuf Marsha1 error %v", err)
		return false
	}

	iMsg := &proxymsg.InternalMessage{
		Fromid:   int32(conf.Server.ServerID),
		Fromtype: conf.Server.ServerType,
		Toid:     toid,
		Totype:   toserver,
		Charid:   charid,
		Msgid:    msgid,
		Msgdata:  msgbuff,
	}

	//SendToRedis
	queueName := fmt.Sprintf("queue_%v_%v", toserver, toid)
	msgbuff, err = proto.Marshal(iMsg)
	if err != nil {
		log.Error("SendMessageTo Marshal Error %v %v", msgid, err)
		return false
	}

	m.Lock()
	defer m.Unlock()
	_, err = redis.DoWithTimeout(Predis.conn, 1*time.Second, "PUBLISH", queueName, msgbuff)
	if err != nil {
		log.Error("DoWithTimeout queueName %v Error %v", queueName, err)
		return false
	} else {
		return true
	}
}
