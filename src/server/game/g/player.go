package g

import (
	"strings"

	"github.com/ciaos/leaf/gate"

	"github.com/ciaos/leaf/log"
)

const (
	PLAYER_STATUS_OFFLINE = 0
	PLAYER_STATUS_ONLINE  = 1
	PLAYER_STATUS_BATTLE  = 2
)

var GamePlayerManager = make(map[uint32]*gate.Agent)
var BattlePlayerManager = make(map[uint32]*gate.Agent)

func AddGamePlayer(clientid uint32, agent *gate.Agent) {
	exist, ok := GamePlayerManager[clientid]
	if ok {
		(*exist).Close()
		delete(GamePlayerManager, clientid)
	}
	(*agent).SetUserData(clientid)
	GamePlayerManager[clientid] = agent

	log.Debug("AddGamePlayer %v", clientid)
}

func RemoveGamePlayer(clientid uint32, remoteaddr string) {
	agent, ok := GamePlayerManager[clientid]
	if ok {
		if strings.Compare((*agent).RemoteAddr().String(), remoteaddr) == 0 {
			delete(GamePlayerManager, clientid)
			log.Debug("RemoveGamePlayer %v", clientid)
		}
	}
}

func AddBattlePlayer(clientid uint32, agent *gate.Agent) {
	exist, ok := BattlePlayerManager[clientid]
	if ok {
		(*exist).Close()
		delete(BattlePlayerManager, clientid)
	}
	(*agent).SetUserData(clientid)
	BattlePlayerManager[clientid] = agent

	log.Debug("AddBattlePlayer %v", clientid)
}

func RemoveBattlePlayer(clientid uint32, remoteaddr string) {
	agent, ok := BattlePlayerManager[clientid]
	if ok {
		if strings.Compare((*agent).RemoteAddr().String(), remoteaddr) == 0 {
			delete(BattlePlayerManager, clientid)
			LeaveRoom(clientid)
			log.Debug("RemoveBattlePlayer %v", clientid)
		}
	}
}
