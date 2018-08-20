package internal

import (
	"server/datastruct"
	"reflect"
    "server/msg"
    "server/db"
    "server/game"
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
    handleMsg(&msg.CS_UserLogin{}, handleUserLogin)
   
}

// 消息处理  
func handleUserLogin(args []interface{}) {
    // 收到的消息  
    m := args[0].(*msg.CS_UserLogin)  
    // 消息的发送者  
    a := args[1].(gate.Agent)
     
   
    
    if m.MsgContent.LoginName == datastruct.NULLSTRING{
        loginFailed(a)
        return
    }
    
    if m.MsgContent.Platform != msg.PC_Platform{
       str:=thirdParty.GetOpenID(m.MsgContent.Platform,m.MsgContent.LoginName)
        if str!=""{
           m.MsgContent.LoginName = str
        }
    }
    
    if m.MsgContent.LoginName == datastruct.NULLSTRING{
        loginFailed(a)
        
        return 
    }
   
    avatar:= m.MsgContent.Avatar
    if avatar == datastruct.NULLSTRING{
        avatar = tools.GetDefaultAvatar()
        m.MsgContent.Avatar = avatar
    }
    uid := db.Module.UserLogin(m)
    
    
    //log.Debug("RemoteAddr %v", a.RemoteAddr().String())//客户端地址
	// log.Debug("LocalAddr %v", a.LocalAddr().String())//服务器本机地址

   
   
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
      
      var msgHeader json.MsgHeader
      msgHeader.MsgName = "SC_UserLogin"
      var msgContent msg.SC_UserLoginContent
      msgContent.IsSuccessed = 1
      msgContent.Uid =uid
      if m.MsgContent.Platform != msg.PC_Platform{
        msgContent.WXOpenID=m.MsgContent.LoginName
      }
      a.WriteMsg(&msg.SC_UserLogin{
        MsgHeader:msgHeader,
        MsgContent:msgContent,
      })
      log.Debug("nickname:%v,IP:%v",extra.PlayName,a.RemoteAddr().String())
      tools.ReSetAgentUserData(uid,mode,p_id,a,connUUID,extra)
      game.Module.Relogin(m.MsgContent.LoginName,a)
      //log.Debug("a UserData:%v",a.UserData())
      //log.Release("a UserData:%v",a.UserData())
   }else{
      loginFailed(a)
   }
   
    
}

func loginFailed(a gate.Agent){
    var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_UserLogin"
    var msgContent msg.SC_UserLoginContent
    msgContent.IsSuccessed = 0
    a.WriteMsg(&msg.SC_UserLogin{
        MsgHeader:msgHeader,
        MsgContent:msgContent,
    })
    a.Close()
    a.Destroy()
}



