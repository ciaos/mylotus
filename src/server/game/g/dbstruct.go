package g

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

const (
	MAX_TABLE_COUNT = 10000
	MAX_ROOM_COUNT  = 10000

	CRYPTO_PREFIX = "room_%d"
)

type Character struct {
	Id         bson.ObjectId `json:"id"        bson:"_id"`
	CharId     uint32
	UserId     uint32
	GsId       int32
	Status     int32
	CreateTime time.Time
	UpdateTime time.Time
}

type UserInfo struct {
	CharId   uint32
	CharName string
	Level    int32
}
