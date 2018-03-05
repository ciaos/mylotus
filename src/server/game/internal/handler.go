package internal

import (
	"fmt"
	"reflect"
	"server/conf"
	"server/game/internal/data"
	"server/msg/clientmsg"
	"server/tool"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/db/mongodb"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	handler(&clientmsg.Ping{}, handlePing)
	handler(&clientmsg.Req_ServerTime{}, handleReqServerTime)
	handler(&clientmsg.Req_Login{}, handleReqLogin)
	handler(&clientmsg.Req_Match{}, handleReqMatch)

	mongo, _ = mongodb.Dial(conf.Server.MongoDBHost, 10)
}

var mongo *mongodb.DialContext

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handlePing(args []interface{}) {
	m := args[0].(*clientmsg.Ping)
	a := args[1].(gate.Agent)

	log.Debug("RecvPing %v %v", a.RemoteAddr())
	a.WriteMsg(&clientmsg.Pong{ID: proto.Uint32(m.GetID())})

	//SendMessageTo(int32(conf.Server.ServerID), conf.Server.ServerType, uint64(1), uint32(0), m)
}

func handleReqServerTime(args []interface{}) {
	//	m := args[0].(*clientmsg.Req_ServerTime)
	a := args[1].(gate.Agent)

	a.WriteMsg(&clientmsg.Rlt_ServerTime{Time: proto.Uint32(uint32(time.Now().Unix()))})
}

func handleReqLogin(args []interface{}) {
	m := args[0].(*clientmsg.Req_Login)
	a := args[1].(gate.Agent)

	if int(m.GetServerID()) != conf.Server.ServerID {
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode.Enum(clientmsg.Type_GameRetCode_GRC_OTHER),
		})
		return
	}

	userid, err := tool.DesDecrypt(m.GetSessionKey(), []byte(tool.CRYPT_KEY))
	if err != nil {
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode.Enum(clientmsg.Type_GameRetCode_GRC_OTHER),
		})
		return
	}

	if strings.Compare(string(userid), m.GetUserID()) != 0 {
		log.Error("strings.Compare(string(userid), m.GetUserID()) != 0")
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode: clientmsg.Type_GameRetCode.Enum(clientmsg.Type_GameRetCode_GRC_OTHER),
		})
		return
	}

	s := mongo.Ref()
	defer mongo.UnRef(s)

	c := s.DB("game").C("character")

	var pcharid string
	result := data.Character{}
	err = c.Find(bson.M{"userid": m.GetUserID(), "gsid": conf.Server.ServerID}).One(&result)
	if err != nil {
		//create new character
		charid := bson.NewObjectId()
		err = c.Insert(&data.Character{
			Id:         charid,
			UserId:     m.GetUserID(),
			GsId:       m.GetServerID(),
			Status:     data.STATUS_ONLINE,
			CreateTime: time.Now(),
		})
		if err != nil {
			log.Error("create new character error %v", err)
			a.WriteMsg(&clientmsg.Rlt_Login{
				RetCode: clientmsg.Type_GameRetCode.Enum(clientmsg.Type_GameRetCode_GRC_OTHER),
			})
			return
		}

		c = s.DB("game").C(fmt.Sprintf("userinfo_%d", m.GetServerID()))
		c.Insert(&data.UserInfo{
			CharId:   charid.String(),
			CharName: "",
			Level:    1,
		})

		a.SetUserData(charid.String())
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode.Enum(clientmsg.Type_GameRetCode_GRC_NONE),
			CharID:         proto.String(charid.String()),
			IsNewCharacter: proto.Bool(true),
		})

		pcharid = charid.String()

	} else {
		c.Update(bson.M{"_id": result.Id}, bson.M{"$set": bson.M{"updatetime": time.Now(), "status": data.STATUS_ONLINE}})

		a.SetUserData(result.Id.String())
		a.WriteMsg(&clientmsg.Rlt_Login{
			RetCode:        clientmsg.Type_GameRetCode.Enum(clientmsg.Type_GameRetCode_GRC_NONE),
			CharID:         proto.String(result.Id.String()),
			IsNewCharacter: proto.Bool(false),
		})

		pcharid = result.Id.String()
	}

	data.PlayerManager[pcharid] = &a
	log.Debug("PlayerManager Add %v", pcharid)
}

func handleReqMatch(args []interface{}) {

}
