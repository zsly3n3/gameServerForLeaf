package internal

import (
	"reflect"
    "server/msg"
    "server/db"
    "github.com/name5566/leaf/gate"
    "github.com/name5566/leaf/network/json"
    "server/thirdParty"
    //"github.com/name5566/leaf/log"
    //"fmt"
)

func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
    // 向当前模块（login 模块）注册 UserLogin 消息的消息处理函数 handleUserLogin  
    handleMsg(&msg.CS_UserLogin{}, handleUserLogin)
}

// 消息处理  
func handleUserLogin(args []interface{}) {
    // 收到的消息  
    m := args[0].(*msg.CS_UserLogin)  
    // 消息的发送者  
    a := args[1].(gate.Agent)
   
   
    
    //fmt.Println("handleUserLogin:")
    //fmt.Println(m.MsgContent.Platform)
    
    p_str:=thirdParty.GetOpenID("微信",m.MsgContent.LoginName)
    if p_str!=nil{
       m.MsgContent.LoginName = *p_str  
    }
    uid := db.Module.UserLogin(m)

    // fmt.Println("handleUserLogin:")
    // fmt.Println(*m.MsgContent)
    // fmt.Println(*m.MsgHeader)
	//log.Debug("RemoteAddr %v", a.RemoteAddr().String())//客户端地址
	// log.Debug("LocalAddr %v", a.LocalAddr().String())//服务器本机地址
	    
    msgHeader:=new(json.MsgHeader)
    msgHeader.MsgId = 123
    msgHeader.MsgName = "SC_UserLogin"

    msgContent:=new(msg.SC_UserLoginContent)
    msgContent.Uid =uid

    
   
    a.WriteMsg(&msg.SC_UserLogin{
        MsgHeader:msgHeader,
        MsgContent:msgContent,
    })
    
} 