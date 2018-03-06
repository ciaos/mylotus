package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(10000))
	password := "123456"

	t.Log("Register Username", username)
	Register(t, &conn, username, password, true, clientmsg.Type_LoginRetCode_LRC_ACCOUNT_NOT_EXIST)

	userid, sessionkey, _ := Register(t, &conn, username, password, false, clientmsg.Type_LoginRetCode_LRC_NONE)
	t.Log("Register UserID ", userid, " SessionKey", sessionkey)
	Register(t, &conn, username, password, false, clientmsg.Type_LoginRetCode_LRC_ACCOUNT_EXIST)

	userid, sessionkey, _ = Register(t, &conn, username, password, true, clientmsg.Type_LoginRetCode_LRC_NONE)
	t.Log("Register UserID ", userid, " SessionKey", sessionkey)
	password = "1234"
	Register(t, &conn, username, password, true, clientmsg.Type_LoginRetCode_LRC_PASSWORD_ERROR)
}
