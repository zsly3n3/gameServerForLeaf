package conf

import (
	"log"
	"time"
)

var (
	// log conf
	LogFlag = log.LstdFlags
	
	IsRelease = false //是否为正式服的开关
	
	// gate conf
	PendingWriteNum        = 2000
	MaxMsgLen       uint32 = 4096*1024
	HTTPTimeout            = 10 * time.Second
	LenMsgLen              = 2
	LittleEndian           = false

	// skeleton conf
	GoLen              = 10000
	TimerDispatcherLen = 10000
	AsynCallLen        = 10000
	ChanRPCLen         = 10000
)
