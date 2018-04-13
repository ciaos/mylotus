package g

import (
	"time"
)

func (asset *PlayerAsset) AssetCash_AddExp(exp uint32) {
	asset.AssetCash.Exp += exp

	asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_ALL
}

func (asset *PlayerAsset) AssetCash_RefreshLastCheckGlobalMailTs() {
	asset.AssetCash.LastCheckGlobalMailTs = time.Now().Unix()

	asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_DB
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

	asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_ALL
}

func (asset *PlayerAsset) AssetCash_AddSilverCoin(coin int) {
	if coin > 0 {
		asset.AssetCash.SilverCoin += uint32(coin)
	} else {
		asset.AssetCash.SilverCoin -= uint32(0 - coin)
	}

	asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_ALL
}

func (asset *PlayerAsset) AssetCash_AddDiamondCoin(coin int) {
	if coin > 0 {
		asset.AssetCash.Diamond += uint32(coin)
	} else {
		asset.AssetCash.Diamond -= uint32(0 - coin)
	}

	asset.DirtyFlag_AssetCash |= DIRTYFLAG_TO_ALL
}
