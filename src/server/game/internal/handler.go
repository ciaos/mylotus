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

	g.AddGamePlayer(player, &a)
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
		Action:    int32(m.Action),
	}

	var res bool
	skeleton.Go(func() {
		res = g.RandSendMessageTo("matchserver", charid.(uint32), uint32(proxymsg.ProxyMessageType_PMT_GS_MS_MATCH), innerReq)
	}, func() {
		if res == false {
			a.WriteMsg(&clientmsg.Rlt_Match{
				RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
			})
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

	skeleton.Go(func() {
		g.RandSendMessageTo("matchserver", charid.(uint32), uint32(proxymsg.ProxyMessageType_PMT_GS_MS_TEAM_OPERATE), m)
	}, func() {
	})
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
