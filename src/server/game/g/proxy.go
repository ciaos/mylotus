package g

import (
	"fmt"
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

func RandSendMessageTo(toserver string, charid uint32, msgid proxymsg.ProxyMessageType, msgdata interface{}) (int, bool) {

	switch toserver {
	case "matchserver":
		if len(conf.Server.MatchServerList) > 0 {
			idx := int(charid) % len(conf.Server.MatchServerList)
			matchserver := &conf.Server.MatchServerList[idx]
			ret := SendMessageTo(int32((*matchserver).ServerID), conf.Server.MatchServerRename, charid, msgid, msgdata)
			return matchserver.ServerID, ret
		} else {
			return 0, false
		}
	case "battleserver":
		if len(conf.Server.BattleServerList) > 0 {
			idx := int(charid) % len(conf.Server.BattleServerList)
			battleserver := &conf.Server.BattleServerList[idx]
			ret := SendMessageTo(int32((*battleserver).ServerID), conf.Server.BattleServerRename, charid, msgid, msgdata)
			return battleserver.ServerID, ret
		} else {
			return 0, false
		}
	}

	return 0, false
}

func BroadCastMessageTo(toserver string, charid uint32, msgid proxymsg.ProxyMessageType, msgdata interface{}) {

	switch toserver {
	case "matchserver":
		for _, server := range conf.Server.MatchServerList {
			SendMessageTo(int32(server.ServerID), conf.Server.MatchServerRename, charid, msgid, msgdata)
		}
	}
}

func SendMessageTo(toid int32, toserver string, charid uint32, msgid proxymsg.ProxyMessageType, msgdata interface{}) bool {
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
		Msgid:    uint32(msgid),
		Msgdata:  msgbuff,
		Time:     time.Now().Unix(),
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
