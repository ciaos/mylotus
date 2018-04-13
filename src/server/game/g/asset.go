package g

import (
	"server/msg/clientmsg"
	"time"

	"server/conf"
	"server/gamedata"
	"server/gamedata/cfg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

const (
	DIRTYFLAG_TO_CLIENT = 1
	DIRTYFLAG_TO_DB     = 2
	DIRTYFLAG_TO_ALL    = 3

	AssetName_Friend      = "asset_friend"
	AssetName_Cash        = "asset_cash"
	AssetName_Mail        = "asset_mail"
	AssetName_Item        = "asset_item"
	AssetName_Hero        = "asset_hero"
	AssetName_Tutorial    = "asset_tutorial"
	AssetName_Statistic   = "asset_statistic"
	AssetName_Achievement = "asset_achievement"
	AssetName_Task        = "asset_task"
)

type PlayerAsset struct {
	AssetFriend      *clientmsg.Rlt_Asset_Friend
	AssetCash        *clientmsg.Rlt_Asset_Cash
	AssetMail        *clientmsg.Rlt_Asset_Mail
	AssetItem        *clientmsg.Rlt_Asset_Item
	AssetHero        *clientmsg.Rlt_Asset_Hero
	AssetTutorial    *clientmsg.Rlt_Asset_Tutorial
	AssetStatistic   *clientmsg.Rlt_Asset_Statistic
	AssetAchievement *clientmsg.Rlt_Asset_Achievement
	AssetTask        *clientmsg.Rlt_Asset_Task

	DirtyFlag_AssetFriend      int8
	DirtyFlag_AssetCash        int8
	DirtyFlag_AssetMail        int8
	DirtyFlag_AssetItem        int8
	DirtyFlag_AssetHero        int8
	DirtyFlag_AssetTutorial    int8
	DirtyFlag_AssetStatistic   int8
	DirtyFlag_AssetAchievement int8
	DirtyFlag_AssetTask        int8

	lastSaveDBTime int64
}

////////////////////////////////////////////////////////////////////
// Interface
func (player *Player) LoadPlayerAsset() bool {
	ret := player.loadPlayerAssetFriend() && player.loadPlayerAssetCash() && player.loadPlayerAssetMail() && player.loadPlayerAssetItem() && player.loadPlayerAssetHero() && player.loadPlayerAssetTutorial() && player.loadPlayerAssetStatistic() && player.loadPlayerAssetAchievement() && player.loadPlayerAssetTask()

	player.Asset.lastSaveDBTime = time.Now().Unix()
	return ret
}

func (player *Player) SyncPlayerAsset() bool {
	player.Asset.DirtyFlag_AssetFriend |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetMail |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetItem |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetHero |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetTutorial |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetStatistic |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetAchievement |= DIRTYFLAG_TO_CLIENT
	player.Asset.DirtyFlag_AssetTask |= DIRTYFLAG_TO_CLIENT

	return true
}

func (pinfo *PlayerInfo) UpdatePlayerAsset(now *time.Time) {

	//sync to client
	pinfo.syncPlayerAssetFriend()
	pinfo.syncPlayerAssetCash()
	pinfo.syncPlayerAssetMail()
	pinfo.syncPlayerAssetItem()
	pinfo.syncPlayerAssetHero()
	pinfo.syncPlayerAssetTutorial()
	pinfo.syncPlayerAssetStatistic()
	pinfo.syncPlayerAssetAchievement()
	pinfo.syncPlayerAssetTask()

	//sync to db
	if now.Unix()-pinfo.player.Asset.lastSaveDBTime > int64(conf.Server.SaveAssetStep) {

		pinfo.player.savePlayerAssetFriend()
		pinfo.player.savePlayerAssetCash()
		pinfo.player.savePlayerAssetMail()
		pinfo.player.savePlayerAssetItem()
		pinfo.player.savePlayerAssetHero()
		pinfo.player.savePlayerAssetTutorial()
		pinfo.player.savePlayerAssetStatistic()
		pinfo.player.savePlayerAssetAchievement()
		pinfo.player.savePlayerAssetTask()

		pinfo.player.Asset.lastSaveDBTime = now.Unix()
	}
}

func (player *Player) SavePlayerAsset() bool {
	ret := player.savePlayerAssetFriend() && player.savePlayerAssetCash() && player.savePlayerAssetMail() && player.savePlayerAssetItem() && player.savePlayerAssetHero() && player.savePlayerAssetTutorial() && player.savePlayerAssetStatistic() && player.savePlayerAssetAchievement() && player.savePlayerAssetTask()

	player.Asset.lastSaveDBTime = time.Now().Unix()
	return ret
}

func (player *Player) GetPlayerAsset() *PlayerAsset {
	if player != nil {
		return &player.Asset
	} else {
		return nil
	}
}

////////////////////////////////////////////////////////////////////
//load
func (player *Player) loadPlayerAssetFriend() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
	player.Asset.AssetFriend = &clientmsg.Rlt_Asset_Friend{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetFriend)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetFriend.CharID = player.Char.CharID
		err = c.Insert(player.Asset.AssetFriend)
	}
	if err != nil {
		log.Error("Load Player %v AssetFriend Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetFriend |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetCash() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Cash)
	player.Asset.AssetCash = &clientmsg.Rlt_Asset_Cash{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetCash)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetCash.CharID = player.Char.CharID
		player.Asset.AssetCash.Level = 1
		player.Asset.AssetCash.LastCheckGlobalMailTs = player.Char.CreateTime.Unix()
		r := gamedata.CSVNewPlayer.Record(0)
		row := r.(*cfg.NewPlayer)
		for _, cash := range row.InitCash {
			if len(cash) == 2 {
				switch clientmsg.Type_CashType(cash[0]) {
				case clientmsg.Type_CashType_TCT_GOLD:
					player.Asset.AssetCash_AddGoldCoin(cash[1])
				case clientmsg.Type_CashType_TCT_SILVER:
					player.Asset.AssetCash_AddSilverCoin(cash[1])
				case clientmsg.Type_CashType_TCT_DIAMOND:
					player.Asset.AssetCash_AddDiamondCoin(cash[1])
				}
			}
		}

		err = c.Insert(player.Asset.AssetCash)
	}
	if err != nil {
		log.Error("Load Player %v AssetCash Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetMail() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Mail)
	player.Asset.AssetMail = &clientmsg.Rlt_Asset_Mail{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetMail)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetMail.CharID = player.Char.CharID

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
		player.Asset.AssetMail_AddMail(player.Char.CharID, mail)

		err = c.Insert(player.Asset.AssetMail)
	}
	if err != nil {
		log.Error("Load Player %v AssetMail Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetMail |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetItem() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Item)
	player.Asset.AssetItem = &clientmsg.Rlt_Asset_Item{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetItem)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetItem.CharID = player.Char.CharID
		err = c.Insert(player.Asset.AssetItem)
	}
	if err != nil {
		log.Error("Load Player %v AssetItem Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetItem |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetHero() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Hero)
	player.Asset.AssetHero = &clientmsg.Rlt_Asset_Hero{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetHero)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetHero.CharID = player.Char.CharID

		r := gamedata.CSVNewPlayer.Record(0)
		row := r.(*cfg.NewPlayer)

		for _, hero := range row.InitHeros {
			player.Asset.AssetHero_AddHero(player.Char.CharID, uint32(hero), 0)
		}

		err = c.Insert(player.Asset.AssetHero)
	}
	if err != nil {
		log.Error("Load Player %v AssetHero Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetHero |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetTutorial() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Tutorial)
	player.Asset.AssetTutorial = &clientmsg.Rlt_Asset_Tutorial{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetTutorial)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetTutorial.CharID = player.Char.CharID
		err = c.Insert(player.Asset.AssetTutorial)
	}
	if err != nil {
		log.Error("Load Player %v AssetTutorial Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetTutorial |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetStatistic() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Statistic)
	player.Asset.AssetStatistic = &clientmsg.Rlt_Asset_Statistic{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetStatistic)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetStatistic.CharID = player.Char.CharID
		err = c.Insert(player.Asset.AssetStatistic)
	}
	if err != nil {
		log.Error("Load Player %v AssetStatistic Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetStatistic |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetAchievement() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Achievement)
	player.Asset.AssetAchievement = &clientmsg.Rlt_Asset_Achievement{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetAchievement)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetAchievement.CharID = player.Char.CharID
		err = c.Insert(player.Asset.AssetAchievement)
	}
	if err != nil {
		log.Error("Load Player %v AssetAchievement Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetAchievement |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) loadPlayerAssetTask() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Task)
	player.Asset.AssetTask = &clientmsg.Rlt_Asset_Task{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.Asset.AssetTask)
	if err != nil && err.Error() == "not found" {
		player.Asset.AssetTask.CharID = player.Char.CharID
		err = c.Insert(player.Asset.AssetTask)
	}
	if err != nil {
		log.Error("Load Player %v AssetTask Error %v", player.Char.CharID, err)
		return false
	}
	player.Asset.DirtyFlag_AssetTask |= DIRTYFLAG_TO_CLIENT
	return true
}

// sync
func (pinfo *PlayerInfo) syncPlayerAssetFriend() {
	if (pinfo.player.Asset.DirtyFlag_AssetFriend & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetFriend)
		pinfo.player.Asset.DirtyFlag_AssetFriend ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetCash() {
	if (pinfo.player.Asset.DirtyFlag_AssetCash & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetCash)
		pinfo.player.Asset.DirtyFlag_AssetCash ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetMail() {
	if (pinfo.player.Asset.DirtyFlag_AssetMail & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetMail)
		pinfo.player.Asset.DirtyFlag_AssetMail ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetItem() {
	if (pinfo.player.Asset.DirtyFlag_AssetItem & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetItem)
		pinfo.player.Asset.DirtyFlag_AssetItem ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetHero() {
	if (pinfo.player.Asset.DirtyFlag_AssetHero & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetHero)
		pinfo.player.Asset.DirtyFlag_AssetHero ^= DIRTYFLAG_TO_CLIENT

		log.Debug("syncPlayerAssetHero to %v", pinfo.player.Char.CharID)
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetTutorial() {
	if (pinfo.player.Asset.DirtyFlag_AssetTutorial & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetTutorial)
		pinfo.player.Asset.DirtyFlag_AssetTutorial ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetStatistic() {
	if (pinfo.player.Asset.DirtyFlag_AssetStatistic & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetStatistic)
		pinfo.player.Asset.DirtyFlag_AssetStatistic ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetAchievement() {
	if (pinfo.player.Asset.DirtyFlag_AssetAchievement & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetAchievement)
		pinfo.player.Asset.DirtyFlag_AssetAchievement ^= DIRTYFLAG_TO_CLIENT
	}
}

func (pinfo *PlayerInfo) syncPlayerAssetTask() {
	if (pinfo.player.Asset.DirtyFlag_AssetTask & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.Asset.AssetTask)
		pinfo.player.Asset.DirtyFlag_AssetTask ^= DIRTYFLAG_TO_CLIENT
	}
}

// save
func (player *Player) savePlayerAssetFriend() bool {
	if player.Asset.DirtyFlag_AssetFriend&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Friend)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetFriend)
	if err != nil {
		log.Error("Save Player %v AssetFriend Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetFriend ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetCash() bool {
	if player.Asset.DirtyFlag_AssetCash&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Cash)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetCash)
	if err != nil {
		log.Error("Save Player %v AssetCash Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetCash ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetMail() bool {
	if player.Asset.DirtyFlag_AssetMail&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Mail)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetMail)
	if err != nil {
		log.Error("Save Player %v AssetMail Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetMail ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetItem() bool {
	if player.Asset.DirtyFlag_AssetItem&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Item)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetItem)
	if err != nil {
		log.Error("Save Player %v AssetItem Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetItem ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetHero() bool {
	if player.Asset.DirtyFlag_AssetHero&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Hero)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetHero)
	if err != nil {
		log.Error("Save Player %v AssetHero Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetHero ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetTutorial() bool {
	if player.Asset.DirtyFlag_AssetTutorial&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Tutorial)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetTutorial)
	if err != nil {
		log.Error("Save Player %v AssetTutorial Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetTutorial ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetStatistic() bool {
	if player.Asset.DirtyFlag_AssetStatistic&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Statistic)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetStatistic)
	if err != nil {
		log.Error("Save Player %v AssetStatistic Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetStatistic ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetAchievement() bool {
	if player.Asset.DirtyFlag_AssetAchievement&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Achievement)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetAchievement)
	if err != nil {
		log.Error("Save Player %v AssetAchievement Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetAchievement ^= DIRTYFLAG_TO_DB
	return true
}

func (player *Player) savePlayerAssetTask() bool {
	if player.Asset.DirtyFlag_AssetTask&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Task)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.Asset.AssetTask)
	if err != nil {
		log.Error("Save Player %v AssetTask Error %v", player.Char.CharID, err)
		return false
	}

	player.Asset.DirtyFlag_AssetTask ^= DIRTYFLAG_TO_DB
	return true
}
