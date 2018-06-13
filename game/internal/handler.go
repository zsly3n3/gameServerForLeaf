package internal
import (
    "reflect"  
    "server/msg"
    "server/datastruct"
    "github.com/name5566/leaf/gate"  
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/network/json"
    "server/tools"
    "server/game/internal/match"
)

// 异步处理  
func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
    handleMsg(&msg.CS_PlayerMatching{}, handleSinglePersonMatching)
    handleMsg(&msg.CS_EndlessModeMatching{},handleEndlessModeMatching)

    handleMsg(&msg.CS_PlayerCancelMatching{}, handleCancelMatching)
    handleMsg(&msg.CS_PlayerJoinRoom{}, handlePlayerJoinRoom)
    handleMsg(&msg.CS_MoveData{}, handlePlayerMoveData)
    handleMsg(&msg.CS_EnergyExpended{}, handleEnergyExpended)
    handleMsg(&msg.CS_PlayerDied{}, handlePlayersDied)
    handleMsg(&msg.CS_PlayerLeftRoom{}, handlePlayerLeftRoom)
    handleMsg(&msg.CS_PlayerRelive{}, handlePlayerRelive)
}

func handlePlayerRelive(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    switch agentUserData.GameMode{
       case datastruct.EndlessMode:
        ptr_endlessModeMatch.PlayerRelive(agentUserData.RoomID,agentUserData.PlayId)
    }
}

func handlePlayerLeftRoom(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }    
    agentUserData := a.UserData().(datastruct.AgentUserData)
    
    playerLeftRoom(agentUserData.ConnUUID,agentUserData.GameMode,agentUserData.RoomID)
}

func handlePlayersDied(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    
    agentUserData := a.UserData().(datastruct.AgentUserData)
    m := args[0].(*msg.CS_PlayerDied)
    
    //接收玩家死亡坐标,生成指定范围能量点
    //指定某一帧复活
    
    switch agentUserData.GameMode{
    case datastruct.SinglePersonMode:
         ptr_singleMatch.PlayersDied(agentUserData.RoomID,m.MsgContent)
    case datastruct.EndlessMode:
         ptr_endlessModeMatch.PlayersDied(agentUserData.RoomID,m.MsgContent)
    }
}

func handleEnergyExpended(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    m := args[0].(*msg.CS_EnergyExpended)
    expended:=m.MsgContent.EnergyExpended
    if expended>0{
        switch agentUserData.GameMode{
        case datastruct.SinglePersonMode:
             ptr_singleMatch.EnergyExpended(expended,agentUserData)
        case datastruct.EndlessMode:
             ptr_endlessModeMatch.EnergyExpended(expended,agentUserData)
        }
    }
}

func handlePlayerMoveData(args []interface{}){
    //测试
    //msg.Num = 0
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    r_id:=agentUserData.RoomID
    m := args[0].(*msg.CS_MoveData)
    
    //log.Debug("---1---player_id:%v,x:%v,y:%v,Speed:%v",agentUserData.PlayId,m.MsgContent.X,m.MsgContent.Y,m.MsgContent.Speed)
    
    switch agentUserData.GameMode{
    case datastruct.SinglePersonMode:
         ptr_singleMatch.PlayerMoved(r_id,agentUserData.PlayId,m)
    case datastruct.EndlessMode:
         ptr_endlessModeMatch.PlayerMoved(r_id,agentUserData.PlayId,m)
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
         ptr_endlessModeMatch.PlayerJoin(connUUID,m)
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
         ptr_endlessModeMatch.RemovePlayer(connUUID)
    }
}
 

//收到单人匹配消息的时候加入池，主动离开和自动离开在池中删除，
//完成单人匹配后，在池中删除
func handleSinglePersonMatching(args []interface{}) {
     startMatching(args,datastruct.SinglePersonMode)   
}

func handleEndlessModeMatching(args []interface{}){
     startMatching(args,datastruct.EndlessMode)
}

func removePlayerFromOtherMatchs(connUUID string,mode datastruct.GameModeType){
     switch mode{
      case datastruct.SinglePersonMode:
           log.Debug("SinglePersonMode remove connUUID from other matchs")
           removePlayer(connUUID,datastruct.EndlessMode)
      case datastruct.EndlessMode:
           log.Debug("EndlessMode remove connUUID from other matchs")
           removePlayer(connUUID,datastruct.SinglePersonMode)
     }
}

func removePlayer(key string,mode datastruct.GameModeType){
    switch mode{
      case datastruct.SinglePersonMode:
           ptr_singleMatch.RemovePlayer(key)
      case datastruct.EndlessMode:
           ptr_endlessModeMatch.RemovePlayer(key)
    }
    //在其他匹配模式中删除玩家
}

func playerLeftRoom(connUUID string,mode datastruct.GameModeType,r_id string){
    switch mode{
    case datastruct.SinglePersonMode:
         ptr_singleMatch.PlayerLeftRoom(connUUID)
    case datastruct.EndlessMode:
         ptr_endlessModeMatch.PlayerLeftRoom(r_id,connUUID)
    }
}

func startMatching(args []interface{},mode datastruct.GameModeType){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    
    uid:=agentUserData.Uid
    if uid <= 0{
        log.Error("Uid error : %v",uid)
        return
    }
    connUUID:=agentUserData.ConnUUID
    
    
    tools.ReSetAgentUserData(a,connUUID,uid,datastruct.NULLSTRING,mode,datastruct.NULLID)
    removePlayerFromOtherMatchs(connUUID,mode)
    
    if checkActionPool(connUUID,mode,a){
       return
    }
    
    matchingChanRPC(mode,connUUID,a,uid)
    
    var msgHeader json.MsgHeader
    msgHeader.MsgName = msg.SC_PlayerMatchingKey

    var msgContent msg.SC_PlayerMatchingContent
    msgContent.IsMatching =true

    a.WriteMsg(&msg.SC_PlayerMatching{  
        MsgHeader:msgHeader,
        MsgContent:msgContent,
    })
}

func checkActionPool(connUUID string,mode datastruct.GameModeType,a gate.Agent) bool {
    isMatching:=false
    switch mode{
     case datastruct.SinglePersonMode:
        if ptr_singleMatch.CheckActionPool(connUUID){//已在匹配中
            isMatching = true
        }
     case datastruct.EndlessMode:
        if ptr_endlessModeMatch.CheckActionPool(connUUID){//已在匹配中
            isMatching = true
        }
    }
    if isMatching{
        var msgHeader json.MsgHeader
        msgHeader.MsgName = msg.SC_PlayerAlreadyMatchingKey
        a.WriteMsg(&msg.SC_PlayerAlreadyMatching{
            MsgHeader:msgHeader,
        })
    }
    return isMatching
}

func matchingChanRPC(mode datastruct.GameModeType,connUUID string,a gate.Agent,uid int){
    var match match.ParentMatch
    switch mode{
     case  datastruct.SinglePersonMode:
         match = ptr_singleMatch
     case datastruct.EndlessMode:
         match = ptr_endlessModeMatch
    }
    ChanRPC.Go(MatchingKey,match,connUUID,a,uid)//玩家匹配
}