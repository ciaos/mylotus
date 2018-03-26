package g

import (
	"errors"
	"server/conf"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"strings"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
)

const (
	PLAYER_STATUS_OFFLINE = 0
	PLAYER_STATUS_ONLINE  = 1
	PLAYER_STATUS_BATTLE  = 2
)

//Asset
type FriendAsset_ApplyInfo struct {
	Fromid uint32
	Msg    clientmsg.Req_Friend_Operate
}
type FriendAsset struct {
	CharID    uint32
	Friends   []uint32
	ApplyList []FriendAsset_ApplyInfo
}

//PlayerInfo
type Player struct {
	CharID         uint32
	Charname       string
	Level          uint32
	MatchServerID  int
	BattleServerID int
	OnlineTime     int64
	OfflineTime    int64

	AssetFriend FriendAsset
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

	player.OnlineTime = time.Now().Unix()
	player.OfflineTime = 0

	playerinfo := &PlayerInfo{
		agent:  agent,
		player: player,
	}
	GamePlayerManager[player.CharID] = playerinfo

	log.Debug("AddGamePlayer %v", player.CharID)
}

func RemoveGamePlayer(clientid uint32, remoteaddr string, removenow bool) {
	player, ok := GamePlayerManager[clientid]
	if ok {
		if strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			if removenow {
				delete(GamePlayerManager, clientid)
				log.Debug("RemoveGamePlayer %v", clientid)
			} else {
				player.player.OfflineTime = time.Now().Unix()

				if player.player.MatchServerID > 0 {
					innerReq := &proxymsg.Proxy_GS_MS_Offline{
						Charid: clientid,
					}

					go SendMessageTo(int32(player.player.MatchServerID), conf.Server.MatchServerRename, clientid, uint32(proxymsg.ProxyMessageType_PMT_GS_MS_OFFLINE), innerReq)
				}
			}
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

func (player *PlayerInfo) update(now *time.Time) {
	if player.player.OfflineTime > 0 && (time.Now().Unix()-player.player.OfflineTime > 10) {
		RemoveGamePlayer(player.player.CharID, (*player.agent).RemoteAddr().String(), true)
	}
}

func UpdatePlayerManager(now *time.Time) {
	for _, player := range GamePlayerManager {
		(*player).update(now)
	}
}

func BroadCastMsgToGamePlayers(msgdata interface{}) {
	for _, player := range GamePlayerManager {
		if player.player.OfflineTime == 0 {
			(*player.agent).WriteMsg(msgdata)
		}
	}
}
