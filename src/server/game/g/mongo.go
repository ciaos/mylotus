package g

import (
	"server/conf"

	"github.com/name5566/leaf/db/mongodb"
	"github.com/name5566/leaf/log"
)

var Mongo *mongodb.DialContext

func InitMongoConnection() {
	var err error
	Mongo, err = mongodb.Dial(conf.Server.MongoDBHost, 10)
	if err != nil {
		log.Fatal("InitMongoConnection Error %v", err)
	}
}

func UninitMongoConnection() {
	Mongo.Close()
}
