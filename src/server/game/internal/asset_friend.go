package internal

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

//common interface
func (player *Player) loadPlayerAssetFriend() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
	player.GetPlayerAsset().AssetFriend = &clientmsg.Rlt_Asset_Friend{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetFriend)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetFriend.CharID = player.Char.CharID

		err = c.Insert(player.GetPlayerAsset().AssetFriend)
	}
	if err != nil {
		log.Error("Load Player %v AssetFriend Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetFriend_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetFriend() bool {
	if player.GetPlayerAsset().AssetFriend_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetFriend)
	if err != nil {
		log.Error("Save Player %v AssetFriend Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetFriend_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetFriend() {
	if (pinfo.player.GetPlayerAsset().AssetFriend_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetFriend)
		pinfo.player.GetPlayerAsset().AssetFriend_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}

//method
func (asset *PlayerAsset) AssetFriend_AddApplyInfo(fromid uint32, m *clientmsg.Req_Friend_Operate) {
	if asset == nil { //offline
		s := Mongo.Ref()
		defer Mongo.UnRef(s)
		c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
		exist, err := c.Find(bson.M{"charid": m.OperateCharID, "applylist.fromid": fromid}).Count()
		if exist == 0 && err == nil {
			err := c.Update(bson.M{"charid": m.OperateCharID}, bson.M{"$push": bson.M{
				"applylist": bson.M{"fromid": fromid, "msg": m},
			}})
			if err != nil {
				log.Error("FriendOperateActionType_FOAT_ADD_FRIEND Error %v", err)
			}
		}
	} else {
		exist := false
		for _, applyinfo := range asset.AssetFriend.ApplyList {
			if applyinfo.FromID == fromid {
				exist = true
				break
			}
		}
		if exist == false {
			apply := &clientmsg.Rlt_Asset_Friend_ApplyInfo{
				FromID: fromid,
				Msg:    m,
			}
			asset.AssetFriend.ApplyList = append(asset.AssetFriend.ApplyList, apply)
			asset.AssetFriend_DirtyFlag |= DIRTYFLAG_TO_ALL
		}
	}
}

func (asset *PlayerAsset) AssetFriend_DelFriend(charid uint32, friendid uint32) {
	if asset == nil { //offline
		s := Mongo.Ref()
		defer Mongo.UnRef(s)
		c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
		c.Update(bson.M{"charid": charid}, bson.M{"$pull": bson.M{
			"friends": friendid,
		}})
	} else {
	reloop:
		for i, friend := range asset.AssetFriend.Friends {
			if friend == friendid {
				asset.AssetFriend.Friends = append(asset.AssetFriend.Friends[0:i], asset.AssetFriend.Friends[i+1:]...)
				asset.AssetFriend_DirtyFlag |= DIRTYFLAG_TO_ALL
				goto reloop
			}
		}
	}
}

func (asset *PlayerAsset) AssetFriend_AcceptApplyInfo(charid uint32, friendid uint32) {
	if asset == nil { //offline
		s := Mongo.Ref()
		defer Mongo.UnRef(s)
		c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
		exist, _ := c.Find(bson.M{"charid": charid, "friends": friendid}).Count()
		if exist == 0 {
			c.Update(bson.M{"charid": charid}, bson.M{"$push": bson.M{
				"friends": friendid,
			}})
		}
	} else {
	reloop:
		for i, applyinfo := range asset.AssetFriend.ApplyList {
			if applyinfo.FromID == friendid {
				asset.AssetFriend.ApplyList = append(asset.AssetFriend.ApplyList[0:i], asset.AssetFriend.ApplyList[i+1:]...)
				goto reloop
			}
		}

		exist := false
		for _, friend := range asset.AssetFriend.Friends {
			if friend == friendid {
				exist = true
			}
		}
		if exist == false {
			asset.AssetFriend.Friends = append(asset.AssetFriend.Friends, friendid)
		}
		asset.AssetFriend_DirtyFlag |= DIRTYFLAG_TO_ALL
	}
}

func (asset *PlayerAsset) AssetFriend_RejectApplyInfo(fromid uint32) {
	if asset == nil {
		return
	}

reloop:
	for i, applyinfo := range asset.AssetFriend.ApplyList {
		if applyinfo.FromID == fromid {
			asset.AssetFriend.ApplyList = append(asset.AssetFriend.ApplyList[0:i], asset.AssetFriend.ApplyList[i+1:]...)
			asset.AssetFriend_DirtyFlag |= DIRTYFLAG_TO_ALL
			goto reloop
		}
	}
}

func (asset *PlayerAsset) AssetFriend_QueryCharIDGSID(charid uint32) (bool, int32) {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(TB_NAME_CHARACTER)

	character := &Character{}
	err := c.Find(bson.M{"charid": charid}).Select(bson.M{"gsid": 1}).One(character)
	if err == nil {
		return true, character.GsId
	}

	log.Error("AssetFriend_QueryCharIDGSID Charid %v error %v", charid, err)
	return false, 0
}
