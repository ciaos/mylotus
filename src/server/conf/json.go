package conf

import (
	"encoding/json"
	"io/ioutil"

	"github.com/name5566/leaf/log"
)

var Server struct {
	LogLevel    string
	LogPath     string
	WSAddr      string
	CertFile    string
	KeyFile     string
	TCPAddr     string
	MaxConnNum  int
	ConsolePort int
	ProfilePath string

	TickInterval int

	ServerID      int
	ServerType    string
	RedisHost     string
	RedisPassWord string
	MongoDBHost   string

	GameServerList   []GameServerCfg
	BattleServerList []BattleServerCfg
	MatchServerList  []MatchServerCfg
}

type GameServerCfg struct {
	ServerID    int
	ServerType  string
	ServerName  string
	Tag         int
	ConnectAddr string
}

type BattleServerCfg struct {
	ServerID    int
	ServerType  string
	ConnectAddr string
}

type MatchServerCfg struct {
	ServerID   int
	ServerType string
}

func init() {
	data, err := ioutil.ReadFile("conf/server.json")
	if err != nil {
		log.Fatal("%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("%v", err)
	}
}
