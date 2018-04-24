package internal

import (
	"encoding/binary"
	"strings"
	//"math/rand"
	"reflect"
	"server/conf"
	"server/game/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"server/tool"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	handler(&clientmsg.Ping{}, handlePing)

	handler(&clientmsg.Req_ServerTime{}, handleReqServerTime)
	handler(&clientmsg.Req_Login{}, handleReqLogin)
	handler(&clientmsg.Req_SetCharName{}, handleReqSetCharName)
	handler(&clientmsg.Req_Match{}, handleReqMatch)
	handler(&clientmsg.Transfer_Team_Operate{}, handleTransferTeamOperate)
	handler(&clientmsg.Req_Friend_Operate{}, handleReqFriendOperate)
	handler(&clientmsg.Req_Chat{}, handleReqChat)
	handler(&clientmsg.Req_QueryCharInfo{}, handleReqQueryCharInfo)
	handler(&clientmsg.Req_MakeTeamOperate{}, handleReqMakeTeamOperate)
	handler(&clientmsg.Req_Mail_Action{}, handleReqMailAction)
	handler(&clientmsg.Req_Re_ConnectGS{}, handleReqReConnectGS)
	handler(&clientmsg.Req_GM_Command{}, handleReqGMCommand)

	handler(&clientmsg.Req_ConnectBS{}, handleReqConnectBS)
	handler(&clientmsg.Req_EndBattle{}, handleReqEndBattle)
	handler(&clientmsg.Transfer_Loading_Progress{}, handleTransferLoadingProgress)
	handler(&clientmsg.Transfer_Command{}, handleTransferCommand)
	handler(&clientmsg.Transfer_Battle_Message{}, handleTransferBattleMessage)
	handler(&clientmsg.Req_Re_ConnectBS{}, handleReqReConnectBS)
	handler(&clientmsg.Transfer_Battle_Heartbeat{}, handleReqBattleHeartBeat)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func getNextSeq() (int, error) {
	return g.Mongo.NextSeq(g.DB_NAME_GAME, g.TB_NAME_COUNTER, "counterid")
}

func getGSPlayer(a *gate.Agent) *g.Player {
	charid := (*a).UserData()
	if charid != nil {
		player, err := g.GetPlayer(charid.(uint32))
		if err == nil {
			return player
		}
	}
	return nil
}

func getBSPlayer(a *gate.Agent) *g.BPlayer {
	charid := (*a).UserData()
	if charid != nil {
		player, err := g.GetBattlePlayer(charid.(uint32))
		if err == nil {
			return player
		}
	}
	return nil
}

func handlePing(args []interface{}) {
	m := args[0].(*clientmsg.Ping)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player != nil {
		player.PingTime = time.Now()
	}

	a.WriteMsg(&clientmsg.Pong{ID: m.ID})
}

func handleReqServerTime(args []interface{}) {
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player != nil {
		player.PingTime = time.Now()
	}

	a.WriteMsg(&clientmsg.Rlt_ServerTime{Time: uint64(time.Now().Unix())})
}

func handleReqLogin(args []interface{}) {
	m := args[0].(*clientmsg.Req_Login)
	a := args[1].(gate.Agent)

	if len(g.GamePlayerManager) >= conf.Server.MaxOnlineNum {
		log.Error("Server Online Full")
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_ONLINE_TOO_MANY,
		})
		a.Close()
		return
	}

	if g.WaitLoginQueue.Full() {
		log.Error("Server WaitLogin Full")
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_LOGIN_TOO_MANY,
		})
		a.Close()
		return
	} else {
		if !g.WaitLoginQueue.Empty() {
			a.WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_LOGIN_LINE_UP,
			})
		}
	}

	useridBuf, err := tool.DesDecrypt(m.SessionKey, []byte(tool.CRYPT_KEY))
	if a.UserData() != nil || int(m.ServerID) != conf.Server.ServerID || err != nil {
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		a.Close()
		log.Error("handleReqLogin a.UserData() %v ServerID %v Error %v", a.UserData(), m.ServerID, err)
		return
	}

	userid := binary.BigEndian.Uint32(useridBuf)
	if userid != m.UserID {
		log.Error("userid %v != m.UserID %v ", userid, m.UserID)
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		a.Close()
		return
	}

	req := &g.WaitInfo{
		UserID:    userid,
		UserAgent: &a,
		LoginTime: time.Now(),
	}

	g.WaitLoginQueue.Append(req)
}

func handleReqReConnectGS(args []interface{}) {
	m := args[0].(*clientmsg.Req_Re_ConnectGS)
	a := args[1].(gate.Agent)

	if a.UserData() != nil {
		a.Close()
		return
	}

	useridBuf, err := tool.DesDecrypt(m.SessionKey, []byte(tool.CRYPT_KEY))
	if err != nil {
		a.WriteMsg(&clientmsg.Rlt_Re_ConnectGS{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		a.Close()
		log.Error("handleReqReConnectGS DesDecrypt Error", err)
		return
	}

	userid := binary.BigEndian.Uint32(useridBuf)
	player, err := g.GetPlayer(m.CharID)
	if err != nil || player.Char.UserID != userid || player.Char.Status != int32(clientmsg.UserStatus_US_PLAYER_OFFLINE) {
		a.WriteMsg(&clientmsg.Rlt_Re_ConnectGS{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		a.Close()
		return
	}

	g.ReconnectGamePlayer(m.CharID, &a)
	g.SendMsgToPlayer(m.CharID, &clientmsg.Rlt_Re_ConnectGS{
		RetCode: clientmsg.Type_GameRetCode_GRC_OK,
	})

	if player.MatchServerID != 0 {
		innerReq := &proxymsg.Proxy_GS_MS_Reconnect{
			Charid: m.CharID,
		}
		g.SendMessageTo(int32(player.MatchServerID), conf.Server.MatchServerRename, m.CharID, proxymsg.ProxyMessageType_PMT_GS_MS_RECONNECT, innerReq)
	}
}

func handleReqSetCharName(args []interface{}) {
	m := args[0].(*clientmsg.Req_SetCharName)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.WriteMsg(&clientmsg.Rlt_SetCharName{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		a.Close()
		return
	}

	if m.CharName == "" {
		log.Error("Player %v SetCharName empty", player.Char.CharID)
		a.WriteMsg(&clientmsg.Rlt_SetCharName{
			RetCode: clientmsg.Type_GameRetCode_GRC_NAME_NOT_VALID,
		})
		return
	}

	a.WriteMsg(&clientmsg.Rlt_SetCharName{
		RetCode:     clientmsg.Type_GameRetCode_GRC_OK,
		CharName:    m.CharName,
		IsNewCreate: m.IsNewCreate,
	})

	player.Char.CharName = m.CharName
}

func handleReqMatch(args []interface{}) {
	m := args[0].(*clientmsg.Req_Match)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.WriteMsg(&clientmsg.Rlt_Match{
			RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
		})
		a.Close()
		return
	}

	innerReq := &proxymsg.Proxy_GS_MS_Match{
		Charid:    player.Char.CharID,
		Charname:  player.Char.CharName,
		Matchmode: int32(m.Mode),
		Mapid:     m.MapID,
		Action:    int32(m.Action),
	}

	var msid int
	if m.Action == clientmsg.MatchActionType_MAT_JOIN {
		if player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_ONLINE { //防止多次点击匹配
			player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_MATCH)
			msid, _ = g.RandSendMessageTo("matchserver", player.Char.CharID, proxymsg.ProxyMessageType_PMT_GS_MS_MATCH, innerReq)
		} else {
			log.Error("Invalid Status %v When Match CharID %v", player.GetGamePlayerStatus(), player.Char.CharID)
			a.WriteMsg(&clientmsg.Rlt_Match{
				RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
			})
			return
		}

		player.MatchServerID = msid
	} else {
		if player.GetGamePlayerStatus() == clientmsg.UserStatus_US_PLAYER_MATCH {
			g.SendMessageTo(int32(player.MatchServerID), conf.Server.MatchServerRename, player.Char.CharID, proxymsg.ProxyMessageType_PMT_GS_MS_MATCH, innerReq)

			if m.Action == clientmsg.MatchActionType_MAT_CANCEL {
				player.ChangeGamePlayerStatus(clientmsg.UserStatus_US_PLAYER_ONLINE)
				player.MatchServerID = 0

				a.WriteMsg(&clientmsg.Rlt_Match{
					RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_CANCELED,
				})
			}
		}
	}
}

func handleTransferTeamOperate(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Team_Operate)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	if player.MatchServerID > 0 {
		go g.SendMessageTo(int32(player.MatchServerID), conf.Server.MatchServerRename, player.Char.CharID, proxymsg.ProxyMessageType_PMT_GS_MS_TEAM_OPERATE, m)
	} else {
		log.Error("handleTransferTeamOperate CharID %v Invalid MatchServerID %v", player.Char.CharID, player.MatchServerID)
	}
}

func handleReqFriendOperate(args []interface{}) {
	m := args[0].(*clientmsg.Req_Friend_Operate)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	rsp := &clientmsg.Rlt_Friend_Operate{
		Action:  m.Action,
		RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	if m.Action == clientmsg.FriendOperateActionType_FOAT_SEARCH {
		c := s.DB(g.DB_NAME_GAME).C(g.TB_NAME_CHARACTER)
		results := []g.Character{}
		err := c.Find(bson.M{"charname": bson.M{"$regex": bson.RegEx{m.SearchName, "i"}}}).Select(bson.M{"charid": 1}).Limit(10).All(&results)
		if err == nil {
			for _, result := range results {
				rsp.SearchedCharIDs = append(rsp.SearchedCharIDs, result.CharID)
			}
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
		}
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_ADD_FRIEND {
		c := s.DB(g.DB_NAME_GAME).C(g.TB_NAME_CHARACTER)
		character := &g.Character{}
		err := c.Find(bson.M{"charid": m.OperateCharID}).Select(bson.M{"gsid": 1}).One(character)
		if err == nil {
			go g.SendMessageTo(character.GsId, conf.Server.GameServerRename, player.Char.CharID, proxymsg.ProxyMessageType_PMT_GS_GS_FRIEND_OPERATE, m)
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
		}
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_DEL_FRIEND {
		player.GetPlayerAsset().AssetFriend_DelFriend(player.Char.CharID, m.OperateCharID)

		c := s.DB(g.DB_NAME_GAME).C(g.TB_NAME_CHARACTER)
		character := &g.Character{}
		err := c.Find(bson.M{"charid": m.OperateCharID}).Select(bson.M{"gsid": 1}).One(character)
		if err == nil {
			go g.SendMessageTo(character.GsId, conf.Server.GameServerRename, player.Char.CharID, proxymsg.ProxyMessageType_PMT_GS_GS_FRIEND_OPERATE, m)
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
		}
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_ACCEPT {
		player.GetPlayerAsset().AssetFriend_AcceptApplyInfo(player.Char.CharID, m.OperateCharID)

		c := s.DB(g.DB_NAME_GAME).C(g.TB_NAME_CHARACTER)
		character := &g.Character{}
		err := c.Find(bson.M{"charid": m.OperateCharID}).Select(bson.M{"gsid": 1}).One(character)
		if err == nil {
			go g.SendMessageTo(character.GsId, conf.Server.GameServerRename, player.Char.CharID, proxymsg.ProxyMessageType_PMT_GS_GS_FRIEND_OPERATE, m)
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
		}
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_REJECT {
		player.GetPlayerAsset().AssetFriend_RejectApplyInfo(m.OperateCharID)
		rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
	}
	a.WriteMsg(rsp)
}

func handleReqChat(args []interface{}) {
	m := args[0].(*clientmsg.Req_Chat)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	if m.Channel == clientmsg.ChatChannelType_CCT_WORLD {
		rsp := &clientmsg.Rlt_Chat{
			Channel:     m.Channel,
			TargetID:    m.TargetID,
			MessageType: m.MessageType,
			MessageData: m.MessageData,
			SenderID:    player.Char.CharID,
		}
		g.BroadCastMsgToGamePlayers(rsp)
	}
}

func handleReqQueryCharInfo(args []interface{}) {
	m := args[0].(*clientmsg.Req_QueryCharInfo)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	rsp := &clientmsg.Rlt_QueryCharInfo{}
	if len(m.CharIDs) <= 0 || len(m.CharIDs) > 10 {
		rsp.RetCode = clientmsg.Type_GameRetCode_GRC_QUERY_TOO_MANY
		a.WriteMsg(rsp)
		return
	}

	charids := make([]uint32, len(m.CharIDs))
	for i, charid := range m.CharIDs {
		charids[i] = charid
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB(g.DB_NAME_GAME).C(g.TB_NAME_CHARACTER)
	results := []g.Character{}
	err := c.Find(bson.M{"charid": bson.M{"$in": charids}}).All(&results)
	if err != nil {
		log.Error("handleReqQueryCharInfo %v Error %v", charids, err)
		rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OTHER
		a.WriteMsg(rsp)
		return
	}

	rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
	for _, result := range results {
		userinfo := &clientmsg.Rlt_QueryCharInfo_UserBasicInfo{
			CharID:   result.CharID,
			CharName: result.CharName,
			Status:   clientmsg.UserStatus(result.Status),
		}
		rsp.UserInfo = append(rsp.UserInfo, userinfo)
	}
	a.WriteMsg(rsp)
}

func handleReqMakeTeamOperate(args []interface{}) {
}

func handleReqMailAction(args []interface{}) {
	m := args[0].(*clientmsg.Req_Mail_Action)
	a := args[1].(gate.Agent)

	player := getGSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	rsp := player.GetPlayerAsset().AssetMail_Action(m)
	if rsp != nil {
		a.WriteMsg(rsp)
	}
}

func handleTransferLoadingProgress(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Loading_Progress)
	a := args[1].(gate.Agent)

	player := getBSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	g.LoadingRoom(player.CharID, m)
}

func handleReqConnectBS(args []interface{}) {
	m := args[0].(*clientmsg.Req_ConnectBS)
	a := args[1].(gate.Agent)

	if a.UserData() != nil { //防止多次发送
		log.Error("handleReqConnectBS Exist Connection")
		a.Close()
		return
	}

	if g.ConnectRoom(m.CharID, m.RoomID, m.BattleKey, a.RemoteAddr().String()) {
		player := &g.BPlayer{
			CharID:        m.CharID,
			GameServerID:  int(g.GetMemberGSID(m.CharID)),
			HeartBeatTime: time.Now(),
		}
		g.AddBattlePlayer(player, &a)
		rsp := g.GenRoomInfoPB(m.CharID, false)
		a.WriteMsg(rsp)
	} else {
		a.WriteMsg(&clientmsg.Rlt_ConnectBS{
			RetCode: clientmsg.Type_BattleRetCode_BRC_OTHER,
		})
	}
}

func handleReqEndBattle(args []interface{}) {
	m := args[0].(*clientmsg.Req_EndBattle)
	a := args[1].(gate.Agent)

	player := getBSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	log.Debug("handleReqEndBattle CharID %v PlayerID %v", player.CharID, m.CharID)
	a.WriteMsg(&clientmsg.Rlt_EndBattle{
		RetCode: clientmsg.Type_BattleRetCode_BRC_OK,
		CharID:  m.CharID,
	})

	g.EndBattle(player.CharID)
}

func handleTransferCommand(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Command)
	a := args[1].(gate.Agent)

	player := getBSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}
	g.AddMessage(player.CharID, m)
}

func handleTransferBattleMessage(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Battle_Message)
	a := args[1].(gate.Agent)

	player := getBSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}
	g.TransferRoomMessage(player.CharID, m)
}

func handleReqBattleHeartBeat(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Battle_Heartbeat)
	a := args[1].(gate.Agent)

	player := getBSPlayer(&a)
	if player == nil {
		a.Close()
		return
	}

	player.HeartBeatTime = time.Now()
	//time.Sleep(time.Duration(rand.Intn(100)) * time.Microsecond)
	a.WriteMsg(m)
}

func handleReqReConnectBS(args []interface{}) {
	m := args[0].(*clientmsg.Req_Re_ConnectBS)
	a := args[1].(gate.Agent)

	if a.UserData() != nil {
		a.Close()
		return
	}

	if g.ReConnectRoom(m.CharID, m.FrameID, m.BattleKey, a.RemoteAddr().String()) {
		g.ReconnectBattlePlayer(m.CharID, &a)
		rsp := g.GenRoomInfoPB(m.CharID, true)
		a.WriteMsg(rsp)
	} else {
		a.WriteMsg(&clientmsg.Rlt_ConnectBS{
			RetCode: clientmsg.Type_BattleRetCode_BRC_OTHER,
		})
	}
}

func handleReqGMCommand(args []interface{}) {
	m := args[0].(*clientmsg.Req_GM_Command)
	a := args[1].(gate.Agent)

	if a.UserData() == nil {
		a.Close()
		return
	}

	sCmd := strings.Split(m.Command, " ")

	cmd := make([]interface{}, len(sCmd))
	for i, c := range sCmd {
		cmd[i] = c
	}

	result := RunGMCmd(cmd)
	rsp := &clientmsg.Rlt_GM_Command{
		Result: result.(string),
	}
	a.WriteMsg(rsp)
}
