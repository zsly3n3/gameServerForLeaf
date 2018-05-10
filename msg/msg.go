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
	Processor.Register(&SC_PlayerOnline{})
	Processor.Register(&SC_PlayerRoomData{})
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


/*发送正在匹配中*/
type SC_PlayerMatching struct {
	MsgHeader json.MsgHeader
	MsgContent SC_PlayerMatchingContent
}
type SC_PlayerMatchingContent struct {
	IsMatching bool
}

/*已在线(在匹配或在房间)*/
type SC_PlayerOnline struct {
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

/*发送给客户端当前帧数据*/
type SC_GetRoomFrameData struct {
	MsgHeader json.MsgHeader
	MsgContent SC_GetRoomFrameDataContent
}
type SC_GetRoomFrameDataContent struct {
	 //暂定
}

//RoomID string







