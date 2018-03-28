package g

import (
	"errors"
	"fmt"
	"server/conf"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"strings"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

type Character struct {
	CharID     uint32
	UserID     uint32
	GsId       int32
	Status     int32
	CharName   string
	CreateTime time.Time
	UpdateTime time.Time
}

//PlayerInfo
type Player struct {
	Char           *Character
	Asset          PlayerAsset
	MatchServerID  int
	BattleServerID int
	OfflineTime    int64
}

type PlayerInfo struct {
	agent  *gate.Agent
	player *Player
}

type BPlayer struct {
	CharID        uint32
	GameServerID  int
	HeartBeatTime int64
	OnlineTime    int64
	OfflineTime   int64
}

type BPlayerInfo struct {
	agent  *gate.Agent
	player *BPlayer
}

var GamePlayerManager = make(map[uint32]*PlayerInfo)
var BattlePlayerManager = make(map[uint32]*BPlayerInfo)

func (player *Player) ChangeGamePlayerStatus(status clientmsg.UserStatus) {
	player.Char.Status = int32(status)
	log.Debug("ChangeGamePlayerStatus GamePlayer %v Status %v", player.Char.CharID, status)
	player.saveGamePlayerCharacterInfo()
}

func (player *Player) GetGamePlayerStatus() clientmsg.UserStatus {
	return clientmsg.UserStatus(player.Char.Status)
}

func (player *Player) saveGamePlayerCharacterInfo() {
	s := Mongo.Ref()
	defer Mongo.UnRef(s)
	c := s.DB(DB_NAME_GAME).C(TB_NAME_CHARACTER)
	c.Update(bson.M{"userid": player.Char.UserID, "gsid": player.Char.GsId}, player.Char)
}

func AddGamePlayer(player *Player, agent *gate.Agent) {
	exist, ok := GamePlayerManager[player.Char.CharID]
	if ok {
		(*exist.agent).Close()
		delete(GamePlayerManager, player.Char.CharID)
	}
	(*agent).SetUserData(player.Char.CharID)

	playerinfo := &PlayerInfo{
		agent:  agent,
		player: player,
	}
	GamePlayerManager[player.Char.CharID] = playerinfo

	log.Debug("AddGamePlayerFromDB %v", player.Char.CharID)
}

func AddCachedGamePlayer(player *Player, agent *gate.Agent) {
	exist, ok := GamePlayerManager[player.Char.CharID]
	if ok {
		(*exist.agent).Close()
		exist.agent = agent
	}
	(*agent).SetUserData(player.Char.CharID)

	log.Debug("AddGamePlayerFromCache %v", player.Char.CharID)
}

func ReconnectGamePlayer(charid uint32, agent *gate.Agent) {
	exist, ok := GamePlayerManager[charid]
	if ok {
		(*exist.agent).Close()
		exist.agent = agent
		(*agent).SetUserData(charid)

		exist.player.Char.UpdateTime = time.Now()
		exist.player.Char.Status = int32(clientmsg.UserStatus_US_PLAYER_ONLINE)

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

				if player.player.Char.Status == int32(clientmsg.UserStatus_US_PLAYER_MATCH) && player.player.MatchServerID > 0 {
					innerReq := &proxymsg.Proxy_GS_MS_Offline{
						Charid: clientid,
					}

					go SendMessageTo(int32(player.player.MatchServerID), conf.Server.MatchServerRename, clientid, proxymsg.ProxyMessageType_PMT_GS_MS_OFFLINE, innerReq)
				}

				player.player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_OFFLINE)
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

	player.OnlineTime = time.Now().Unix()
	player.HeartBeatTime = time.Now().Unix()
	player.OfflineTime = 0

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
		exist.player.OnlineTime = time.Now().Unix()
		exist.player.HeartBeatTime = time.Now().Unix()
		exist.player.OfflineTime = 0
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
	if player.player.Char.Status == int32(clientmsg.UserStatus_US_PLAYER_OFFLINE) && (time.Now().Unix()-player.player.OfflineTime > 600) {
		RemoveGamePlayer(player.player.Char.CharID, (*player.agent).RemoteAddr().String(), true)
	}
}

func (player *BPlayerInfo) update(now *time.Time) {
	if player.player.OfflineTime == 0 && now.Unix()-player.player.HeartBeatTime > 60 {
		player.player.OfflineTime = now.Unix()
		LeaveRoom(player.player.CharID)
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
		player.update(now)
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
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tAddr:%v\tOnlineTime:%v\tOfflineTime:%v\tMSID:%v\tBSID:%v\t", player.player.Char.CharID, player.player.Char.CharName, (*player.agent).RemoteAddr().String(), player.player.Char.UpdateTime, player.player.OfflineTime, player.player.MatchServerID, player.player.BattleServerID)}, "\r\n")
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
