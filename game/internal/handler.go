package internal
import (
	"server/db"
    "reflect"  
    "server/msg"
    "server/datastruct"
    "github.com/name5566/leaf/gate"  
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/network/json"
    "sync"
    // "time"
    // "fmt"
)
func init() {  
    //向当前模块（game 模块）注册 CS_PlayerMatching 消息 
    handler(&msg.CS_PlayerMatching{}, handlePlayerMatching)
}  
// 异步处理  
func handler(m interface{}, h interface{}) {  
    skeleton.RegisterChanRPC(reflect.TypeOf(m), h)  
}

// 消息处理  
func handlePlayerMatching(args []interface{}) {
    m := args[0].(*msg.CS_PlayerMatching)
    a := args[1].(gate.Agent)

    uid:=m.MsgContent.Uid

    if uid <= 0{
        log.Error("uid error : %v",uid)
        return
     }

    if onlinePlayers.Check(uid){//已在线
       log.Error("uid %v onlined",uid)
       return
    }
    user:=db.Module.GetUserInfo(uid)
    player:=new(datastruct.Player)
    player.Avatar=user.Avatar
    player.Id=uid
    player.NickName=user.NickName
    player.Agent = &a
    player.LocationStatus = datastruct.Matching
    player.Mutex = new(sync.RWMutex)
    addOnlinePlayer(player)
    
    //ChanRPC.Go("MatchingPlayers",player.Id) //玩家匹配

    // &msg.SC_PlayerMatching{
    //     MsgHeader:msgHeader,
    //     MsgContent:msgContent,
    // }

    msgHeader:=new(json.MsgHeader)
    msgHeader.MsgId = 321
    msgHeader.MsgName = "SC_PlayerMatching"

    msgContent:=new(msg.SC_PlayerMatchingContent)
    msgContent.IsMatching =true

    a.WriteMsg(&msg.SC_PlayerMatching{  
        MsgHeader:msgHeader,
        MsgContent:msgContent,
    })
}  

func addOnlinePlayer(player *datastruct.Player){
   onlinePlayers.Set(player.Id,player)
}



