package g

import (
	"github.com/name5566/leaf/gate"

	"github.com/name5566/leaf/log"
)

const (
	PLAYER_STATUS_OFFLINE = 0
	PLAYER_STATUS_ONLINE  = 1
	PLAYER_STATUS_BATTLE  = 2
)

var GamePlayerManager = make(map[string]*gate.Agent)
var BattlePlayerManager = make(map[string]*gate.Agent)

func AddGamePlayer(clientid string, agent *gate.Agent) {
	exist, ok := GamePlayerManager[clientid]
	if ok {
		(*exist).Close()
		delete(GamePlayerManager, clientid)
	}
	(*agent).SetUserData(clientid)
	GamePlayerManager[clientid] = agent

	log.Debug("AddGamePlayer %v", clientid)
}

func RemoveGamePlayer(clientid string, remoteaddr string) {
	agent, ok := GamePlayerManager[clientid]
	if ok {
		if (*agent).RemoteAddr().String() == remoteaddr {
			delete(GamePlayerManager, clientid)
			log.Debug("RemoveGamePlayer %v", clientid)
		}
	}
}

func AddBattlePlayer(clientid string, agent *gate.Agent) {
	exist, ok := BattlePlayerManager[clientid]
	if ok {
		(*exist).Close()
		delete(BattlePlayerManager, clientid)
	}
	(*agent).SetUserData(clientid)
	BattlePlayerManager[clientid] = agent

	log.Debug("AddBattlePlayer %v", clientid)
}

func RemoveBattlePlayer(clientid string, remoteaddr string) {
	agent, ok := BattlePlayerManager[clientid]
	if ok {
		if (*agent).RemoteAddr().String() == remoteaddr {
			delete(BattlePlayerManager, clientid)
			LeaveRoom(clientid)
			log.Debug("RemoveBattlePlayer %v", clientid)
		}
	}
}