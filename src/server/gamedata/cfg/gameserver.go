package cfg

type GameServer struct {
	// index 0
	ServerID    int32 "index"
	AreaName    string
	ServerName  string
	ServerTag   int32
	ConnectAddr string
}
