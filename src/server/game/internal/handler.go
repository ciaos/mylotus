package internal

import (
	"fmt"
	"reflect"
	"server/conf"
	"server/game/g"
	"server/msg/clientmsg"
	"server/msg/proxymsg"
	"server/tool"
	"strings"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	handler(&clientmsg.Ping{}, handlePing)
	handler(&clientmsg.Req_ServerTime{}, handleReqServerTime)
	handler(&clientmsg.Req_Login{}, handleReqLogin)
	handler(&clientmsg.Req_Match{}, handleReqMatch)
	handler(&clientmsg.Req_ConnectBS{}, handleReqConnectBS)
	handler(&clientmsg.Req_EndBattle{}, handleReqEndBattle)

	handler(&clientmsg.Transfer_Command{}, handleTransferMessage)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handlePing(args []interface{}) {
	m := args[0].(*clientmsg.Ping)
	a := args[1].(gate.Agent)

	//log.Debug("RecvPing %v From %v ", m.GetID(), a.RemoteAddr())
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

	userid, err := tool.DesDecrypt(m.SessionKey, []byte(tool.CRYPT_KEY))
	if err != nil {
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		return
	}

	if strings.Compare(string(userid), m.UserID) != 0 {
		log.Error("strings.Compare(string(userid), m.GetUserID()) != 0")
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
		})
		return
	}

	s := g.Mongo.Ref()
	defer g.Mongo.UnRef(s)

	c := s.DB("game").C("character")

	var pcharid string
	result := g.Character{}
	err = c.Find(bson.M{"userid": m.UserID, "gsid": conf.Server.ServerID}).One(&result)
	if err != nil {
		//create new character
		charid := bson.NewObjectId()
		err = c.Insert(&g.Character{
			Id:         charid,
			UserId:     m.UserID,
			GsId:       m.ServerID,
			Status:     g.PLAYER_STATUS_ONLINE,
			CreateTime: time.Now(),
		})
		if err != nil {
			log.Error("create new character error %v", err)
			a.WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
			})
			return
		}

		c = s.DB("game").C(fmt.Sprintf("userinfo_%d", m.ServerID))
		err = c.Insert(&g.UserInfo{
			CharId:   charid.String(),
			CharName: "",
			Level:    1,
		})
		if err != nil {
			log.Error("create new userinfo error %v", err)
			a.WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode_GRC_OTHER,
			})
			return
		}

		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode_GRC_NONE,
			CharID:         charid.String(),
			IsNewCharacter: true,
		})

		pcharid = charid.String()

	} else {
		c.Update(bson.M{"_id": result.Id}, bson.M{"$set": bson.M{"updatetime": time.Now(), "status": g.PLAYER_STATUS_ONLINE}})

		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode_GRC_NONE,
			CharID:         result.Id.String(),
			IsNewCharacter: false,
		})

		pcharid = result.Id.String()
	}

	g.AddGamePlayer(pcharid, &a)
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

		return
	}

	innerReq := &proxymsg.Proxy_GS_MS_Match{
		Charid:    charid.(string),
		Matchmode: int32(m.Mode),
		Action:    int32(m.Action),
	}

	var res bool
	skeleton.Go(func() {
		res = g.RandSendMessageTo("matchserver", charid.(string), uint32(proxymsg.ProxyMessageType_PMT_GS_MS_MATCH), innerReq)
	}, func() {
		if res == false {
			a.WriteMsg(&clientmsg.Rlt_Match{
				RetCode: clientmsg.Type_GameRetCode_GRC_MATCH_ERROR,
			})
		}
	})
}

func handleReqConnectBS(args []interface{}) {
	m := args[0].(*clientmsg.Req_ConnectBS)
	a := args[1].(gate.Agent)

	if g.ConnectRoom(m.CharID, m.RoomID, m.BattleKey) {
		g.AddBattlePlayer(m.CharID, &a)
		a.WriteMsg(&clientmsg.Rlt_ConnectBS{
			RetCode: clientmsg.Type_BattleRetCode_BRC_NONE,
		})
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
		RetCode: clientmsg.Type_BattleRetCode_BRC_NONE,
		CharID:  m.CharID,
	})
}

func handleTransferMessage(args []interface{}) {
	a := args[1].(gate.Agent)
	if a.UserData() != nil {
		charid := a.UserData().(string)
		g.AddMessage(charid, args[0])
	}
}
