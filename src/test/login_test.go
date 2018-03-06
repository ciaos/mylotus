package test

import (
	"fmt"
	"math/rand"
	"net"
	"server/msg/clientmsg"
	"testing"
	"time"
)

func TestLogin(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	//Register First
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(100))
	password := "123456"
	islogin := false

	t.Log("Register Username", username)
	userid, sessionkey, err := Register(t, &conn, username, password, islogin, clientmsg.Type_LoginRetCode_LRC_NONE)
	if err != nil {
		t.Fatal("Register Error", err)
	}

	t.Log("Login UserID", userid)
	charid, err := Login(t, &conn, userid, sessionkey, clientmsg.Type_GameRetCode_GRC_NONE)
	if err != nil {
		t.Fatal("Login Error", err)
	}
	t.Log("Login OK CharID", charid)
}
