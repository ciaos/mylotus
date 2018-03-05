package g

import (
	"server/conf"

	"github.com/name5566/leaf/db/mongodb"
)

var Mongo *mongodb.DialContext

func InitMongoConnection() {
	Mongo, _ = mongodb.Dial(conf.Server.MongoDBHost, 10)
}

func UninitMongoConnection() {
	Mongo.Close()
}
