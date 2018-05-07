package msg

import (
	"github.com/name5566/leaf/network/json"
)

var Processor = json.NewProcessor()

func init() {
	Processor.Register(&CS_UserLogin{})
	Processor.Register(&SC_UserLogin{})
	Processor.Register(&CS_PlayerMatching{})
	Processor.Register(&SC_PlayerMatching{})
}

/*客户端发送来完成注册*/
type CS_UserLogin struct {
	MsgHeader *json.MsgHeader
	MsgContent *CS_UserLoginContent
}

type CS_UserLoginContent struct {
	LoginName string //如果是微信发送过来就是微信code
	NickName string
	Avatar string
	Platform string //告知服务端是从哪家平台发送过来的,比如"微信","QQ"
}


/*服务端发送给客户端*/
type SC_UserLogin struct {
    MsgHeader *json.MsgHeader
	MsgContent *SC_UserLoginContent
}
type SC_UserLoginContent struct {
	Uid int //生成的用户id;为-1时,代表没登陆成功
}


/*玩家开始匹配*/
type CS_PlayerMatching struct {
	MsgHeader *json.MsgHeader
	MsgContent *CS_PlayerMatchingContent
}
type CS_PlayerMatchingContent struct {
	Uid int
}


/*发送正在匹配中*/
type SC_PlayerMatching struct {
	MsgHeader *json.MsgHeader
	MsgContent *SC_PlayerMatchingContent
}
type SC_PlayerMatchingContent struct {
	IsMatching bool
}


/*发送匹配成功的机器人*/
type SC_UserMatchRobot struct {
	MapData string //地图数据
}








