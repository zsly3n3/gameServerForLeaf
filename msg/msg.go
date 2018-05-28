package msg

import (
	"github.com/name5566/leaf/network/json"
)

const PC_Platform ="pc"  //pc端
const WX_Platform ="wx" //微信平台


var Processor = json.NewProcessor()

func init() {
	Processor.Register(&CS_UserLogin{})
	Processor.Register(&SC_UserLogin{})
	Processor.Register(&CS_PlayerMatching{})
	Processor.Register(&SC_PlayerMatching{})
	Processor.Register(&SC_PlayerAlreadyMatching{})
	Processor.Register(&SC_PlayerMatchingEnd{})
	Processor.Register(&CS_PlayerCancelMatching{})

	Processor.Register(&CS_PlayerJoinRoom{})

	Processor.Register(&SC_PlayerReMatch{})

	Processor.Register(&SC_InitRoomData{})
	Processor.Register(&SC_RoomFrameData{})
	Processor.Register(&CS_MoveData{})
	//Processor.Register(&PlayerMoved{})
}

/*客户端发送来完成注册*/
type CS_UserLogin struct {
	MsgHeader json.MsgHeader
	MsgContent CS_UserLoginContent
}

type CS_UserLoginContent struct {
	LoginName string //如果是微信发送过来就是微信code
	NickName string
	Avatar string
	Platform string //告知服务端是从哪家平台发送过来的,比如"微信","QQ"
}


/*服务端发送给客户端*/
type SC_UserLogin struct {
    MsgHeader json.MsgHeader
	MsgContent SC_UserLoginContent
}
type SC_UserLoginContent struct {
	Uid int //生成的用户id;为-1时,代表没登陆成功
}


/*玩家开始匹配*/
type CS_PlayerMatching struct {
	MsgHeader json.MsgHeader
}

/*玩家取消匹配*/
type CS_PlayerCancelMatching struct {
	MsgHeader json.MsgHeader
}

/*发送正在匹配中*/
type SC_PlayerMatching struct {
	MsgHeader json.MsgHeader
	MsgContent SC_PlayerMatchingContent
}
type SC_PlayerMatchingContent struct {
	IsMatching bool
}



/*已在匹配中*/
type SC_PlayerAlreadyMatching struct {
	MsgHeader json.MsgHeader
}

/*发送匹配成功的信息*/
type SC_PlayerMatchingEnd struct {
	MsgHeader json.MsgHeader
	MsgContent SC_PlayerMatchingEndContent
}

type SC_PlayerMatchingEndContent struct {
	RoomID string
}

/*客户端发送来加入房间*/
type CS_PlayerJoinRoom struct {
	MsgHeader json.MsgHeader
	MsgContent CS_PlayerJoinRoomContent
}
type CS_PlayerJoinRoomContent struct {
	RoomID string
}

/*玩家加入房间无效*/
type SC_PlayerJoinInvalid struct {
	MsgHeader json.MsgHeader
}

/*重新开始匹配*/
type SC_PlayerReMatch struct {
	MsgHeader json.MsgHeader
}


type EnergyPointType int

//能量点类型
const (
	TypeA EnergyPointType= 1 +iota
    TypeB 
    TypeC 
    TypeD
)

type EnergyPoint struct {
	Type int
    X int
	Y int
}



type Point struct {
    X int
    Y int
}

type Quadrant struct {
    X_Min int
    X_Max int
    Y_Min int
    Y_Max int
}


/*发送给客户端房间初始化数据*/
type SC_InitRoomData struct {
	MsgHeader json.MsgHeader
	MsgContent SC_InitRoomDataContent
}

type SC_InitRoomDataContent struct {
	 MapHeight int//3000
	 MapWidth int//4000
	 CurrentFrameIndex int //游戏进行到当前多少帧,从0开始
	 Interval int //毫秒单位 比如50,代表50毫秒	 
}




/*接收客户端的帧数据*/
type CS_MoveData struct {
	MsgHeader json.MsgHeader
	MsgContent CS_MoveDataContent //{"Action":1,"Direction":{X:-1,Y:-2}
}

type CS_MoveDataContent struct {
	X int
	Y int
	Speed int
}



/*发送给客户端当前帧数据*/
type SC_RoomFrameData struct {
	MsgHeader json.MsgHeader
	MsgContent *SC_RoomFrameDataContent
}

type SC_RoomFrameDataContent struct {
	 FramesData []FrameData
}

type FrameData struct {
	FrameIndex int
	PlayerFrameData []interface{}
	CreateEnergyPoints []EnergyPoint
}

type ActionType int
const (
    Create ActionType = iota // value --> 0
    Move              // value --> 1
	Death            // value --> 2
	
	NullAction        
)

/*以下为玩家事件*/
type CreatePlayer struct {//玩家的创建
	 PlayerId int
	 X int
	 Y int
	 Action ActionType
}

type PlayerIsDied struct {//玩家的死亡
	//point Point
}

var DefaultDirection = Point{X:0,Y:1}
var DefaultSpeed = 1
type PlayerMoved struct {//玩家的移动
	PlayerId int
	Action ActionType
	Speed int//默认速度 1
	X int
	Y int
}


// var Test1Point= Point{X:100,Y:50}
// var Test2Point= Point{X:500,Y:50}

// var num = 0

func GetCreatePlayerAction(p_id int,x int,y int) CreatePlayer{
	  var action CreatePlayer
	  action.Action = Create
	  action.PlayerId = p_id
	//   switch num{
	//   case 0:
	// 	action.X = Test1Point.X
	// 	action.Y = Test1Point.Y
	//   case 1:
	// 	action.X = Test2Point.X
	// 	action.Y = Test2Point.Y
	//   default:
	// 	action.X = x
	// 	action.Y = y
	//   }
	//   num++
	  action.X = x
	  action.Y = y
	  return action
}
func GetCreatePlayerMoved(p_id int,x int,y int,speed int) PlayerMoved{
	var action PlayerMoved
	action.Action = Move
	action.PlayerId = p_id
	action.X = x
	action.Y = y
	action.Speed = speed
	return action
}

func GetMatchingEndMsg(r_id string) *SC_PlayerMatchingEnd{
	var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_PlayerMatchingEnd"
    var msgContent SC_PlayerMatchingEndContent
    msgContent.RoomID =r_id
    
    return &SC_PlayerMatchingEnd{
		MsgHeader:msgHeader,
		MsgContent:msgContent,
	}
}

func GetReMatchMsg() *SC_PlayerReMatch{
	var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_PlayerReMatch"
    return &SC_PlayerReMatch{
		MsgHeader:msgHeader,
	}
}

func GetJoinInvalidMsg() *SC_PlayerJoinInvalid{
	var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_PlayerJoinInvalid"
    return &SC_PlayerJoinInvalid{
		MsgHeader:msgHeader,
	}
}

func GetInitRoomDataMsg(content SC_InitRoomDataContent) *SC_InitRoomData{
	var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_InitRoomData"
    return &SC_InitRoomData{
		MsgHeader:msgHeader,
		MsgContent:content,
	}
}

func GetRoomFrameDataMsg(content *SC_RoomFrameDataContent) *SC_RoomFrameData{
	var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_RoomFrameData"
    return &SC_RoomFrameData{
		MsgHeader:msgHeader,
		MsgContent:content,
	}
}

func GetPower(e_type EnergyPointType) int {
	 power:=0
	 switch e_type{
	   case TypeA:
		 power=10
	   case TypeB:
		 power=20
	   case TypeC:
		 power=40
	   case TypeD: 
	 }
	 return power
}





