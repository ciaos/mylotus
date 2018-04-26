package internal

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func (player *Player) loadPlayerAssetItem() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Item)
	player.GetPlayerAsset().AssetItem = &clientmsg.Rlt_Asset_Item{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetItem)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetItem.CharID = player.Char.CharID
		err = c.Insert(player.GetPlayerAsset().AssetItem)
	}
	if err != nil {
		log.Error("Load Player %v AssetItem Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetItem_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetItem() bool {
	if player.GetPlayerAsset().AssetItem_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Item)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetItem)
	if err != nil {
		log.Error("Save Player %v AssetItem Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetItem_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetItem() {
	if (pinfo.player.GetPlayerAsset().AssetItem_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetItem)
		pinfo.player.GetPlayerAsset().AssetItem_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}
