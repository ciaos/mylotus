package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"server/msg/clientmsg"

	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/log"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		panic(err)
	}
	/*
		reqMsg := &clientmsg.Hello{
			Name: proto.String("PJ"),
		}
	*/
	reqMsg := &clientmsg.Req_Register{
		UserName:      proto.String("PJ"),
		Password:      proto.String("123456"),
		ClientVersion: proto.Int32(1),
		IsLogin:       proto.Bool(false),
	}

	data, err := proto.Marshal(reqMsg)
	if err != nil {
		log.Fatal("Marsha1 failed")
	}
	// len + data
	reqbuf := make([]byte, 4+len(data))

	// 默认使用大端序
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(1))

	copy(reqbuf[4:], data)

	// 发送消息
	conn.Write(reqbuf)

	rspbuf := make([]byte, 2014)

	len, err := conn.Read(rspbuf[0:])

	if err != nil {
	}

	fmt.Println("recv ", rspbuf, len)

	msgid := binary.BigEndian.Uint16(rspbuf[2:])

	switch int(msgid) {
	case 0:
		msg := &clientmsg.Hello{}
		proto.Unmarshal(rspbuf[4:len], msg)
		fmt.Println("Recv 0 ", msg.GetName())
	case 2:
		msg := &clientmsg.Rlt_Register{}
		proto.Unmarshal(rspbuf[4:len], msg)
		fmt.Println("Recv 2 ", msg.GetCode())
	default:
		fmt.Println("Invalid msgid ", msgid, rspbuf)
	}
}
