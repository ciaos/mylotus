package g

import (
	"server/msg/clientmsg"
	"time"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

const (
	MAX_MAIL_CNT = 20
)

func CreateMail(mailtype clientmsg.MailInfo_MailType, mailownerid uint32, title string, content string, rewards *clientmsg.Rlt_Give_Reward, expirets int64) *clientmsg.MailInfo {
	mail := &clientmsg.MailInfo{
		Mailid:      bson.NewObjectId().String(),
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
		log.Error("create new character error %v", err)

		return nil
	}
	return mail
}

func (player *Player) AssetMail_CheckGlobalMail() {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)

	c := s.DB(DB_NAME_GAME).C(TB_NAME_MAIL)
	results := []clientmsg.MailInfo{}

	err := c.Find(bson.M{"mailtype": 0, "createtime": bson.M{"$gt": player.Asset.AssetCash_GetLastCheckGlobalMailTs()}, "expiretime": bson.M{"$lt": time.Now().Unix()}}).All(&results)
	if err == nil {
		for _, result := range results {
			player.Asset.AssetMail_AddMail(player.Char.CharID, &result)
		}
	} else {
		log.Error("AssetMail_CheckGlobalMail charid %v error %v", player.Char.CharID, err)
	}

	player.Asset.AssetCash_RefreshLastCheckGlobalMailTs()
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
		asset.DirtyFlag_AssetMail |= DIRTYFLAG_TO_ALL
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

					asset.DirtyFlag_AssetMail |= DIRTYFLAG_TO_ALL
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

					asset.DirtyFlag_AssetMail |= DIRTYFLAG_TO_ALL
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
					asset.DirtyFlag_AssetMail |= DIRTYFLAG_TO_ALL
					break
				}
			}
		}

	} else {
		log.Error("Invalid Mail Action %v", m.Action)
	}
	return nil
}
