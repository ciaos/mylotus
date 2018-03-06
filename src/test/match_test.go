package test

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	conn, err := net.Dial("tcp", TestServerAddr)
	if err != nil {
		t.Fatal("Connect Server Error ", err)
	}
	defer conn.Close()

	//Login First
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("pengjing%d", rand.Intn(100000))
	password := "123456"
	QuickLogin(t, &conn, username, password)

	QuickMatch(t, &conn)
}
