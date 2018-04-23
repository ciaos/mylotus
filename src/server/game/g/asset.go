package g

import (
	"server/msg/clientmsg"
	"time"

	"server/conf"
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

	AssetFriend_DirtyFlag      int8
	AssetCash_DirtyFlag        int8
	AssetMail_DirtyFlag        int8
	AssetItem_DirtyFlag        int8
	AssetHero_DirtyFlag        int8
	AssetTutorial_DirtyFlag    int8
	AssetStatistic_DirtyFlag   int8
	AssetAchievement_DirtyFlag int8
	AssetTask_DirtyFlag        int8

	lastSaveDBTime int64
}

////////////////////////////////////////////////////////////////////
// Interface
func (player *Player) LoadPlayerAsset() bool {
	ret := player.loadPlayerAssetFriend() &&
		player.loadPlayerAssetCash() &&
		player.loadPlayerAssetMail() &&
		player.loadPlayerAssetItem() &&
		player.loadPlayerAssetHero() &&
		player.loadPlayerAssetTutorial() &&
		player.loadPlayerAssetStatistic() &&
		player.loadPlayerAssetAchievement() &&
		player.loadPlayerAssetTask()

	player.Asset.lastSaveDBTime = time.Now().Unix()
	return ret
}

func (player *Player) SyncPlayerAsset() bool {
	player.GetPlayerAsset().AssetFriend_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetCash_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetMail_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetItem_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetHero_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetTutorial_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetStatistic_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetAchievement_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	player.GetPlayerAsset().AssetTask_DirtyFlag |= DIRTYFLAG_TO_CLIENT

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
	ret := player.savePlayerAssetFriend() &&
		player.savePlayerAssetCash() &&
		player.savePlayerAssetMail() &&
		player.savePlayerAssetItem() &&
		player.savePlayerAssetHero() &&
		player.savePlayerAssetTutorial() &&
		player.savePlayerAssetStatistic() &&
		player.savePlayerAssetAchievement() &&
		player.savePlayerAssetTask()

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
