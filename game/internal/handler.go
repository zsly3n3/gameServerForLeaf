package internal
import (
    "reflect"  
    "server/msg"
    "server/datastruct"
    "github.com/name5566/leaf/gate"  
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/network/json"
    "server/tools"
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
    m := args[0].(*msg.CS_MoveData)
    
    switch agentUserData.GameMode{
    case datastruct.SinglePersonMode:
         ptr_singleMatch.PlayerMoved(r_id,connUUID,agentUserData.Uid,m)
    case datastruct.EndlessMode:

    }
   
}

func handlePlayerJoinRoom(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID
    m := args[0].(*msg.CS_PlayerJoinRoom)
    switch agentUserData.GameMode{
    case datastruct.SinglePersonMode:
         ptr_singleMatch.PlayerJoin(connUUID,m)
    case datastruct.EndlessMode:

    }

    
}

func handleRoomOff(a gate.Agent,connUUID string,uid int,mode datastruct.GameModeType){
    a.WriteMsg(msg.GetReMatchMsg())
    switch mode{
     case datastruct.SinglePersonMode:
         ptr_singleMatch.Matching(connUUID,a,uid)
     case datastruct.EndlessMode:
   
    }
}

func handleCancelMatching(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID

    switch agentUserData.GameMode{
    case datastruct.SinglePersonMode:
         ptr_singleMatch.RemovePlayerFromMatchingPool(connUUID)
    case datastruct.EndlessMode:
        
    }

}
 




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
    
    tools.UpdateAgentUserData(a,agentUserData.ConnUUID,agentUserData.Uid,agentUserData.RoomID,datastruct.SinglePersonMode)

    removePlayerFromOtherMatchs(connUUID,datastruct.SinglePersonMode)
     
    if ptr_singleMatch.CheckActionPool(connUUID){//已在匹配中
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

func removePlayerFromOtherMatchs(connUUID string,mode datastruct.GameModeType){
     switch mode{
      case datastruct.SinglePersonMode:
        log.Debug("remove connUUID from other matchs")
      case datastruct.EndlessMode:
     }
}

func singlePersonMatchingPlayers(p_uuid string, a gate.Agent,uid int){
     ptr_singleMatch.Matching(p_uuid,a,uid)
}

func removePlayer(key string,mode datastruct.GameModeType){
    switch mode{
      case datastruct.SinglePersonMode:
           ptr_singleMatch.RemovePlayer(key)
      case datastruct.EndlessMode:
     
    }
    //在其他匹配模式中删除玩家
}