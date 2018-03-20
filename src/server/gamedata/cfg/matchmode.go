package cfg

type MatchMode struct {
	// index 0
	IndexInt         int32 "index"
	PlayerCnt        int
	MatchTimeOutSec  int32
	ChooseTimeOutSec int32
	FixedWaitTimeSec int32
}
