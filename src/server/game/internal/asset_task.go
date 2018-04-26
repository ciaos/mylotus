package internal

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func (player *Player) loadPlayerAssetTask() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Task)
	player.GetPlayerAsset().AssetTask = &clientmsg.Rlt_Asset_Task{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetTask)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetTask.CharID = player.Char.CharID
		err = c.Insert(player.GetPlayerAsset().AssetTask)
	}
	if err != nil {
		log.Error("Load Player %v AssetTask Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetTask_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetTask() bool {
	if player.GetPlayerAsset().AssetTask_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Task)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetTask)
	if err != nil {
		log.Error("Save Player %v AssetTask Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetTask_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetTask() {
	if (pinfo.player.GetPlayerAsset().AssetTask_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetTask)
		pinfo.player.GetPlayerAsset().AssetTask_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}
