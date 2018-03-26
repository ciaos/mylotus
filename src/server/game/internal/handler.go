package internal

import (
	"encoding/binary"
	"reflect"
	"server/conf"
	"server/game/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"server/tool"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2"
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

	handler(&clientmsg.Req_ConnectBS{}, handleReqConnectBS)
	handler(&clientmsg.Req_EndBattle{}, handleReqEndBattle)
	handler(&clientmsg.Transfer_Loading_Progress{}, handleTransferLoadingProgress)
	handler(&clientmsg.Transfer_Command{}, handleTransferMessage)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func getNextSeq() uint32 {
	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB("game").C("counter")

	doc := struct{ Seq uint32 }{}
	cid := "counterid"

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}},
		Upsert:    true,
		ReturnNew: true,
	}
	if _, err := c.Find(bson.M{"_id": cid}).Apply(change, &doc); err != nil {
		log.Error("getNextSeq counter failed:", err.Error())
		return 0
	}

	return doc.Seq
}

func handlePing(args []interface{}) {
	m := args[0].(*clientmsg.Ping)
	a := args[1].(gate.Agent)

	//log.Error("RecvPing %v From %v ", m.ID, a.RemoteAddr())
	a.WriteMsg(&clientmsg.Pong{ID: m.ID})

	//SendMessageTo(int32(conf.Server.ServerID), conf.Server.ServerType, uint64(1), uint32(0), m)
}

func handleReqServerTime(args []interface{}) {
	//	m := args[0].(*clientmsg.Req_ServerTime)
	a := args[1].(gate.Agent)

	a.WriteMsg(&clientmsg.Rlt_ServerTime{Time: uint32(time.Now().Unix())})
}

func handleReqLogin(args []interface{}) {
	m := args[0].(*clientmsg.Req_Login)
	a := args[1].(gate.Agent)

	if int(m.ServerID) != conf.Server.ServerID {
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		return
	}

	useridBuf, err := tool.DesDecrypt(m.SessionKey, []byte(tool.CRYPT_KEY))
	if err != nil {
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		log.Error("handleReqLogin DesDecrypt Error", err)
		return
	}

	userid := binary.BigEndian.Uint32(useridBuf)
	if userid != m.UserID {
		log.Error("userid != m.UserID ", userid, " ", m.UserID, useridBuf, m.SessionKey)
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		return
	}

	log.Debug("GamePlayer Begin Login UserID %v", userid)

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB("game").C("character")

	player := &g.Player{}

	result := g.Character{}
	err = c.Find(bson.M{"userid": m.UserID, "gsid": conf.Server.ServerID}).One(&result)
	if err != nil {
		//create new character
		charid := getNextSeq()
		if charid == 0 {
			a.WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
			})
			log.Error("handleReqLogin getNextSeq Failed")
			return
		}

		err = c.Insert(&g.Character{
			Id:         bson.NewObjectId(),
			CharId:     charid,
			UserId:     m.UserID,
			GsId:       m.ServerID,
			Status:     g.PLAYER_STATUS_ONLINE,
			CharName:   "",
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		})
		if err != nil {
			log.Error("create new character error %v", err)
			a.WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
			})
			return
		}

		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode_GRC_OK,
			CharID:         charid,
			IsNewCharacter: true,
		})

		player.CharID = charid

	} else {
		c.Update(bson.M{"_id": result.Id}, bson.M{"$set": bson.M{"updatetime": time.Now(), "status": g.PLAYER_STATUS_ONLINE}})

		isnew := false
		if result.CharName == "" {
			isnew = true
		}

		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode_GRC_OK,
			CharID:         result.CharId,
			IsNewCharacter: isnew,
		})

		player.CharID = result.CharId
		player.Charname = result.CharName
	}
	log.Debug("GamePlayer End Login %v", player.CharID)
	g.AddGamePlayer(player, &a)

	//加载资产
	skeleton.Go(func() {
		s1 := g.Mongo.Ref()
		defer g.Mongo.UnRef(s1)

		c := s1.DB("game").C("friendship")
		friendasset := g.FriendAsset{}
		err = c.Find(bson.M{"charid": player.CharID}).One(&friendasset)
		if err != nil && err.Error() == "not found" {
			c.Insert(&g.FriendAsset{
				CharID: player.CharID,
			})
		}
		player.AssetFriend = friendasset
	}, func() {

	})
}

func handleReqSetCharName(args []interface{}) {
	m := args[0].(*clientmsg.Req_SetCharName)
	a := args[1].(gate.Agent)

	charid := a.UserData()
	if charid == nil {
		log.Error("Player SetCharName Login")
		a.Close()
		return
	}

	player, err := g.GetPlayer(charid.(uint32))
	if err != nil {
		a.WriteMsg(&clientmsg.Rlt_SetCharName{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		return
	}

	if m.CharName == "" {
		log.Error("Player %v SetCharName empty", charid)
		a.WriteMsg(&clientmsg.Rlt_SetCharName{
			RetCode: clientmsg.Type_GameRetCode_GRC_NAME_NOT_VALID,
		})
		return
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB("game").C("character")
	c.Update(bson.M{"charid": charid.(uint32)}, bson.M{"$set": bson.M{"charname": m.CharName}})
	a.WriteMsg(&clientmsg.Rlt_SetCharName{
		RetCode: clientmsg.Type_GameRetCode_GRC_OK,
	})

	player.Charname = m.CharName
}

func handleReqMatch(args []interface{}) {
	m := args[0].(*clientmsg.Req_Match)
	a := args[1].(gate.Agent)

	charid := a.UserData()
	if charid == nil {
		log.Error("Player Match Not Login")
		a.WriteMsg(&clientmsg.Rlt_Match{
			RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
		})
		a.Close()
		return
	}

	player, err := g.GetPlayer(charid.(uint32))
	if err != nil {
		a.WriteMsg(&clientmsg.Rlt_Match{
			RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
		})
		return
	}

	innerReq := &proxymsg.Proxy_GS_MS_Match{
		Charid:    charid.(uint32),
		Charname:  player.Charname,
		Matchmode: int32(m.Mode),
		Mapid:     m.MapID,
		Action:    int32(m.Action),
	}

	var msid int
	skeleton.Go(func() {
		if m.Action == clientmsg.MatchActionType_MAT_JOIN {
			msid, _ = g.RandSendMessageTo("matchserver", charid.(uint32), uint32(proxymsg.ProxyMessageType_PMT_GS_MS_MATCH), innerReq)
		} else {
			g.SendMessageTo(int32(player.MatchServerID), conf.Server.MatchServerRename, charid.(uint32), uint32(proxymsg.ProxyMessageType_PMT_GS_MS_MATCH), innerReq)
		}
	}, func() {
		if m.Action == clientmsg.MatchActionType_MAT_JOIN {
			player.MatchServerID = msid
		} else if m.Action == clientmsg.MatchActionType_MAT_CANCEL {
			player.MatchServerID = 0
		}
	})
}

func handleTransferTeamOperate(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Team_Operate)
	a := args[1].(gate.Agent)

	charid := a.UserData()
	if charid == nil {
		log.Error("Player TeamOperate Not Login")
		return
	}

	//log.Debug("handleTransferTeamOperate %v %v %v %v", charid, m.Action, m.CharID, m.CharType)
	player, err := g.GetPlayer(charid.(uint32))
	if err != nil {
		log.Error("PlayerInfo Not Found %v", charid.(uint32))
		return
	}

	if player.MatchServerID > 0 {
		go g.SendMessageTo(int32(player.MatchServerID), conf.Server.MatchServerRename, charid.(uint32), uint32(proxymsg.ProxyMessageType_PMT_GS_MS_TEAM_OPERATE), m)
	}
}

func handleReqFriendOperate(args []interface{}) {
	m := args[0].(*clientmsg.Req_Friend_Operate)
	a := args[1].(gate.Agent)

	charid := a.UserData()
	if charid == nil {
		log.Error("Player ReqFriendOperate Not Login")
		return
	}

	rsp := &clientmsg.Rlt_Friend_Operate{
		Action:  m.Action,
		RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	if m.Action == clientmsg.FriendOperateActionType_FOAT_SEARCH {
		c := s.DB("game").C("character")
		results := []g.Character{}
		err := c.Find(bson.M{"charname": bson.M{"$regex": bson.RegEx{m.SearchName, "i"}}}).Select(bson.M{"charid": 1}).Limit(10).All(&results)
		if err == nil {
			for _, result := range results {
				rsp.SearchedCharIDs = append(rsp.SearchedCharIDs, result.CharId)
			}
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
		}
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_ADD_FRIEND {
		c := s.DB("game").C("friendship")

		exist, _ := c.Find(bson.M{"charid": m.OperateCharID, "applylist.fromid": charid.(uint32)}).Count()
		if exist == 0 {
			err := c.Update(bson.M{"charid": m.OperateCharID}, bson.M{"$push": bson.M{
				"applylist": bson.M{"fromid": charid.(uint32), "msg": m},
			}})
			if err == nil {
				rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
			} else {
				log.Error("FriendOperateActionType_FOAT_ADD_FRIEND Error %v", err)
			}
		} else {
			rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
		}
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_DEL_FRIEND {
		c := s.DB("game").C("friendship")
		c.Update(bson.M{"charid": charid.(uint32)}, bson.M{"$pull": bson.M{
			"friends": m.OperateCharID,
		}})
		c.Update(bson.M{"charid": m.OperateCharID}, bson.M{"$pull": bson.M{
			"friends": charid.(uint32),
		}})
		rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
	} else if m.Action == clientmsg.FriendOperateActionType_FOAT_ACCEPT {
		c := s.DB("game").C("friendship")
		err := c.Update(bson.M{"charid": charid.(uint32)}, bson.M{"$pull": bson.M{
			"applylist": bson.M{"fromid": m.OperateCharID},
		}})
		if err != nil {
			log.Error("FriendOperateActionType_FOAT_ACCEPT Error %v", err)
		}
		exist, _ := c.Find(bson.M{"charid": charid.(uint32), "friends": m.OperateCharID}).Count()
		if exist == 0 {
			c.Update(bson.M{"charid": charid.(uint32)}, bson.M{"$push": bson.M{
				"friends": m.OperateCharID,
			}})
		}
		exist, _ = c.Find(bson.M{"charid": m.OperateCharID, "friends": charid.(uint32)}).Count()
		if exist == 0 {
			c.Update(bson.M{"charid": m.OperateCharID}, bson.M{"$push": bson.M{
				"friends": charid.(uint32),
			}})
		}
		rsp.RetCode = clientmsg.Type_GameRetCode_GRC_OK
	}
	a.WriteMsg(rsp)
}

func handleReqChat(args []interface{}) {
	m := args[0].(*clientmsg.Req_Chat)
	a := args[1].(gate.Agent)

	charid := a.UserData()
	if charid == nil {
		log.Error("Player ReqChat Not Login")
		return
	}

	if m.Channel == clientmsg.ChatChannelType_CCT_WORLD {
		rsp := &clientmsg.Rlt_Chat{
			Channel:     m.Channel,
			TargetID:    m.TargetID,
			MessageType: m.MessageType,
			MessageData: m.MessageData,
			SenderID:    charid.(uint32),
		}
		g.BroadCastMsgToGamePlayers(rsp)
	}
}

func handleReqQueryCharInfo(args []interface{}) {
	m := args[0].(*clientmsg.Req_QueryCharInfo)
	a := args[1].(gate.Agent)

	charid := a.UserData()
	if charid == nil {
		log.Error("Player ReqQueryCharInfo Not Login")
		return
	}

	rsp := &clientmsg.Rlt_QueryCharInfo{}
	if len(m.CharIDs) <= 0 || len(m.CharIDs) > 10 {
		log.Error("handleReqQueryCharInfo Too Many %v", len(m.CharIDs))
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

	c := s.DB("game").C("friendship")
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
			CharID:   result.CharId,
			CharName: result.CharName,
			Level:    0,
		}
		rsp.UserInfo = append(rsp.UserInfo, userinfo)
	}
	a.WriteMsg(rsp)
}

func handleReqMakeTeamOperate(args []interface{}) {
}

func handleTransferLoadingProgress(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Loading_Progress)
	a := args[1].(gate.Agent)

	if a.UserData() != nil {
		g.LoadingRoom(a.UserData().(uint32), m)
	}
}

func handleReqConnectBS(args []interface{}) {
	m := args[0].(*clientmsg.Req_ConnectBS)
	a := args[1].(gate.Agent)

	if g.ConnectRoom(m.CharID, m.RoomID, m.BattleKey) {
		g.AddBattlePlayer(m.CharID, &a)
		rsp := g.GenRoomInfoPB(m.RoomID)
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

	g.EndBattle(m.CharID)

	log.Debug("handleReqEndBattle %v", m.CharID)

	a.WriteMsg(&clientmsg.Rlt_EndBattle{
		RetCode: clientmsg.Type_BattleRetCode_BRC_OK,
		CharID:  m.CharID,
	})
}

func handleTransferMessage(args []interface{}) {
	m := args[0].(*clientmsg.Transfer_Command)
	a := args[1].(gate.Agent)
	if a.UserData() != nil {
		charid := a.UserData().(uint32)
		g.AddMessage(charid, m)
	}
}
