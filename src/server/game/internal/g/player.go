package g

import (
	"github.com/name5566/leaf/gate"
)

const (
	PLAYER_STATUS_OFFLINE = 0
	PLAYER_STATUS_ONLINE  = 1
	PLAYER_STATUS_BATTLE  = 2
)

var PlayerManager = make(map[string]*gate.Agent)
