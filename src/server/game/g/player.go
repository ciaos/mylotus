package g

import (
	"errors"
	"fmt"
	"server/conf"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"server/tool"
	"strings"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

const (
	REASON_DISCONNECT  = 0
	REASON_TIMEOUT     = 1
	REASON_FREE_MEMORY = 2
	REASON_CLEAR       = 3
	REASON_REPLACED    = 4
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

type WaitInfo struct {
	UserID    uint32
	UserAgent *gate.Agent
	LoginTime time.Time
}

var GamePlayerManager = make(map[uint32]*PlayerInfo, 1024)
var BattlePlayerManager = make(map[uint32]*BPlayerInfo, 1024)
var WaitLoginQueue = tool.NewCappedDeque(conf.Server.MaxWaitLoginNum)

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
	c.Update(bson.M{"charid": player.Char.CharID}, player.Char)
}

func AddGamePlayer(player *Player, agent *gate.Agent) {
	exist, ok := GamePlayerManager[player.Char.CharID]
	if ok {
		(*exist.agent).Close()
		_ = exist.agent
		delete(GamePlayerManager, player.Char.CharID)
	}
	(*agent).SetUserData(player.Char.CharID)

	player.Char.UpdateTime = time.Now()
	player.PingTime = time.Now()
	player.OfflineTime = time.Unix(0, 0)

	playerinfo := &PlayerInfo{
		agent:  agent,
		player: player,
	}
	GamePlayerManager[player.Char.CharID] = playerinfo

	log.Debug("AddGamePlayerFromDB %v From %v", player.Char.CharID, (*agent).RemoteAddr().String())
}

func AddCachedGamePlayer(player *Player, agent *gate.Agent) {
	exist, ok := GamePlayerManager[player.Char.CharID]
	if ok {
		(*exist.agent).Close()
		_ = exist.agent
		exist.agent = agent

		exist.player.Char.UpdateTime = time.Now()
		exist.player.PingTime = time.Now()
		exist.player.OfflineTime = time.Unix(0, 0)
	}
	(*agent).SetUserData(player.Char.CharID)

	log.Debug("AddGamePlayerFromCache %v From %v", player.Char.CharID, (*agent).RemoteAddr().String())
}

func ReconnectGamePlayer(charid uint32, agent *gate.Agent) {
	exist, ok := GamePlayerManager[charid]
	if ok {
		log.Debug("ReconnectGamePlayer %v OK From %v Exist %v", charid, (*agent).RemoteAddr().String(), (*exist.agent).RemoteAddr().String())

		(*exist.agent).Close()
		_ = exist.agent
		exist.agent = agent
		(*agent).SetUserData(charid)

		exist.player.Char.UpdateTime = time.Now()
		exist.player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
		exist.player.OfflineTime = time.Unix(0, 0)
	} else {
		log.Error("ReconnectGamePlayer %v Error From %v", charid, (*agent).RemoteAddr().String())
	}
}

func RemoveGamePlayer(clientid uint32, remoteaddr string, reason int32) {
	player, ok := GamePlayerManager[clientid]
	if ok {
		if strings.Compare((*player.agent).RemoteAddr().String(), remoteaddr) == 0 {
			if reason == REASON_FREE_MEMORY {
				player.player.SavePlayerAsset()
				log.Debug("RemoveGamePlayer %v From %v", clientid, remoteaddr)
				delete(GamePlayerManager, clientid)
			} else {
				log.Debug("PreTagGamePlayer %v Reason %v From %v", clientid, reason, remoteaddr)

				if player.agent != nil {
					(*player.agent).Close()
					_ = player.agent
				}

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
	player.OfflineTime = time.Unix(0, 0)

	playerinfo := &BPlayerInfo{
		agent:  agent,
		player: player,
	}
	BattlePlayerManager[player.CharID] = playerinfo

	log.Debug("AddBattlePlayer %v From %v", player.CharID, (*agent).RemoteAddr().String())
}

func ReconnectBattlePlayer(charid uint32, agent *gate.Agent) {
	exist, ok := BattlePlayerManager[charid]
	if ok {
		log.Debug("ReconnectBattlePlayer %v OK From %v Exist %v", charid, (*agent).RemoteAddr().String(), (*exist.agent).RemoteAddr().String())

		(*exist.agent).Close()
		_ = exist.agent
		exist.agent = agent
		exist.player.OnlineTime = time.Now()
		exist.player.HeartBeatTime = time.Now()
		exist.player.IsOffline = false
		exist.player.OfflineTime = time.Unix(0, 0)
		(*agent).SetUserData(charid)
	} else {
		log.Error("ReconnectBattlePlayer %v Error From %v", charid, (*agent).RemoteAddr().String())
	}
}

func RemoveBattlePlayer(clientid uint32, remoteaddr string, reason int32) {
	player, ok := BattlePlayerManager[clientid]
	if ok {
		if reason == REASON_FREE_MEMORY {
			log.Debug("RemoveBattlePlayer %v", clientid)
			delete(BattlePlayerManager, clientid)
			LeaveRoom(clientid)
		} else if reason == REASON_REPLACED {
			log.Debug("RemoveBattlePlayer %v Reason %v", clientid, reason)
			if player.agent != nil {
				(*player.agent).Close()
				_ = player.agent
			}
			delete(BattlePlayerManager, clientid)
		} else {
			log.Debug("PreTagBattlePlayer %v Reason %v From %v Exist %v lastheartbeat %v", clientid, reason, remoteaddr, (*player.agent).RemoteAddr().String(), player.player.HeartBeatTime.Format(TIME_FORMAT))
			if reason == REASON_DISCONNECT && remoteaddr != (*player.agent).RemoteAddr().String() {
				//log.Error("PreTagBattlePlayer Error %v Reason %v From %v Exist %v", clientid, reason, remoteaddr, (*player.agent).RemoteAddr().String())
				return
			}

			player.player.IsOffline = true
			player.player.OfflineTime = time.Now()

			if player.agent != nil {
				(*player.agent).Close()
				_ = player.agent
			}

			if GetMemberRemoteAddr(clientid) == remoteaddr {
				LeaveRoom(clientid)
			}
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
	if player.player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_OFFLINE && (now.Unix()-player.player.OfflineTime.Unix() > 600) {
		RemoveGamePlayer(player.player.Char.CharID, (*player.agent).RemoteAddr().String(), REASON_FREE_MEMORY)
	} else if player.player.GetGamePlayerStatus() != clientmsg.UserStatus_US_PLAYER_OFFLINE && (now.Unix()-player.player.PingTime.Unix() > 120) { //心跳超时转为掉线状态
		RemoveGamePlayer(player.player.Char.CharID, (*player.agent).RemoteAddr().String(), REASON_TIMEOUT)
	}
}

func (player *BPlayerInfo) update(now *time.Time) {
	if player.player.IsOffline == false && now.Unix()-player.player.HeartBeatTime.Unix() > 60 {
		RemoveBattlePlayer(player.player.CharID, "", REASON_TIMEOUT)
	} else if player.player.IsOffline == true && now.Unix()-player.player.OfflineTime.Unix() > 5 {
		RemoveBattlePlayer(player.player.CharID, "", REASON_FREE_MEMORY)
	}
}

func login(req *WaitInfo) {
	log.Release("GamePlayer Begin Login UserID %v From %v", req.UserID, (*req.UserAgent).RemoteAddr().String())

	player := &Player{}
	var isnew bool
	var ret bool
	s := Mongo.Ref()
	defer Mongo.UnRef(s)

	c := s.DB(DB_NAME_GAME).C(TB_NAME_CHARACTER)
	isnew = false
	err := c.Find(bson.M{"userid": req.UserID, "gsid": conf.Server.ServerID}).One(&player.Char)
	if err != nil && err.Error() == "not found" {
		//create new character
		charid, err := Mongo.NextSeq(DB_NAME_GAME, TB_NAME_COUNTER, "counterid")
		if err != nil {
			(*req.UserAgent).WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
			})
			(*req.UserAgent).Close()
			log.Error("handleReqLogin getNextSeq Failed %v", err)
			return
		}

		character := &Character{
			CharID:     uint32(charid),
			UserID:     req.UserID,
			GsId:       int32(conf.Server.ServerID),
			Status:     int32(clientmsg.UserStatus_US_PLAYER_ONLINE),
			CharName:   "",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		err = c.Insert(character)
		if err != nil {
			log.Error("create new character error %v", err)
			(*req.UserAgent).WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
			})
			return
		}

		isnew = true
		player.Char = character
	} else if err != nil {
		log.Error("query character error %v", err)
		(*req.UserAgent).WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		return
	}

	//check if in cache
	cache, _ := GetPlayer(player.Char.CharID)
	if cache != nil {
		if cache.Char.CharName == "" {
			isnew = true
		}
	} else if player.Char.CharName == "" {
		isnew = true
	}

	if cache != nil {
		ret = cache.SyncPlayerAsset()
	} else {
		ret = player.LoadPlayerAsset()
	}
	if ret == true {
		if cache != nil {
			AddCachedGamePlayer(cache, req.UserAgent)
			cache.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
		} else {
			AddGamePlayer(player, req.UserAgent)
			player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
			player.AssetMail_CheckGlobalMail()
		}

		log.Release("GamePlayer End Login UserID %v From %v", req.UserID, (*req.UserAgent).RemoteAddr().String())
	}

	if ret == true {
		(*req.UserAgent).WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode_GRC_OK,
			CharID:         player.Char.CharID,
			IsNewCharacter: isnew,
			CharName:       player.Char.CharName,
		})
	} else {
		(*req.UserAgent).WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		(*req.UserAgent).Close()
		log.Error("load asset Error %v", player.Char.CharID)
	}
}

func updateLogin() {
	for i := 0; i < 1; i++ {
		if WaitLoginQueue.Empty() {
			break
		}

		req := WaitLoginQueue.Shift()
		if req != nil {
			login(req.(*WaitInfo))
		}
	}
}

func UpdateGamePlayerManager(now *time.Time) {
	for _, player := range GamePlayerManager {
		player.update(now)
		player.UpdatePlayerAsset(now)
	}

	updateLogin()
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
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tCharName:%v\tStatus:%v\tAddr:%v\tOnlineTime:%v\tOfflineTime:%v\tPingTime:%v\tMSID:%v\tBSID:%v\t", player.player.Char.CharID, player.player.Char.CharName, player.player.GetGamePlayerStatus(), (*player.agent).RemoteAddr().String(), player.player.Char.UpdateTime.Format(TIME_FORMAT), player.player.OfflineTime.Format(TIME_FORMAT), player.player.PingTime.Format(TIME_FORMAT), player.player.MatchServerID, player.player.BattleServerID)}, "\r\n")
	}
	output = strings.Join([]string{output, fmt.Sprintf("GamePlayerCnt:%d", len(GamePlayerManager))}, "\r\n")
	return strings.TrimLeft(output, "\r\n")
}

func FormatOneGPlayerInfo(charid uint32, assetname string) string {
	output := ""
	player, ok := GamePlayerManager[charid]
	if ok {
		output = fmt.Sprintf("CharID:%v\tCharName:%v\tStatus:%v\tAddr:%v\tOnlineTime:%v\tOfflineTime:%v\tPingTime:%v\tMSID:%v\tBSID:%v\t", player.player.Char.CharID, player.player.Char.CharName, player.player.GetGamePlayerStatus(), (*player.agent).RemoteAddr().String(), player.player.Char.UpdateTime.Format(TIME_FORMAT), player.player.OfflineTime.Format(TIME_FORMAT), player.player.PingTime.Format(TIME_FORMAT), player.player.MatchServerID, player.player.BattleServerID)

		if assetname == "all" || assetname == "friend" {
			output = strings.Join([]string{output, fmt.Sprintf("Assetfriend:\t%v", player.player.Asset.AssetFriend.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "cash" {
			output = strings.Join([]string{output, fmt.Sprintf("Assetcash:\t%v", player.player.Asset.AssetCash.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "item" {
			output = strings.Join([]string{output, fmt.Sprintf("Assetitem:\t%v", player.player.Asset.AssetItem.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "hero" {
			output = strings.Join([]string{output, fmt.Sprintf("Assethero:\t%v", player.player.Asset.AssetHero.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "mail" {
			output = strings.Join([]string{output, fmt.Sprintf("Assetmail:\t%v", player.player.Asset.AssetMail.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "statistic" {
			output = strings.Join([]string{output, fmt.Sprintf("Assetstatistic:\t%v", player.player.Asset.AssetStatistic.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "tutorial" {
			output = strings.Join([]string{output, fmt.Sprintf("Assettutorial:\t%v", player.player.Asset.AssetTutorial.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "task" {
			output = strings.Join([]string{output, fmt.Sprintf("Assettask:\t%v", player.player.Asset.AssetTask.String())}, "\r\n")
		}
		if assetname == "all" || assetname == "achievement" {
			output = strings.Join([]string{output, fmt.Sprintf("Assetachievement:\t%v", player.player.Asset.AssetAchievement.String())}, "\r\n")
		}
	}
	return output
}

func FormatBPlayerInfo() string {
	var output string
	for _, player := range BattlePlayerManager {
		output = strings.Join([]string{output, fmt.Sprintf("CharID:%v\tAddr:%v\tIsOffline:%v\tOnlineTime:%v\tOfflineTime:%v\tHeartBeatTime:%v\tGSID:%v\t", player.player.CharID, (*player.agent).RemoteAddr().String(), player.player.IsOffline, player.player.OnlineTime.Format(TIME_FORMAT), player.player.OfflineTime.Format(TIME_FORMAT), player.player.HeartBeatTime.Format(TIME_FORMAT), player.player.GameServerID)}, "\r\n")
	}
	output = strings.Join([]string{output, fmt.Sprintf("BattlePlayerCnt:%d", len(BattlePlayerManager))}, "\r\n")
	return strings.TrimLeft(output, "\r\n")
}

func FormatOneBPlayerInfo(charid uint32) string {
	output := ""
	player, ok := BattlePlayerManager[charid]
	if ok {
		output = fmt.Sprintf("CharID:%v\tAddr:%v\tIsOffline:%v\tOnlineTime:%v\tOfflineTime:%v\tHeartBeatTime:%v\tGSID:%v\t", player.player.CharID, (*player.agent).RemoteAddr().String(), player.player.IsOffline, player.player.OnlineTime.Format(TIME_FORMAT), player.player.OfflineTime.Format(TIME_FORMAT), player.player.HeartBeatTime.Format(TIME_FORMAT), player.player.GameServerID)
	}
	return output
}
