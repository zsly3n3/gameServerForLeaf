package internal

import (
	"server/datastruct"
	//"fmt"
	"reflect"
    "server/msg"
    "server/db"
    "github.com/name5566/leaf/gate"
    "github.com/name5566/leaf/network/json"
    "server/thirdParty"
    "server/tools"
    //"github.com/name5566/leaf/log"
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
   
    if m.MsgContent.Platform != msg.PC_Platform{
       str:=thirdParty.GetOpenID(m.MsgContent.Platform,m.MsgContent.LoginName)
        if str!=""{
           m.MsgContent.LoginName = str
        }
    }
    
    uid := db.Module.UserLogin(m)

	//log.Debug("RemoteAddr %v", a.RemoteAddr().String())//客户端地址
	// log.Debug("LocalAddr %v", a.LocalAddr().String())//服务器本机地址

    var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_UserLogin"

    var msgContent msg.SC_UserLoginContent
    msgContent.Uid =uid
   
   if uid > 0{
      connUUID:=tools.UniqueId()
      rid:=datastruct.NULLSTRING
      mode:=datastruct.NULLMode
      p_id:=datastruct.NULLID
      tools.ReSetAgentUserData(a,connUUID,uid,rid,mode,p_id)
   }
   
   a.WriteMsg(&msg.SC_UserLogin{
        MsgHeader:msgHeader,
        MsgContent:msgContent,
   })
    
}



