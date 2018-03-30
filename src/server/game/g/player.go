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
	PingTime       time.Time
	OfflineTime    time.Time
}

type PlayerInfo struct {
	agent  *gate.Agent
	player *Player
}

type BPlayer struct {
	CharID        uint32
	GameServerID  int
	HeartBeatTime time.Time
	OnlineTime    time.Time
	OfflineTime   time.Time
	IsOffline     bool
}

type BPlayerInfo struct {
	agent  *gate.Agent
	player *BPlayer
}

var GamePlayerManager = make(map[uint32]*PlayerInfo, 1024)
var BattlePlayerManager = make(map[uint32]*BPlayerInfo, 1024)

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
		_ = exist.agent
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
		_ = exist.agent
		exist.agent = agent
	}
	(*agent).SetUserData(player.Char.CharID)

	log.Debug("AddGamePlayerFromCache %v", player.Char.CharID)
}

func ReconnectGamePlayer(charid uint32, agent *gate.Agent) {
	exist, ok := GamePlayerManager[charid]
	if ok {
		(*exist.agent).Close()
		_ = exist.agent
		exist.agent = agent
		(*agent).SetUserData(charid)

		exist.player.Char.UpdateTime = time.Now()
		exist.player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)

		log.Debug("ReconnectGamePlayer %v OK", charid)
	} else {
		log.Error("ReconnectGamePlayer %v Error", charid)
	}
}

func RemoveGamePlayer(clientid uint32, remoteaddr string, removenow bool) {
	player, ok := GamePlayerManager[clientid]
	if ok {
		if strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			(*player.agent).Close()
			if removenow {
				player.player.SavePlayerAsset()
				delete(GamePlayerManager, clientid)
				log.Debug("RemoveGamePlayer %v", clientid)
			} else {
				player.player.OfflineTime = time.Now()

				if player.player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_MATCH && player.player.MatchServerID > 0 {
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
		_ = exist.agent
		delete(BattlePlayerManager, player.CharID)
	}
	(*agent).SetUserData(player.CharID)

	player.OnlineTime = time.Now()
	player.HeartBeatTime = time.Now()
	player.IsOffline = false

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
		_ = exist.agent
		exist.agent = agent
		exist.player.OnlineTime = time.Now()
		exist.player.HeartBeatTime = time.Now()
		exist.player.IsOffline = false
		(*agent).SetUserData(charid)
		log.Debug("ReconnectBattlePlayer %v OK", charid)
	} else {
		log.Error("ReconnectBattlePlayer %v Error", charid)
	}
}

func RemoveBattlePlayer(clientid uint32, remoteaddr string, force bool) {
	player, ok := BattlePlayerManager[clientid]
	if ok {
		if force == true || strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			(*player.agent).Close()
			_ = player.agent
			delete(BattlePlayerManager, clientid)
			if GetMemberRemoteAddr(clientid) == remoteaddr {
				LeaveRoom(clientid)
			}
			log.Debug("RemoveBattlePlayer %v force %v", clientid, force)
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
	if player.player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_OFFLINE && (time.Now().Unix()-player.player.OfflineTime.Unix() > 600) {
		RemoveGamePlayer(player.player.Char.CharID, (*player.agent).RemoteAddr().String(), true)
	} else if player.player.GetGamePlayerStatus() != clientmsg.UserStatus_US_PLAYER_OFFLINE && now.Unix()-player.player.PingTime.Unix() > 600 { //心跳超时转为掉线状态
		RemoveGamePlayer(player.player.Char.CharID, (*player.agent).RemoteAddr().String(), false)
	}
}

func (player *BPlayerInfo) update(now *time.Time) {
	if player.player.IsOffline == false && now.Unix()-player.player.HeartBeatTime.Unix() > 60 {
		player.player.OfflineTime = *now
		player.player.IsOffline = true
		LeaveRoom(player.player.CharID)
	} else if player.player.IsOffline == true && now.Unix()-player.player.OfflineTime.Unix() > 60 {
		RemoveBattlePlayer(player.player.CharID, "", true)
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
		//to 大厅广播在线非战斗状态的玩家
		if player.player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_ONLINE {
			(*player.agent).WriteMsg(msgdata)
		}
	}
}

func FormatGPlayerInfo() string {
	var output string
	for _, player := range GamePlayerManager {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tStatus:%v\tAddr:%v\tOnlineTime:%v\tOfflineTime:%v\tMSID:%v\tBSID:%v\t", player.player.Char.CharID, player.player.Char.CharName, player.player.GetGamePlayerStatus(), (*player.agent).RemoteAddr().String(), player.player.Char.UpdateTime.Format("2006-01-02 15:04:05"), player.player.OfflineTime.Format("2006-01-02 15:04:05"), player.player.MatchServerID, player.player.BattleServerID)}, "\r\n")
	}
	output = strings.Join([]string{output, fmt.Sprintf("GamePlayerCnt:%d", len(GamePlayerManager))}, "\r\n")
	return strings.TrimLeft(output, "\r\n")
}

func FormatOneGPlayerInfo(charid uint32) string {
	output := ""
	player, ok := GamePlayerManager[charid]
	if ok {
		output = fmt.Sprintf("CharID:%v\tCharName:%v\tStatus:%v\tAddr:%v\tOnlineTime:%v\tOfflineTime:%v\tMSID:%v\tBSID:%v\t", player.player.Char.CharID, player.player.Char.CharName, player.player.GetGamePlayerStatus(), (*player.agent).RemoteAddr().String(), player.player.Char.UpdateTime.Format("2006-01-02 15:04:05"), player.player.OfflineTime.Format("2006-01-02 15:04:05"), player.player.MatchServerID, player.player.BattleServerID)
	}
	return output
}

func FormatBPlayerInfo() string {
	var output string
	for _, player := range BattlePlayerManager {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tAddr:%v\tIsOffline:%v\tOnlineTime:%v\tOfflineTime:%v\tHeartBeatTime:%v\tGSID:%v\t", player.player.CharID, (*player.agent).RemoteAddr().String(), player.player.IsOffline, player.player.OnlineTime.Format("2006-01-02 15:04:05"), player.player.OfflineTime.Format("2006-01-02 15:04:05"), player.player.HeartBeatTime.Format("2006-01-02 15:04:05"), player.player.GameServerID)}, "\r\n")
	}
	output = strings.Join([]string{output, fmt.Sprintf("BattlePlayerCnt:%d", len(BattlePlayerManager))}, "\r\n")
	return strings.TrimLeft(output, "\r\n")
}

func FormatOneBPlayerInfo(charid uint32) string {
	output := ""
	player, ok := BattlePlayerManager[charid]
	if ok {
		output = fmt.Sprintf("CharID:%v\tAddr:%v\tIsOffline:%v\tOnlineTime:%v\tOfflineTime:%v\tHeartBeatTime:%v\tGSID:%v\t", player.player.CharID, (*player.agent).RemoteAddr().String(), player.player.IsOffline, player.player.OnlineTime.Format("2006-01-02 15:04:05"), player.player.OfflineTime.Format("2006-01-02 15:04:05"), player.player.HeartBeatTime.Format("2006-01-02 15:04:05"), player.player.GameServerID)
	}
	return output
}
