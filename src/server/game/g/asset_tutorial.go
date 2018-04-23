package g

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func (player *Player) loadPlayerAssetTutorial() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Tutorial)
	player.GetPlayerAsset().AssetTutorial = &clientmsg.Rlt_Asset_Tutorial{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetTutorial)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetTutorial.CharID = player.Char.CharID
		err = c.Insert(player.GetPlayerAsset().AssetTutorial)
	}
	if err != nil {
		log.Error("Load Player %v AssetTutorial Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetTutorial_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetTutorial() bool {
	if player.GetPlayerAsset().AssetTutorial_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Tutorial)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetTutorial)
	if err != nil {
		log.Error("Save Player %v AssetTutorial Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetTutorial_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetTutorial() {
	if (pinfo.player.GetPlayerAsset().AssetTutorial_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetTutorial)
		pinfo.player.GetPlayerAsset().AssetTutorial_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}
