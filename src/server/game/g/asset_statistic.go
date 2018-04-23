package g

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func (player *Player) loadPlayerAssetStatistic() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Statistic)
	player.GetPlayerAsset().AssetStatistic = &clientmsg.Rlt_Asset_Statistic{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetStatistic)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetStatistic.CharID = player.Char.CharID
		err = c.Insert(player.GetPlayerAsset().AssetStatistic)
	}
	if err != nil {
		log.Error("Load Player %v AssetStatistic Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetStatistic_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetStatistic() bool {
	if player.GetPlayerAsset().AssetStatistic_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Statistic)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetStatistic)
	if err != nil {
		log.Error("Save Player %v AssetStatistic Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetStatistic_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetStatistic() {
	if (pinfo.player.GetPlayerAsset().AssetStatistic_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetStatistic)
		pinfo.player.GetPlayerAsset().AssetStatistic_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}
