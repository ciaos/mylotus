package g

import (
	"errors"
	"strings"

	"github.com/ciaos/leaf/gate"

	"github.com/ciaos/leaf/log"
)

const (
	PLAYER_STATUS_OFFLINE = 0
	PLAYER_STATUS_ONLINE  = 1
	PLAYER_STATUS_BATTLE  = 2
)

type Player struct {
	CharID   uint32
	Charname string
	Level    uint32
}

type PlayerInfo struct {
	agent  *gate.Agent
	player *Player
}

var GamePlayerManager = make(map[uint32]*PlayerInfo)
var BattlePlayerManager = make(map[uint32]*gate.Agent)

func AddGamePlayer(player *Player, agent *gate.Agent) {
	exist, ok := GamePlayerManager[player.CharID]
	if ok {
		(*exist.agent).Close()
		delete(GamePlayerManager, player.CharID)
	}
	(*agent).SetUserData(player.CharID)

	playerinfo := &PlayerInfo{
		agent:  agent,
		player: player,
	}
	GamePlayerManager[player.CharID] = playerinfo

	log.Debug("AddGamePlayer %v", player.CharID)
}

func RemoveGamePlayer(clientid uint32, remoteaddr string) {
	player, ok := GamePlayerManager[clientid]
	if ok {
		if strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			delete(GamePlayerManager, clientid)
			log.Debug("RemoveGamePlayer %v", clientid)
		}
	}
}

func GetPlayer(clientid uint32) (*Player, error) {
	exist, ok := GamePlayerManager[clientid]
	if ok {
		return exist.player, nil
	}
	log.Error("Player Not Found %v", clientid)
	return nil, errors.New("GetPlayer Error")
}

func SendMsgToPlayer(clientid uint32, msgdata interface{}) {
	player, ok := GamePlayerManager[clientid]
	if !ok {
		log.Error("SendMsgToPlayer GamePlayerManager Not Found %v", clientid)
		return
	}

	(*player.agent).WriteMsg(msgdata)
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
