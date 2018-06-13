package datastruct

import (
	"time"
	"github.com/name5566/leaf/gate"
)

const NULLSTRING = ""
const NULLID = -1

type User struct {
    Id       int       `xorm:"not null pk autoincr INT(11)"`
    LoginName string    `xorm:"VARCHAR(64) not null"`
	Avatar      string    `xorm:"VARCHAR(256)"`
	NickName string `xorm:"not null CHAR(64)"`
	CreatedAt time.Time `xorm:"created"`
}




type PlayerEnterType int //玩家进入房间的类型
const (
	NULLWay PlayerEnterType = iota
	FromMatchingPool//通过匹配池准备进入
	FreeRoom//通过遍历空闲房间准备进入
	BeInvited//通过被邀请准备进入
)

type GameModeType int //玩家进入房间的类型

const (
	NULLMode GameModeType = iota //默认无模式
	SinglePersonMode //单人匹配
	EndlessMode //无尽模式
)

type AgentUserData struct {
	 ConnUUID string //每条连接的uuid
	 Uid int //对应user表中的Id
	 PlayId   int //在游戏中生成的Id
	 RoomID string
	 GameMode GameModeType
}

type Player struct {
	Uid       int //对应user表中的Id  
	Avatar   string
	NickName string
	Agent    gate.Agent
    GameData PlayerGameData
}

type PlayerGameData struct{
	RoomId   string //房间id
	PlayId   int //在游戏中生成的Id
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

func CreatePlayer(user *User) *Player{
	player := new(Player)
    player.Avatar=user.Avatar
	player.Uid=user.Id
	player.NickName=user.NickName
    var game_data PlayerGameData
    game_data.StartMatchingTime = time.Now()
    game_data.EnterType = NULLWay
	game_data.RoomId = NULLSTRING
	game_data.PlayId = NULLID
    player.GameData = game_data
    return player
}

const DefaultReliveFrameIndex = -1 //当前帧立即复活







