package g

import (
	"errors"
	"fmt"
	"server/conf"
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
	PLAYER_STATUS_LOGIN   = 3
)

//PlayerInfo
type Player struct {
	UserID         uint32
	CharID         uint32
	Charname       string
	Level          uint32
	MatchServerID  int
	BattleServerID int
	OnlineTime     int64
	OfflineTime    int64
	Status         int
	Asset          PlayerAsset
}

type PlayerInfo struct {
	agent  *gate.Agent
	player *Player
}

type BPlayer struct {
	CharID       uint32
	GameServerID int
	UpdateTime   int64
}

type BPlayerInfo struct {
	agent  *gate.Agent
	player *BPlayer
}

var GamePlayerManager = make(map[uint32]*PlayerInfo)
var BattlePlayerManager = make(map[uint32]*BPlayerInfo)

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

func ReconnectGamePlayer(charid uint32, agent *gate.Agent) {
	exist, ok := GamePlayerManager[charid]
	if ok {
		(*exist.agent).Close()
		exist.agent = agent
		(*agent).SetUserData(charid)
		log.Debug("ReconnectGamePlayer %v OK", charid)
	} else {
		log.Error("ReconnectGamePlayer %v Error", charid)
	}
}

func RemoveGamePlayer(clientid uint32, remoteaddr string, removenow bool) {
	player, ok := GamePlayerManager[clientid]
	if ok {
		if strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			if removenow {
				player.player.SavePlayerAsset()
				delete(GamePlayerManager, clientid)
				log.Debug("RemoveGamePlayer %v", clientid)
			} else {
				player.player.OfflineTime = time.Now().Unix()

				if player.player.MatchServerID > 0 {
					innerReq := &proxymsg.Proxy_GS_MS_Offline{
						Charid: clientid,
					}

					go SendMessageTo(int32(player.player.MatchServerID), conf.Server.MatchServerRename, clientid, proxymsg.ProxyMessageType_PMT_GS_MS_OFFLINE, innerReq)
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

func AddBattlePlayer(player *BPlayer, agent *gate.Agent) {
	exist, ok := BattlePlayerManager[player.CharID]
	if ok {
		(*exist.agent).Close()
		delete(BattlePlayerManager, player.CharID)
	}
	(*agent).SetUserData(player.CharID)

	playerinfo := &BPlayerInfo{
		agent:  agent,
		player: player,
	}
	BattlePlayerManager[player.CharID] = playerinfo

	log.Debug("AddBattlePlayer %v", player.CharID)
}

func ReconnectBattlePlayer(charid uint32, agent *gate.Agent) {
	exist, ok := BattlePlayerManager[charid]
	if ok {
		(*exist.agent).Close()
		exist.agent = agent
		exist.player.UpdateTime = time.Now().Unix()
		(*agent).SetUserData(charid)
		log.Debug("ReconnectBattlePlayer %v OK", charid)
	} else {
		log.Error("ReconnectBattlePlayer %v Error", charid)
	}
}

func RemoveBattlePlayer(clientid uint32, remoteaddr string) {
	player, ok := BattlePlayerManager[clientid]
	if ok {
		if strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			delete(BattlePlayerManager, clientid)
			LeaveRoom(clientid)
			log.Debug("RemoveBattlePlayer %v", clientid)
		}
	}
}

func GetBattlePlayer(clientid uint32) (*BPlayer, error) {
	exist, ok := BattlePlayerManager[clientid]
	if ok {
		return exist.player, nil
	}
	return nil, errors.New("GetPlayer Error")
}

func (player *PlayerInfo) update(now *time.Time) {
	if player.player.OfflineTime > 0 && (time.Now().Unix()-player.player.OfflineTime > 10) {
		RemoveGamePlayer(player.player.CharID, (*player.agent).RemoteAddr().String(), true)
	}
}

func UpdateGamePlayerManager(now *time.Time) {
	for _, player := range GamePlayerManager {
		player.update(now)
		player.UpdatePlayerAsset(now)
	}
}

func UpdateBattlePlayerManager(now *time.Time) {
	for _, player := range BattlePlayerManager {
		if now.Unix()-player.player.UpdateTime > 60 {
			player.player.UpdateTime = now.Unix()
			LeaveRoom(player.player.CharID)
		}
	}
}

func BroadCastMsgToGamePlayers(msgdata interface{}) {
	for _, player := range GamePlayerManager {
		if player.player.OfflineTime == 0 {
			(*player.agent).WriteMsg(msgdata)
		}
	}
}

func FormatGPlayerInfo() string {
	output := fmt.Sprintf("GamePlayerCnt:%d", len(GamePlayerManager))
	for _, player := range GamePlayerManager {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tAddr:%v\tOnlineTime:%v\tOfflineTime:%v\tMSID:%v\tBSID:%v\t", player.player.CharID, player.player.Charname, (*player.agent).RemoteAddr().String(), player.player.OnlineTime, player.player.OfflineTime, player.player.MatchServerID, player.player.BattleServerID)}, "\r\n")
	}
	return output
}

func FormatBPlayerInfo() string {
	output := fmt.Sprintf("BattlePlayerCnt:%d", len(BattlePlayerManager))
	for _, player := range BattlePlayerManager {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tGameServerID:%v\tAddr:%v\t", player.player.CharID, player.player.GameServerID, (*player.agent).RemoteAddr().String())}, "\r\n")
	}
	return output
}
