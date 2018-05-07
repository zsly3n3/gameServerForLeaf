package datastruct

import (
	"time"
	"github.com/name5566/leaf/gate"
	"sync"
)


type User struct {
    Id       int       `xorm:"not null pk autoincr INT(11)"`
    LoginName string    `xorm:"VARCHAR(64) not null"`
	Avatar      string    `xorm:"VARCHAR(256)"`
	NickName string `xorm:"not null CHAR(64)"`
	CreatedAt time.Time `xorm:"created"`
}

type LocationStatus int //玩家当前在匹配中，房间中还是?

const (
	Matching LocationStatus = iota
	Playing
)

type Player struct {
	Mutex *sync.RWMutex
	Id       int       
	Avatar   string   
	NickName string
	Agent    *gate.Agent
	RoomId   string
	MatchingPoolId  int//匹配池的id
	LocationStatus  LocationStatus //当前位置状态
}

type Room struct {
	RoomId string //根据时间来hash
	Players []*Player
	IsOn bool
	Mutex *sync.RWMutex //读写互斥量
}


type Hall struct {
	Rooms []*Room
	Mutex *sync.RWMutex //读写互斥量
}

/*匹配池*/
type MatchingPool struct {
	Mutex *sync.RWMutex //读写互斥量
	Pool  []int //存放玩家id
}



