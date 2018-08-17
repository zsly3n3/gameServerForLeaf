package internal

import (
	"server/datastruct"
	"reflect"
    "server/msg"
    "server/db"
    "github.com/name5566/leaf/gate"
    "github.com/name5566/leaf/network/json"
    "server/thirdParty"
    "server/tools"    
    "github.com/name5566/leaf/log"
)

func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
    // 向当前模块（login 模块）注册 UserLogin 消息的消息处理函数 handleUserLogin  
    //handleMsg(&msg.CS_UserLogin{}, handleUserLogin)
   
}

// 消息处理  
func handleUserLogin(args []interface{}) {
    // 收到的消息  
    m := args[0].(*msg.CS_UserLogin)  
    // 消息的发送者  
    a := args[1].(gate.Agent)
     
    log.Debug("handleUserLogin_0")
    if m.MsgContent.Platform != msg.PC_Platform{
       str:=thirdParty.GetOpenID(m.MsgContent.Platform,m.MsgContent.LoginName)
        if str!=""{
           m.MsgContent.LoginName = str
        }
    }
    log.Debug("handleUserLogin_1")
    avatar:= m.MsgContent.Avatar
    if avatar == datastruct.NULLSTRING{
        avatar = tools.GetDefaultAvatar()
        m.MsgContent.Avatar = avatar
    }
    uid := db.Module.UserLogin(m)
    
    
    //log.Debug("RemoteAddr %v", a.RemoteAddr().String())//客户端地址
	// log.Debug("LocalAddr %v", a.LocalAddr().String())//服务器本机地址

   var msgHeader json.MsgHeader
   msgHeader.MsgName = "SC_UserLogin"

   var msgContent msg.SC_UserLoginContent
   msgContent.Uid =uid
   if m.MsgContent.Platform != msg.PC_Platform{
    msgContent.WXOpenID=m.MsgContent.LoginName
   }
   //log.Debug("login uid:%v",uid)
   //log.Release("login uid:%v",uid)
   if uid > 0{
      connUUID:=tools.UniqueId()
      mode:=datastruct.NULLMode
      p_id:=datastruct.NULLID
      var extra datastruct.ExtraUserData
      extra.Avatar = m.MsgContent.Avatar
      extra.PlayName = m.MsgContent.NickName
      extra.RoomID = datastruct.NULLSTRING
      extra.WaitRoomID = datastruct.NULLSTRING
      extra.IsSettle = false
      tools.ReSetAgentUserData(uid,mode,p_id,a,connUUID,extra)
      //log.Debug("a UserData:%v",a.UserData())
      //log.Release("a UserData:%v",a.UserData())
   }
   a.WriteMsg(&msg.SC_UserLogin{
        MsgHeader:msgHeader,
        MsgContent:msgContent,
   })
    
}



