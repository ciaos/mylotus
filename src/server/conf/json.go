package conf

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ciaos/leaf/log"
)

var Server struct {
	LogLevel    string
	LogPath     string
	WSAddr      string
	CertFile    string
	KeyFile     string
	TCPAddr     string
	UDPAddr     string
	MaxConnNum  int
	ConsolePort int
	ProfilePath string

	TickInterval int

	ServerID      int
	ServerType    string
	RedisHost     string
	RedisPassWord string
	MongoDBHost   string
	ConnectAddr   string

	BattleServerList []BattleServerCfg
	MatchServerList  []MatchServerCfg
}

type BattleServerCfg struct {
	ServerID   int
	ServerType string
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
