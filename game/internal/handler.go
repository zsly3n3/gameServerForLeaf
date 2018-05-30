package internal
import (
	"server/db"
    "reflect"  
    "server/msg"
    "server/datastruct"
    "github.com/name5566/leaf/gate"  
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/network/json"
    "server/tools"
    "time"
    "server/game/internal/Matching"
)

// 异步处理  
func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
    handleMsg(&msg.CS_PlayerMatching{}, handleSinglePersonMatching)
    handleMsg(&msg.CS_PlayerCancelMatching{}, handleCancelMatching)
    handleMsg(&msg.CS_PlayerJoinRoom{}, handlePlayerJoinRoom)
    handleMsg(&msg.CS_MoveData{}, handlePlayerMoveData)
}

func handlePlayerMoveData(args []interface{}){
   
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID
    r_id:=agentUserData.RoomID
    
    
    
    room:=rooms.Get(r_id)
    if v,ok:=room.playersData.CheckValue(connUUID);ok{
        if v.ActionType == msg.Create{
            return
        }
    }
   
    m := args[0].(*msg.CS_MoveData)
    action:=msg.GetCreatePlayerMoved(agentUserData.Uid,m.MsgContent.X,m.MsgContent.Y,m.MsgContent.Speed)
    var actionData PlayerActionData
    actionData.ActionType = action.Action
    actionData.Data = action
    room.playersData.Set(connUUID,actionData)
    
}

func handlePlayerJoinRoom(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID
    player,tf:=onlinePlayers.CheckAndCleanState(connUUID,datastruct.EmptyWay,datastruct.NULLSTRING)
    if tf{
        m := args[0].(*msg.CS_PlayerJoinRoom)
        r_id := m.MsgContent.RoomID
        if player.GameData.EnterType == datastruct.FreeRoom&&player.GameData.RoomId==r_id{
           room:=rooms.Get(r_id)
           isOn:=room.Join(connUUID,player,false)
           if isOn{
              log.Debug("通过遍历空闲房间进入") 
              removeFromMatchActionPool(connUUID)
           }else{
              go handleRoomOff(a,connUUID)
           }
        }else if player.GameData.EnterType == datastruct.FromMatchingPool{
            room:=rooms.Get(r_id)
            for _,v:=range room.unlockedData.AllowList{
                if v == connUUID{
                    log.Debug("通过匹配池进入")
                    room.Join(connUUID,player,true)
                    removeFromMatchActionPool(connUUID)
                    break
                }
            }
        }else{
            a.WriteMsg(msg.GetJoinInvalidMsg())
        }
    }
}

func handleRoomOff(a gate.Agent,connUUID string){
    a.WriteMsg(msg.GetReMatchMsg())
    matchingPlayers(connUUID)
}

func handleCancelMatching(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID
    removeFromMatchActionPool(connUUID)
    m_pool:=getPool()
    m_pool.Mutex.Lock()
    defer m_pool.Mutex.Unlock()
    rm_index:=-1
    for index,v := range m_pool.Pool{
        if connUUID == v{
            rm_index = index
            break
        }
    }
    if rm_index != -1{
        m_pool.Pool=append(m_pool.Pool[:rm_index], m_pool.Pool[rm_index+1:]...)
    }
}
 



//修改后

//收到单人匹配消息的时候加入池，主动离开和自动离开在池中删除，
//完成单人匹配后，在池中删除
func handleSinglePersonMatching(args []interface{}) {
    
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)

    uid:=agentUserData.Uid
    if uid <= 0{
        log.Error("uid error : %v",uid)
        return
    }
    connUUID:=agentUserData.ConnUUID
    var msgHeader json.MsgHeader
    
    removePlayerFromOtherMatchs(connUUID,singleMatch)
     
    if singleMatch.CheckActionPool(connUUID){//已在匹配中
        msgHeader.MsgName = msg.SC_PlayerAlreadyMatchingKey
        a.WriteMsg(&msg.SC_PlayerAlreadyMatching{
            MsgHeader:msgHeader,
        })
        return
    }
    
    ChanRPC.Go(SinglePersonMatchingKey,connUUID,a,uid) //玩家匹配
    
    msgHeader.MsgName = msg.SC_PlayerMatchingKey

    var msgContent msg.SC_PlayerMatchingContent
    msgContent.IsMatching =true

    a.WriteMsg(&msg.SC_PlayerMatching{  
        MsgHeader:msgHeader,
        MsgContent:msgContent,
    })
} 

func removePlayerFromOtherMatchs(connUUID string,match interface{}){
     switch match.(type){
           case Matching.SingleMatch:
             log.Debug("remove connUUID from other matchs")
     }
}

func singlePersonMatchingPlayers(p_uuid string, a gate.Agent,uid int){
     singleMatch.Matching(p_uuid,a,uid)
}

func removePlayer(key string){
     singleMatch.RemovePlayer(key)
     //在其他匹配模式中删除玩家
}