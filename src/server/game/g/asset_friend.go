package g

import (
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func (player *Player) AssetFriend_AddApplyInfo(fromid uint32, m *clientmsg.Req_Friend_Operate) {
	if player == nil { //offline
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
		for _, applyinfo := range player.Asset.AssetFriend.ApplyList {
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
			player.Asset.AssetFriend.ApplyList = append(player.Asset.AssetFriend.ApplyList, apply)
			player.Asset.DirtyFlag_AssetFriend |= DIRTYFLAG_TO_ALL
		}
	}
}

func (player *Player) AssetFriend_DelFriend(charid uint32, friendid uint32) {
	if player == nil { //offline
		s := Mongo.Ref()
		defer Mongo.UnRef(s)
		c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
		c.Update(bson.M{"charid": charid}, bson.M{"$pull": bson.M{
			"friends": friendid,
		}})
	} else {
	reloop:
		for i, friend := range player.Asset.AssetFriend.Friends {
			if friend == friendid {
				player.Asset.AssetFriend.Friends = append(player.Asset.AssetFriend.Friends[0:i], player.Asset.AssetFriend.Friends[i+1:]...)
				player.Asset.DirtyFlag_AssetFriend |= DIRTYFLAG_TO_ALL
				goto reloop
			}
		}
	}
}

func (player *Player) AssetFriend_AcceptApplyInfo(charid uint32, friendid uint32) {
	if player == nil { //offline
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
		for i, applyinfo := range player.Asset.AssetFriend.ApplyList {
			if applyinfo.FromID == friendid {
				player.Asset.AssetFriend.ApplyList = append(player.Asset.AssetFriend.ApplyList[0:i], player.Asset.AssetFriend.ApplyList[i+1:]...)
				goto reloop
			}
		}

		exist := false
		for _, friend := range player.Asset.AssetFriend.Friends {
			if friend == friendid {
				exist = true
			}
		}
		if exist == false {
			player.Asset.AssetFriend.Friends = append(player.Asset.AssetFriend.Friends, friendid)
		}
		player.Asset.DirtyFlag_AssetFriend |= DIRTYFLAG_TO_ALL
	}
}

func (player *Player) AssetFriend_RejectApplyInfo(fromid uint32) {
	if player == nil {
		return
	}

reloop:
	for i, applyinfo := range player.Asset.AssetFriend.ApplyList {
		if applyinfo.FromID == fromid {
			player.Asset.AssetFriend.ApplyList = append(player.Asset.AssetFriend.ApplyList[0:i], player.Asset.AssetFriend.ApplyList[i+1:]...)
			player.Asset.DirtyFlag_AssetFriend |= DIRTYFLAG_TO_ALL
			goto reloop
		}
	}
}
