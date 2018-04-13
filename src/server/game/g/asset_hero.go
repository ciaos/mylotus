package g

import (
	"time"

	"server/msg/clientmsg"

	"gopkg.in/mgo.v2/bson"
)

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
		asset.DirtyFlag_AssetHero |= DIRTYFLAG_TO_ALL
	}
}
