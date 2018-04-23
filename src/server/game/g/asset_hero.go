package g

import (
	"time"

	"server/gamedata"
	"server/gamedata/cfg"
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/log"

	"gopkg.in/mgo.v2/bson"
)

func (player *Player) loadPlayerAssetHero() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Hero)
	player.GetPlayerAsset().AssetHero = &clientmsg.Rlt_Asset_Hero{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetHero)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetHero.CharID = player.Char.CharID

		r := gamedata.CSVNewPlayer.Record(0)
		row := r.(*cfg.NewPlayer)

		for _, hero := range row.InitHeros {
			player.GetPlayerAsset().AssetHero_AddHero(player.Char.CharID, uint32(hero), 0)
		}

		err = c.Insert(player.GetPlayerAsset().AssetHero)
	}
	if err != nil {
		log.Error("Load Player %v AssetHero Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetHero_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetHero() bool {
	if player.GetPlayerAsset().AssetHero_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Hero)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetHero)
	if err != nil {
		log.Error("Save Player %v AssetHero Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetHero_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetHero() {
	if (pinfo.player.GetPlayerAsset().AssetHero_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetHero)
		pinfo.player.GetPlayerAsset().AssetHero_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}

//method
func (asset *PlayerAsset) AssetHero_AddHero(charid uint32, chartypeid uint32, deadlinetime int64) {
	data := &clientmsg.Rlt_Asset_Hero{}

	if asset == nil { //offline
		s := Mongo.Ref()
		defer Mongo.UnRef(s)

		c := s.DB(DB_NAME_GAME).C(AssetName_Hero)
		err := c.Find(bson.M{"charid": charid}).One(data)
		if err != nil {
			return
		}
	} else {
		data = asset.AssetHero
	}

	hasowned := false
	for _, role := range data.Roles {
		if role.CharTypeID == chartypeid {
			hasowned = true

			if deadlinetime == 0 {
				role.RoleStatus = clientmsg.Rlt_Asset_Hero_RS_OWNED
				role.DeadLineTime = 0
			} else {
				if role.RoleStatus == clientmsg.Rlt_Asset_Hero_RS_LIMITED && role.DeadLineTime < deadlinetime {
					role.DeadLineTime = deadlinetime
				}
			}
		}
	}
	if hasowned == false {
		roleinfo := &clientmsg.Rlt_Asset_Hero_RoleInfo{
			CharTypeID:   chartypeid,
			DeadLineTime: deadlinetime,
			OwnedTime:    time.Now().Unix(),
		}
		if deadlinetime == 0 {
			roleinfo.RoleStatus = clientmsg.Rlt_Asset_Hero_RS_OWNED
		} else {
			roleinfo.RoleStatus = clientmsg.Rlt_Asset_Hero_RS_LIMITED
		}
		data.Roles = append(data.Roles, roleinfo)
	}

	if asset == nil {
		s := Mongo.Ref()
		defer Mongo.UnRef(s)

		c := s.DB(DB_NAME_GAME).C(AssetName_Hero)

		c.Update(bson.M{"charid": charid}, data)
	} else {
		asset.AssetMail_DirtyFlag |= DIRTYFLAG_TO_ALL
	}
}
