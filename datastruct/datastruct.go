package datastruct

import (
	"time"
	"sync"
	"github.com/name5566/leaf/gate" 
)

const NULLSTRING = ""
const NULLID = -1

type User struct {
    Id       int       `xorm:"not null pk autoincr INT(11)"`
    LoginName string    `xorm:"VARCHAR(64) not null"`
	Avatar      string    `xorm:"VARCHAR(255)"`
	NickName string `xorm:"not null VARCHAR(255)"`
	CreatedAt time.Time `xorm:"created"`
}

type RobotName struct {
    Id       int       `xorm:"not null pk autoincr INT(11)"`
	Name string    `xorm:"VARCHAR(128) not null"`
	State int8 `xorm:"TINYINT(1) not null"`
}

type MaxScoreInEndlessMode struct {
	Id   int       `xorm:"not null pk autoincr INT(11)"`
	Uid int    `xorm:"INT(11) not null"`
	MaxScore int    `xorm:"INT(11) not null"`
	MaxKillNum int    `xorm:"INT(11) not null"`
	UpdateTime int64 `xorm:"bigint not null"`
}

type MaxScoreInSinglePersonMode struct {
	Id   int       `xorm:"not null pk autoincr INT(11)"`
	Uid int    `xorm:"INT(11) not null"`
	MaxScore int    `xorm:"INT(11) not null"`
	MaxKillNum int    `xorm:"INT(11) not null"`
	UpdateTime int64 `xorm:"bigint not null"`
}

type MaxScoreInInviteMode struct {
	Id   int       `xorm:"not null pk autoincr INT(11)"`
	Uid int    `xorm:"INT(11) not null"`
	MaxScore int    `xorm:"INT(11) not null"`
	MaxKillNum int    `xorm:"INT(11) not null"`
}

type SkinFragment struct {
	Id   int       `xorm:"not null pk autoincr INT(11)"`
	Uid int    `xorm:"INT(11) not null"`
	FragmentNum int    `xorm:"INT(11) not null"`
}

/*积分表*/
type GameIntegral struct {
	Id   int       `xorm:"not null pk autoincr INT(11)"`
	Uid int    `xorm:"INT(11) not null"`
	Integral int    `xorm:"INT(11) not null"`
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
	InviteMode //邀请模式
)

type AgentUserData struct {
	 ConnUUID string //每条连接的uuid
	 Uid int //对应user表中的Id
	 PlayId int //在游戏中生成的Id
	 GameMode GameModeType
	 Extra ExtraUserData
}

type ExtraUserData struct {
	 PlayName string
	 Avatar string
	 RoomID string
	 WaitRoomID string
	 IsSettle bool
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

type EnergyPoint struct {
	Type int
    X int
	Y int
	Scale float32 //默认值是1.0
}

type PlayerDiedData struct {
	PlayerId int
	Points []EnergyPoint
	AddEnergy int
	FrameIndex int
}

/*机器人*/
type Robot struct {
	Id int
	Avatar   string 
	NickName string 
	IsRelive bool //是否能重生
	Action interface{}
	PathIndex int //决定使用哪条线路
	MoveStep int //默认从1开始，移动的步数
	// SpeedInterval int //加速的时间间隔
	// StopSpeedFrameIndex int //持续到多少帧结束 
	// DirectionInterval int //转向的时间间隔
}

/*用于的排行榜数据*/
type PlayerScore struct {
	Rank int 
	Score int
	Avatar string
	Name string
}

type PlayerRankScoreData struct {
	 LengthRank *PlayerScore
}

/*用于的排行榜数据*/
type PlayerKillNum struct {
	Rank int
	KillNum int
	Avatar string
	Name string
}

type PlayerRankKillNumData struct {
	 KillNumRank *PlayerKillNum
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

type WaitRoomState int //等候室状态
const (
	NotFull WaitRoomState = iota //房间还有位置
	Full //房间满员
	NotExist //房间不存在
)

type PlayerInWaitRoom struct {
	NickName string
	Avatar string
	IsMaster int //是否为房主
	Seat int //座位号
}

type MsgLevel int //消息等级
const (
	Common MsgLevel = iota //一般消息
	Importance//重要
)

type GateUserData struct {
	UserData map[string]AgentUserData
	Mutex sync.RWMutex
}


