package g

import (
	"server/msg/clientmsg"
	"time"

	"server/gamedata"
	"server/gamedata/cfg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

const (
	MAX_MAIL_CNT = 20
)

//
func (player *Player) loadPlayerAssetMail() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Mail)
	player.GetPlayerAsset().AssetMail = &clientmsg.Rlt_Asset_Mail{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetMail)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetMail.CharID = player.Char.CharID

		r := gamedata.CSVNewPlayer.Record(0)
		row := r.(*cfg.NewPlayer)

		rewards := &clientmsg.Rlt_Give_Reward{}
		for _, re := range row.InitMail.Rewards {
			reward := &clientmsg.Rlt_Give_Reward_Reward{
				X: int32(clientmsg.Type_Vec3X_TVX_CASH),
				Y: int32(re[0]),
				Z: int32(re[1]),
			}
			rewards.Rewardlist = append(rewards.Rewardlist, reward)
		}
		mail := CreateMail(clientmsg.MailInfo_MT_SYSTEM, player.Char.CharID, row.InitMail.Title, row.InitMail.Content, rewards, row.InitMail.Expirets)
		if mail == nil {
			log.Error("create new character %v mail error %v", player.Char.CharID, err)
			return false
		}
		player.GetPlayerAsset().AssetMail_AddMail(player.Char.CharID, mail)

		err = c.Insert(player.GetPlayerAsset().AssetMail)
	}
	if err != nil {
		log.Error("Load Player %v AssetMail Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetMail_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetMail() bool {
	if player.GetPlayerAsset().AssetMail_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Mail)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetMail)
	if err != nil {
		log.Error("Save Player %v AssetMail Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetMail_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetMail() {
	if (pinfo.player.GetPlayerAsset().AssetMail_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetMail)
		pinfo.player.GetPlayerAsset().AssetMail_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}

//interface
func CreateMail(mailtype clientmsg.MailInfo_MailType, mailownerid uint32, title string, content string, rewards *clientmsg.Rlt_Give_Reward, expirets int64) *clientmsg.MailInfo {
	mail := &clientmsg.MailInfo{
		Mailid:      bson.NewObjectId().Hex(),
		Mailtype:    mailtype,
		Mailownerid: mailownerid,
		Title:       title,
		Content:     content,
		Rewards:     rewards,
		CreateTime:  time.Now().Unix(),
		ExpireTime:  expirets,
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)

	c := s.DB(DB_NAME_GAME).C(TB_NAME_MAIL)
	err := c.Insert(mail)
	if err != nil {
		log.Error("create mail error %v", err)

		return nil
	}
	return mail
}

func (player *Player) AssetMail_CheckGlobalMail() {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)

	c := s.DB(DB_NAME_GAME).C(TB_NAME_MAIL)
	results := []clientmsg.MailInfo{}

	err := c.Find(bson.M{"mailtype": 0, "createtime": bson.M{"$gt": player.GetPlayerAsset().AssetCash_GetLastCheckGlobalMailTs()}, "expiretime": bson.M{"$lt": time.Now().Unix()}}).All(&results)
	if err == nil {
		for _, result := range results {
			player.Asset.AssetMail_AddMail(player.Char.CharID, &result)
		}
	} else {
		log.Error("AssetMail_CheckGlobalMail charid %v error %v", player.Char.CharID, err)
	}

	player.GetPlayerAsset().AssetCash_RefreshLastCheckGlobalMailTs()
}

func (asset *PlayerAsset) AssetMail_AddMail(charid uint32, m *clientmsg.MailInfo) {
	maildata := &clientmsg.Rlt_Asset_Mail_MyMailData{
		MailID:     m.Mailid,
		MailStatus: clientmsg.Rlt_Asset_Mail_MMS_NONE,
		CreateTime: time.Now().Unix(),
	}

	if asset == nil { //offline
		s := Mongo.Ref()
		defer Mongo.UnRef(s)

		c := s.DB(DB_NAME_GAME).C(AssetName_Mail)

		data := &clientmsg.Rlt_Asset_Mail{}
		err := c.Find(bson.M{"charid": charid}).One(data)
		if err == nil {
			if len(data.MailData) > MAX_MAIL_CNT {
				data.MailData = append(data.MailData[1:])
			}
			data.MailData = append(data.MailData, maildata)
			c.Update(bson.M{"charid": charid}, data)
		}
	} else {
		if m.Mailtype != clientmsg.MailInfo_MT_USER {
			for _, excludeid := range asset.AssetMail.MailIDExclude {
				if excludeid == m.Mailid {
					return
				}
			}
		}
		if len(asset.AssetMail.MailData) > MAX_MAIL_CNT {
			asset.AssetMail.MailData = append(asset.AssetMail.MailData[1:])
		}
		asset.AssetMail.MailData = append(asset.AssetMail.MailData, maildata)
		asset.AssetMail_DirtyFlag |= DIRTYFLAG_TO_ALL
	}
}

func (asset *PlayerAsset) AssetMail_Action(m *clientmsg.Req_Mail_Action) *clientmsg.Rlt_Mail_Action {
	if asset == nil {
		return nil
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(TB_NAME_MAIL)
	results := []clientmsg.MailInfo{}
	err := c.Find(bson.M{"mailid": bson.M{"$in": m.MailIDs}}).All(&results)

	if m.Action == clientmsg.MailActionType_MAT_LIST_MAIL {
		rsp := &clientmsg.Rlt_Mail_Action{
			Action: m.Action,
		}

		if err != nil {
			log.Error("AssetMail_Action %v Error %v", m.MailIDs, err)
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OTHER

			return rsp
		} else {
			for _, result := range results {
				rsp.Mails = append(rsp.Mails, &result)
			}

			return rsp
		}
	} else if m.Action == clientmsg.MailActionType_MAT_READ {
		for _, mailid := range m.MailIDs {
			for _, mail := range asset.AssetMail.MailData {
				if mail.MailID == mailid && mail.MailStatus == clientmsg.Rlt_Asset_Mail_MMS_NONE {
					mail.MailStatus = clientmsg.Rlt_Asset_Mail_MMS_READ

					asset.AssetMail_DirtyFlag |= DIRTYFLAG_TO_ALL
					break
				}
			}
		}

	} else if m.Action == clientmsg.MailActionType_MAT_RECEIVE {
		for _, mailid := range m.MailIDs {
			for _, mail := range asset.AssetMail.MailData {
				if mail.MailID == mailid && mail.MailStatus != clientmsg.Rlt_Asset_Mail_MMS_RECEIVED {
					mail.MailStatus = clientmsg.Rlt_Asset_Mail_MMS_RECEIVED

					//todo send gift

					asset.AssetMail_DirtyFlag |= DIRTYFLAG_TO_ALL
					break
				}
			}
		}
	} else if m.Action == clientmsg.MailActionType_MAT_ERASE {
		for _, mailid := range m.MailIDs {
			//erase mailid
			for i, maildata := range asset.AssetMail.MailData {
				if maildata.MailID == mailid {

					for _, result := range results {
						//check if global mail
						if maildata.MailID == result.Mailid {
							if result.Mailtype == clientmsg.MailInfo_MT_USER {
								c.Remove(bson.M{"mailid": mailid})
							} else {
								asset.AssetMail.MailIDExclude = append(asset.AssetMail.MailIDExclude, mailid)
							}
							break
						}
					}
					asset.AssetMail.MailData = append(asset.AssetMail.MailData[0:i], asset.AssetMail.MailData[i+1:]...)
					asset.AssetMail_DirtyFlag |= DIRTYFLAG_TO_ALL
					break
				}
			}
		}

	} else {
		log.Error("Invalid Mail Action %v", m.Action)
	}
	return nil
}
