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
