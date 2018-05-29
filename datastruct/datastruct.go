package datastruct

import (
	"time"
	"github.com/name5566/leaf/gate"
	"sync"
)

const NULLSTRING = ""

type User struct {
    Id       int       `xorm:"not null pk autoincr INT(11)"`
    LoginName string    `xorm:"VARCHAR(64) not null"`
	Avatar      string    `xorm:"VARCHAR(256)"`
	NickName string `xorm:"not null CHAR(64)"`
	CreatedAt time.Time `xorm:"created"`
}




type PlayerEnterType int //玩家进入房间的类型
const (
	EmptyWay PlayerEnterType = iota
	FromMatchingPool//通过匹配池准备进入
	FreeRoom//通过遍历空闲房间准备进入
	BeInvited//通过被邀请准备进入
)

type AgentUserData struct {
	 ConnUUID string //每条连接的uuid
	 Uid int //对应user表中的Id
	 RoomID string 
}

type Player struct {
	Id       int //对应user表中的Id   
	Avatar   string 
	NickName string
	Agent    gate.Agent
    GameData PlayerGameData
}

type PlayerGameData struct{
	RoomId   string //房间id
	StartMatchingTime  time.Time //开始匹配的时间
	EnterType PlayerEnterType
	FrameIndex int //保存 已接收第多少帧，大于0
}


/*机器人*/
type Robot struct {
	Id int
	Avatar   string 
	NickName string 
	IsRelive bool //是否能重生
	Action interface{}
	SpeedInterval int ///加速的时间间隔
	StopSpeedFrameIndex int //持续到多少帧结束 
	DirectionInterval int //转向的时间间隔
}






/*匹配池*/
type MatchingPool struct {
	Mutex *sync.RWMutex //读写互斥量
	Pool  []string //存放玩家uuid
}


/*匹配动作池*/ //收到匹配消息的时候加入池，主动离开和自动离开在池中删除，完成匹配后，在池中删除
type MatchActionPool struct {
	Mutex *sync.RWMutex //读写互斥量
	Pool  []string //存放玩家uuid
}


