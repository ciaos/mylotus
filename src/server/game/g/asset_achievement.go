package g

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func (player *Player) loadPlayerAssetAchievement() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Achievement)
	player.GetPlayerAsset().AssetAchievement = &clientmsg.Rlt_Asset_Achievement{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetAchievement)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetAchievement.CharID = player.Char.CharID
		err = c.Insert(player.GetPlayerAsset().AssetAchievement)
	}
	if err != nil {
		log.Error("Load Player %v AssetAchievement Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetAchievement_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetAchievement() bool {
	if player.GetPlayerAsset().AssetAchievement_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Achievement)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetAchievement)
	if err != nil {
		log.Error("Save Player %v AssetAchievement Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetAchievement_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetAchievement() {
	if (pinfo.player.GetPlayerAsset().AssetAchievement_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetAchievement)
		pinfo.player.GetPlayerAsset().AssetAchievement_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}
