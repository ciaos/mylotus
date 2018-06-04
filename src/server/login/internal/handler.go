package internal

import (
	"encoding/binary"
	"reflect"
	"server/gamedata"
	"server/gamedata/cfg"
	"server/msg/clientmsg"
	"server/tool"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2/bson"
)

const (
	DB_NAME_LOGIN   = "login"
	TB_NAME_ACCOUNT = "account"
	TB_NAME_COUNTER = "counter"
)

type Account struct {
	UserID     uint32
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

func getNextSeq() (int, error) {
	id, err := Pmongo.NextSeq(DB_NAME_LOGIN, TB_NAME_COUNTER, "counterid")
	if err != nil && err.Error() == "not found" {
		s := Pmongo.Ref()
		defer Pmongo.UnRef(s)

		type Counter struct {
			Id  string "_id"
			Seq int
		}

		id = 1
		c := s.DB(DB_NAME_LOGIN).C(TB_NAME_COUNTER)
		err = c.Insert(&Counter{
			Id:  "counterid",
			Seq: id,
		})
	}

	return id, err
}

func handleRegister(args []interface{}) {
	m := args[0].(*clientmsg.Req_Register)
	a := args[1].(gate.Agent)

	// session
	s := Pmongo.Ref()
	defer Pmongo.UnRef(s)

	c := s.DB(DB_NAME_LOGIN).C(TB_NAME_ACCOUNT)

	result := Account{}
	err := c.Find(bson.M{"username": m.UserName}).Select(bson.M{"userid": 1, "password": 1}).One(&result)
	if err != nil && err.Error() == "not found" {
		//Account Not Exist
		if m.IsLogin {
			a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST})
		} else {
			userid, err := getNextSeq()
			if err != nil {
				log.Error("getNextSeq Error %v", err)
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				return
			}

			err = c.Insert(&Account{
				UserID:     uint32(userid),
				UserName:   m.UserName,
				PassWord:   m.Password,
				Status:     0,
				CreateTime: time.Now(),
				UpdateTime: time.Now(),
			})
			if err != nil {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
			} else {

				sessionbuf := make([]byte, 12)
				binary.BigEndian.PutUint32(sessionbuf, uint32(userid))
				binary.BigEndian.PutUint64(sessionbuf[4:], uint64(time.Now().Unix()))
				sessionkey, err := tool.DesEncrypt(sessionbuf, []byte(tool.CRYPT_KEY))

				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OK, UserID: uint32(userid), SessionKey: sessionkey})
				}

				log.Release("CreateAccount UserID : %v , UserName : %v", userid, m.UserName)
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

				sessionbuf := make([]byte, 12)
				binary.BigEndian.PutUint32(sessionbuf, result.UserID)
				binary.BigEndian.PutUint64(sessionbuf[4:], uint64(time.Now().Unix()))
				sessionkey, err := tool.DesEncrypt(sessionbuf, []byte(tool.CRYPT_KEY))

				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OK, UserID: result.UserID, SessionKey: sessionkey})

					log.Debug("LoginAccount UserID : %v , UserName : %v", result.UserID, m.UserName)
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

	resMsg.ServerCount = int32(gamedata.CSVGameServer.NumRecord())

	i := 0
	for i < int(resMsg.ServerCount) {

		gscfg := gamedata.CSVGameServer.Record(i).(*cfg.GameServer)
		si := &clientmsg.Rlt_ServerList_ServerInfo{}
		si.ServerID = gscfg.ServerID
		si.ServerName = gscfg.ServerName
		si.Status = gscfg.ServerTag
		si.ConnectAddr = gscfg.ConnectAddr
		si.AreaName = gscfg.AreaName

		resMsg.ServerList = append(resMsg.ServerList, si)
		i++
	}
	a.WriteMsg(resMsg)
}
