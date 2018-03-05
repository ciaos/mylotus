package g

import (
	"time"

	"gopkg.in/mgo.v2/bson"
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
