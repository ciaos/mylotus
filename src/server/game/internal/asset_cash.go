package internal

import (
	"server/msg/clientmsg"
	"time"

	"server/gamedata"
	"server/gamedata/cfg"

	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

//common interface
func (player *Player) loadPlayerAssetCash() bool {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Cash)
	player.GetPlayerAsset().AssetCash = &clientmsg.Rlt_Asset_Cash{}
	err := c.Find(bson.M{"charid": player.Char.CharID}).One(&player.GetPlayerAsset().AssetCash)
	if err != nil && err.Error() == "not found" {
		player.GetPlayerAsset().AssetCash.CharID = player.Char.CharID
		player.GetPlayerAsset().AssetCash.Level = 1
		player.GetPlayerAsset().AssetCash.LastCheckGlobalMailTs = player.Char.CreateTime.Unix()
		r := gamedata.CSVNewPlayer.Record(0)
		row := r.(*cfg.NewPlayer)
		for _, cash := range row.InitCash {
			if len(cash) == 2 {
				switch clientmsg.Type_CashType(cash[0]) {
				case clientmsg.Type_CashType_TCT_GOLD:
					player.GetPlayerAsset().AssetCash_AddGoldCoin(cash[1])
				case clientmsg.Type_CashType_TCT_SILVER:
					player.GetPlayerAsset().AssetCash_AddSilverCoin(cash[1])
				case clientmsg.Type_CashType_TCT_DIAMOND:
					player.GetPlayerAsset().AssetCash_AddDiamondCoin(cash[1])
				}
			}
		}

		err = c.Insert(player.GetPlayerAsset().AssetCash)
	}
	if err != nil {
		log.Error("Load Player %v AssetCash Error %v", player.Char.CharID, err)
		return false
	}
	player.GetPlayerAsset().AssetCash_DirtyFlag |= DIRTYFLAG_TO_CLIENT
	return true
}

func (player *Player) savePlayerAssetCash() bool {
	if player.GetPlayerAsset().AssetCash_DirtyFlag&DIRTYFLAG_TO_DB == 0 {
		return true
	}

	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(AssetName_Cash)
	err := c.Update(bson.M{"charid": player.Char.CharID}, player.GetPlayerAsset().AssetCash)
	if err != nil {
		log.Error("Save Player %v AssetCash Error %v", player.Char.CharID, err)
		return false
	}

	player.GetPlayerAsset().AssetCash_DirtyFlag ^= DIRTYFLAG_TO_DB
	return true
}

func (pinfo *PlayerInfo) syncPlayerAssetCash() {
	if (pinfo.player.GetPlayerAsset().AssetCash_DirtyFlag & DIRTYFLAG_TO_CLIENT) != 0 {
		(*pinfo.agent).WriteMsg(pinfo.player.GetPlayerAsset().AssetCash)
		pinfo.player.GetPlayerAsset().AssetCash_DirtyFlag ^= DIRTYFLAG_TO_CLIENT
	}
}

//method
func (asset *PlayerAsset) AssetCash_AddExp(exp uint32) {
	asset.AssetCash.Exp += exp

	asset.AssetCash_DirtyFlag |= DIRTYFLAG_TO_ALL
}

func (asset *PlayerAsset) AssetCash_RefreshLastCheckGlobalMailTs() {
	asset.AssetCash.LastCheckGlobalMailTs = time.Now().Unix()

	asset.AssetCash_DirtyFlag |= DIRTYFLAG_TO_DB
}

func (asset *PlayerAsset) AssetCash_GetLastCheckGlobalMailTs() int64 {
	return asset.AssetCash.LastCheckGlobalMailTs
}

func (asset *PlayerAsset) AssetCash_AddGoldCoin(coin int) {
	if coin > 0 {
		asset.AssetCash.GoldCoin += uint32(coin)
	} else {
		asset.AssetCash.GoldCoin -= uint32(0 - coin)
	}

	asset.AssetCash_DirtyFlag |= DIRTYFLAG_TO_ALL
}

func (asset *PlayerAsset) AssetCash_AddSilverCoin(coin int) {
	if coin > 0 {
		asset.AssetCash.SilverCoin += uint32(coin)
	} else {
		asset.AssetCash.SilverCoin -= uint32(0 - coin)
	}

	asset.AssetCash_DirtyFlag |= DIRTYFLAG_TO_ALL
}

func (asset *PlayerAsset) AssetCash_AddDiamondCoin(coin int) {
	if coin > 0 {
		asset.AssetCash.Diamond += uint32(coin)
	} else {
		asset.AssetCash.Diamond -= uint32(0 - coin)
	}

	asset.AssetCash_DirtyFlag |= DIRTYFLAG_TO_ALL
}
