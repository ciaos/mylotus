package internal

import (
	"reflect"
	"server/conf"
	"server/msg/clientmsg"
	"time"

	"github.com/name5566/leaf/db/mongodb"
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

type Account struct {
	UserName   string
	PassWord   string
	Status     int32
	CreateTime time.Time
	UpdateTime time.Time
}

func init() {
	handler(&clientmsg.Req_Register{}, handleRegister)
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
			err = c.Insert(&Account{
				UserName:   m.GetUserName(),
				PassWord:   m.GetPassword(),
				Status:     0,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			})
			if err != nil {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_OTHER)})
			} else {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_NONE)})
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

				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_NONE)})
			} else {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode.Enum(clientmsg.Type_LoginRetCode_LRC_PASSWORD_ERROR)})
			}
		}
	}
}
