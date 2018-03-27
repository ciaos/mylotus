package internal

import (
	"encoding/binary"
	"reflect"
	"server/game/g"
	"server/gamedata"
	"server/gamedata/cfg"
	"server/msg/clientmsg"
	"server/tool"
	"time"

	"github.com/ciaos/leaf/gate"
	"github.com/ciaos/leaf/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Account struct {
	Id         bson.ObjectId `json:"id"        bson:"_id"`
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

func getNextSeq() uint32 {
	s := Pmongo.Ref()
	defer Pmongo.UnRef(s)

	c := s.DB(g.DB_NAME_LOGIN).C(g.TB_NAME_COUNTER)

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

func handleRegister(args []interface{}) {
	m := args[0].(*clientmsg.Req_Register)
	a := args[1].(gate.Agent)

	// session
	s := Pmongo.Ref()
	defer Pmongo.UnRef(s)

	c := s.DB(g.DB_NAME_LOGIN).C(g.TB_NAME_ACCOUNT)

	result := Account{}
	err := c.Find(bson.M{"username": m.UserName}).Select(bson.M{"userid": 1, "password": 1}).One(&result)
	if err != nil && err.Error() == "not found" {
		//Account Not Exist
		if m.IsLogin {
			a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST})
		} else {

			userid := getNextSeq()
			if userid == 0 {
				a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				return
			}

			err = c.Insert(&Account{
				Id:         bson.NewObjectId(),
				UserID:     userid,
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
				binary.BigEndian.PutUint32(sessionbuf, userid)
				binary.BigEndian.PutUint64(sessionbuf[4:], uint64(time.Now().Unix()))
				sessionkey, err := tool.DesEncrypt(sessionbuf, []byte(tool.CRYPT_KEY))

				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OK, UserID: userid, SessionKey: sessionkey})
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

				sessionbuf := make([]byte, 12)
				binary.BigEndian.PutUint32(sessionbuf, result.UserID)
				binary.BigEndian.PutUint64(sessionbuf[4:], uint64(time.Now().Unix()))
				sessionkey, err := tool.DesEncrypt(sessionbuf, []byte(tool.CRYPT_KEY))

				if err != nil {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OTHER})
				} else {
					a.WriteMsg(&clientmsg.Rlt_Register{RetCode: clientmsg.Type_LoginRetCode_LRC_OK, UserID: result.UserID, SessionKey: sessionkey})
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

		resMsg.ServerList = append(resMsg.ServerList, si)
		i++
	}
	a.WriteMsg(resMsg)
}
