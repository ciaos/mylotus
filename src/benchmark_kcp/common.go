package main

import (
	"encoding/binary"
	"errors"
	"server/msg/clientmsg"

	"github.com/ciaos/leaf/kcp"
	"github.com/golang/protobuf/proto"
)

const (
	LoginServerAddr = "127.0.0.1:8888"
	GameServerAddr  = "127.0.0.1:8888"

	GameServerID = 1
)

func Send(conn *kcp.UDPSession, msgid clientmsg.MessageType, msgdata interface{}) {
	data, _ := proto.Marshal(msgdata.(proto.Message))
	reqbuf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(reqbuf[0:], uint16(len(data)+2))
	binary.BigEndian.PutUint16(reqbuf[2:], uint16(msgid))

	copy(reqbuf[4:], data)
	(*conn).Write(reqbuf)
}

func Recv(conn *kcp.UDPSession) (error, clientmsg.MessageType, []byte) {
	headdata := make([]byte, 2)
	(*conn).Read(headdata[0:])

	msglen := binary.BigEndian.Uint16(headdata[0:])

	bodydata := make([]byte, msglen)
	bodylen, _ := (*conn).Read(bodydata[0:])
	if msglen == 0 || bodylen == 0 {
		return errors.New("Invalid msglen"), 0, nil
	}
	msgid := clientmsg.MessageType(binary.BigEndian.Uint16(bodydata[0:]))

	return nil, msgid, bodydata[2:bodylen]
}