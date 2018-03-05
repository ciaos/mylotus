package internal

import (
	"reflect"
	"server/conf"
	"server/msg/clientmsg"
	"server/tool"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/db/mongodb"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

type Account struct {
	Id         bson.ObjectId `json:"id"        bson:"_id"`
	UserName   string
	PassWord   string
	Status     int32
	CreateTime time.Time
	UpdateTime time.Time
}

func init() {
	handler(&clientmsg.Req_Register{}, handleRegister)
	handler(&clientmsg.Req_ServerList{}, handlerReqServerList)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handleRegister(args []interface{}) {
	m := args[0].(*clientmsg.Req_Register)
	a := args[1].(gate.Agent)

	dc, err := mongodb.Dial(conf.Server.MongoDBHost, 10)
	if err != nil {
		log.Error("handleRegister mongodb.Dial Error %v %v", conf.Server.MongoDBHost, err)
		return
	}
	defer dc.Close()

	// session
	s := dc.Ref()
	defer dc.UnRef(s)

	c := s.DB("login").C("account")

	result := Account{}
	err = c.Find(bson.M{"username": m.GetUserName()}).One(&result)
	if err != nil {
		//Account Not Exist
		if m.GetIsLogin() {
			a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST)})
		} else {
			userid := bson.NewObjectId()
			err = c.Insert(&Account{
				Id:         userid,
				UserName:   m.GetUserName(),
				PassWord:   m.GetPassword(),
				Status:     0,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			})
			if err != nil {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_OTHER)})
			} else {
				sessionkey, err := tool.DesEncrypt([]byte(userid.String()), []byte(tool.CRYPT_KEY))
				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_OTHER)})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_NONE), UserID: proto.String(userid.String()), SessionKey: sessionkey})
				}
			}
		}
	} else {
		//Account Exist
		if !m.GetIsLogin() {
			a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_ACCOUNT_EXIST)})
			return
		} else {
			if result.PassWord == m.GetPassword() {
				c.Update(bson.M{"username": m.GetUserName()}, bson.M{"$set": bson.M{"updatetime": time.Now()}})
				sessionkey, err := tool.DesEncrypt([]byte(result.Id.String()), []byte(tool.CRYPT_KEY))
				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_OTHER)})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_NONE), UserID: proto.String(result.Id.String()), SessionKey: sessionkey})
				}
			} else {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_PASSWORD_ERROR)})
			}
		}
	}
}

func handlerReqServerList(args []interface{}) {
	//m := args[0].(*clientmsg.Req_ServerList)
	a := args[1].(gate.Agent)

	resMsg := &clientmsg.Rlt_ServerList{}
	resMsg.ServerCount = proto.Int32(int32(len(conf.Server.GameServerList)))

	for _, serverInfo := range conf.Server.GameServerList {

		si := &clientmsg.Rlt_ServerList_ServerInfo{}
		si.ServerID = proto.Int32(int32(serverInfo.ServerID))
		si.ServerName = proto.String(serverInfo.ServerName)
		si.Status = proto.Int32(int32(serverInfo.Tag))
		si.ConnectAddr = proto.String(serverInfo.ConnectAddr)

		resMsg.ServerList = append(resMsg.ServerList, si)
	}

	a.WriteMsg(resMsg)
}
