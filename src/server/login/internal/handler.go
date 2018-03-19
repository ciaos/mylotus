package internal

import (
	"reflect"
	"server/conf"
	"server/msg/clientmsg"
	"server/tool"
	"time"

	"github.com/ciaos/leaf/gate"
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

	// session
	s := Pmongo.Ref()
	defer Pmongo.UnRef(s)

	c := s.DB("login").C("account")

	result := Account{}
	err := c.Find(bson.M{"username": m.UserName}).One(&result)
	if err != nil {
		//Account Not Exist
		if m.IsLogin {
			a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST})
		} else {
			userid := bson.NewObjectId()
			err = c.Insert(&Account{
				Id:         userid,
				UserName:   m.UserName,
				PassWord:   m.Password,
				Status:     0,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			})
			if err != nil {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
			} else {
				sessionkey, err := tool.DesEncrypt([]byte(userid.String()), []byte(tool.CRYPT_KEY))
				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_NONE, UserID: userid.String(), SessionKey: sessionkey})
				}
			}
		}
	} else {
		//Account Exist
		if !m.IsLogin {
			a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_ACCOUNT_EXIST})
			return
		} else {
			if result.PassWord == m.Password {
				c.Update(bson.M{"username": m.UserName}, bson.M{"$set": bson.M{"updatetime": time.Now()}})
				sessionkey, err := tool.DesEncrypt([]byte(result.Id.String()), []byte(tool.CRYPT_KEY))
				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_NONE, UserID: result.Id.String(), SessionKey: sessionkey})
				}
			} else {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_PASSWORD_ERROR})
			}
		}
	}
}

func handlerReqServerList(args []interface{}) {
	//m := args[0].(*clientmsg.Req_ServerList)
	a := args[1].(gate.Agent)

	resMsg := &clientmsg.Rlt_ServerList{}
	resMsg.ServerCount = int32(len(conf.Server.GameServerList))

	for _, serverInfo := range conf.Server.GameServerList {

		si := &clientmsg.Rlt_ServerList_ServerInfo{}
		si.ServerID = int32(serverInfo.ServerID)
		si.ServerName = serverInfo.ServerName
		si.Status = int32(serverInfo.Tag)
		si.ConnectAddr = serverInfo.ConnectAddr

		resMsg.ServerList = append(resMsg.ServerList, si)
	}

	a.WriteMsg(resMsg)
}
