package cfg

type MatchMode struct {
	// index 0
	IndexInt          int32 "index"
	PlayerCnt         int
	MatchTimeOutSec   int32
	ConfirmTimeOutSec int32
	RejectWaitTime    int32
	ChooseTimeOutSec  int32
	FixedWaitTimeSec  int32
}
