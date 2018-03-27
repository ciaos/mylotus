package g

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

const (
	MAX_TABLE_COUNT = 10000
	MAX_ROOM_COUNT  = 10000

	DB_NAME_LOGIN     = "login"
	TB_NAME_ACCOUNT   = "account"
	DB_NAME_GAME      = "game"
	TB_NAME_CHARACTER = "character"
	TB_NAME_COUNTER   = "counter"

	CRYPTO_PREFIX = "room_%d"
)

type Character struct {
	Id         bson.ObjectId `json:"id"        bson:"_id"`
	CharId     uint32
	UserId     uint32
	GsId       int32
	Status     int32
	CharName   string
	CreateTime time.Time
	UpdateTime time.Time
}
