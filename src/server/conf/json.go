package conf

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ciaos/leaf/log"
)

var Server struct {
	LogLevel        string
	LogPath         string
	WSAddr          string
	CertFile        string
	KeyFile         string
	TCPAddr         string
	UDPAddr         string
	MaxConnNum      int
	MaxOnlineNum    int
	MaxWaitLoginNum int
	ConsolePort     int
	ProfilePath     string

	TickInterval  int
	SaveAssetStep int

	ServerID      int
	ServerType    string
	RedisHost     string
	RedisPassWord string
	MongoDBHost   string
	ConnectAddr   string

	GameServerRename   string
	MatchServerRename  string
	BattleServerRename string

	BattleServerList []BattleServerCfg
	MatchServerList  []MatchServerCfg
}

type BattleServerCfg struct {
	ServerID int
}

type MatchServerCfg struct {
	ServerID int
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
