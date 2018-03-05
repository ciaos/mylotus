package data

import (
	"time"

	"github.com/name5566/leaf/gate"
	"gopkg.in/mgo.v2/bson"
)

const (
	STATUS_OFFLINE = 0
	STATUS_ONLINE  = 1
	STATUS_BATTLE  = 2
)

type Character struct {
	Id         bson.ObjectId `json:"id"        bson:"_id"`
	UserId     string
	GsId       int32
	Status     int32
	CreateTime time.Time
	UpdateTime time.Time
}

type UserInfo struct {
	CharId   string
	CharName string
	Level    int32
}

var PlayerManager = make(map[string]*gate.Agent)
