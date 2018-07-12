package match

import (
	"server/db"
	"server/msg"
	"server/datastruct"
	"sync"
    "github.com/name5566/leaf/log"
    "time"
    "server/tools"
)

type RoomDataType int //房间类型,匹配类型还是邀请类型

const (
    SinglePersonMatching RoomDataType = iota
    EndlessMode
	Invite
)

const min_MapWidth = 30
const min_MapHeight = 20
const time_interval = 50//50毫秒
var map_factor = 200

const MaxPeopleInRoom = 20 //每个房间最大人数
const RoomCloseTime = 15*time.Second//房间入口关闭时间

const FirstFrameIndex = 0//第一帧索引

const MaxPlayingTime = 5*time.Minute

const MaxEnergyPower = 5000 //全场最大能量值
const InitEnergyPower = 1000 //地图初始化的能量值
const PerFramePower = 30 //每帧能量30，1秒能量600
const InitEnergy_A=60
const InitEnergy_B=20
const PerFrameEnergy_A=2
const PerFrameEnergy_B=1

const offset = 400//出生点偏移量

const offsetFrames = (1000/time_interval)*2 //多少帧复活

type Room struct {
    Mutex *sync.RWMutex //读写互斥量
    IsOn bool //玩家是否能进入的开关
    players []string//玩家列表
    
    
    currentFrameIndex int//记录当前第几帧
    onlineSyncPlayers []datastruct.Player//同步完成的在线玩家列表,第0帧进来的玩家就已存在同步列表中
    offlineSyncPlayers []datastruct.Player//同步完成的离线玩家列表
    playersData *PlayersFrameData//玩家数据

    gameMap *GameMap //游戏地图
    
    unlockedData *RoomUnlockedData
    history *HistoryFrameData
    
    energyData *EnergyPowerData

    robots *RobotData

    energyExpend *EnergyExpend //历史帧能量消耗事件数据，保存消耗后的能量点数据。 用于计算生产能量数据

    diedData *PlayersDiedData
    
    leftList *LeftList

}


type LeftList struct {
     Mutex *sync.Mutex
     Data []string
}

type PlayersDiedData struct {
    Mutex *sync.Mutex //读写互斥量
    Data map[int]int //key:PlayerId,value:FrameIndex
}

type PlayerDied struct {//玩家的死亡
    Points []msg.EnergyPoint
    Action msg.PlayerIsDied
}





type HistoryFrameData struct {
    Mutex *sync.RWMutex //读写互斥量
    FramesData []*msg.SC_RoomFrameDataContent
}

type RoomUnlockedData struct {
     parentMatch ParentMatch
     leastPeople int
     isExistTicker bool
     ticker *time.Ticker
     points_ch chan []msg.EnergyPoint
     pointData *EnergyPointData
     allowList []string//允许列表
     roomType RoomDataType//房间类型
     roomId string
     startSync chan struct{} //开始同步的管道
     rebotMoveAction chan msg.Point
     rebotsNum int //生成的机器人个数
}

type GameMap struct{
    height int
    width int
}

type EnergyPointData struct{
     quadrant []msg.Quadrant
     firstFramePoint []msg.EnergyPoint //第一帧的能量点数据
}

type EnergyPowerData struct {
     EnableCreateEnergyPower int //当前可以生成的能量
}

type EnergyExpend struct {
    Mutex *sync.Mutex //读写互斥量
    Expended int 
    ConnUUID string
}

type PlayersFrameData struct {
     Mutex *sync.RWMutex //读写互斥量
     Data map[int]*PlayerActionData
}

type PlayerActionData struct {
    ActionType msg.ActionType
    Data interface{}//目前只能存一个动作,之后可能改进为每个单位存一组动作
}

type RobotData struct {
    Mutex *sync.RWMutex //读写互斥量
    robots map[int]*datastruct.Robot//机器人列表map[string]
    names map[int]string//key:robotNameId,value:robotName

    //for map , create actions ,  read
    //isrelive false,remove,  write
    //get robot, set isrelive = false, write
}




func CreateRoom(r_type RoomDataType,connUUIDs []string,r_id string,parentMatch ParentMatch,leastPeople int)*Room{
    room := new(Room)
    room.Mutex = new(sync.RWMutex)
    rebotsNum:=leastPeople-len(connUUIDs)
    room.createGameMap(map_factor)
    room.createRoomUnlockedData(r_type,connUUIDs,r_id,parentMatch,rebotsNum,leastPeople)
    room.createHistoryFrameData()
    room.createEnergyPowerData()
    room.createEnergyExpend()
    room.createPlayersDiedData()
    room.createLeftList(MaxPeopleInRoom)

    room.currentFrameIndex = FirstFrameIndex
    room.onlineSyncPlayers = make([]datastruct.Player,0,MaxPeopleInRoom)
    room.offlineSyncPlayers = make([]datastruct.Player,0,MaxPeopleInRoom-1)
    
    room.IsOn = true
    room.players = make([]string,0,MaxPeopleInRoom)
    room.playersData = NewPlayersFrameData()

      
    //测试
    //room.createRobotData(0,true)
    room.createRobotData(rebotsNum,true)
        
    room.gameStart()   
      
    return room
}

func (room *Room)removeFromRooms(){
     room.stopTicker()
     safeCloseRobotMoved(room.unlockedData.rebotMoveAction)
     safeClosePoint(room.unlockedData.points_ch)
     safeCloseSync(room.unlockedData.startSync)
     room.unlockedData.parentMatch.RemoveRoomWithID(room.unlockedData.roomId)
     db.Module.UpdateRobotNamesState(room.robots.names)
     log.Debug("room removeFromRooms")
}

func (room *Room)createGameMap(fac int){
    g_map:=new(GameMap)
    g_map.width = min_MapWidth*fac
    g_map.height = min_MapHeight*fac
    room.gameMap = g_map
}

func (room *Room)createEnergyPointData(width int,height int) *EnergyPointData{
    p_data:=new(EnergyPointData)
  
    diff:=200
    
    width=width-diff
    height=height-diff
    
    p_data.quadrant = make([]msg.Quadrant,0,4)
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,1))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,2))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,3))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,4))
    
    p_data.firstFramePoint=tools.GetRandomPoint(InitEnergy_A,InitEnergy_B,p_data.quadrant) //第零帧生成能量点
    
    return p_data
}

func(room *Room)IsSyncFinished(connUUID string,player *datastruct.Player) (bool,int){
    length:=len(room.players)

    play_id:=length+room.unlockedData.rebotsNum
    
    if length == MaxPeopleInRoom - 1 {
       room.IsOn = false
    }
    room.players=append(room.players,connUUID)
    
    var content msg.SC_InitRoomDataContent
    content.MapHeight = room.gameMap.height
    content.MapWidth = room.gameMap.width
    content.Interval = time_interval
    
    syncFinished:=false
    
    content.PlayId = play_id

    if room.currentFrameIndex == FirstFrameIndex{
       
        content.CurrentFrameIndex = FirstFrameIndex
        
        
        room.sendInitRoomDataToAgent(player,&content,play_id)

        room.onlineSyncPlayers=append(room.onlineSyncPlayers,*player)
        room.updateRobotRelive(length) 
        
        // var frame_data msg.FrameData
        // var frame_content msg.SC_RoomFrameDataContent

        // frame_data.FrameIndex = FirstFrameIndex
        // frame_data.CreateEnergyPoints = room.unlockedData.pointData.firstFramePoint
        // frame_content.FramesData = make([]msg.FrameData,0,1)
        
        // frame_data.PlayerFrameData=make([]interface{},0,1)

        room.GetCreateAction(play_id,datastruct.DefaultReliveFrameIndex,player.NickName)
        // frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
        // frame_content.FramesData = append(frame_content.FramesData,frame_data)
        syncFinished = true

    }else{
        content.CurrentFrameIndex = room.currentFrameIndex-1
        room.sendInitRoomDataToAgent(player,&content,play_id)
    }
    return syncFinished,content.CurrentFrameIndex
}

func (room *Room)GetCreateAction(play_id int,reliveFrameIndex int,playername string)*msg.PlayerRelive{
     if play_id == datastruct.NULLID{
        panic("GetCreateAction play_id error")
     }
     randomIndex:=tools.GetRandomQuadrantIndex()
     point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
     action:=msg.GetCreatePlayerAction(play_id,point.X,point.Y,reliveFrameIndex,playername)
     actionData:=new(PlayerActionData)
     actionData.ActionType = action.Action.Action
     actionData.Data = action
     room.playersData.Set(play_id,actionData)//添加action 到 lastFrameIndex+1
     return action
}

func (room *Room)sendInitRoomDataToAgent(player *datastruct.Player,content *msg.SC_InitRoomDataContent,play_id int){
     if room.unlockedData.roomType != EndlessMode {
        content.GameTime = 300 * 1000  - room.currentFrameIndex*50
     }
     content.GameMode = int(room.unlockedData.roomType)
     player.Agent.WriteMsg(msg.GetInitRoomDataMsg(*content))
     agentData:=player.Agent.UserData().(datastruct.AgentUserData)
     connUUID:=agentData.ConnUUID
     uid:=agentData.Uid
     rid:=room.unlockedData.roomId
     mode:=agentData.GameMode
     player.GameData.PlayId = play_id
     tools.ReSetAgentUserData(player.Agent,connUUID,uid,rid,mode,play_id,agentData.PlayName,agentData.Details)
}

func (room *Room)timeSleepWriteMsg(player *datastruct.Player,startIndex int) int {
      room.history.Mutex.RLock()
      var copyData []*msg.SC_RoomFrameDataContent
      if startIndex == 0{
         copyData=make([]*msg.SC_RoomFrameDataContent,len(room.history.FramesData))
         copy(copyData,room.history.FramesData)
      }else{
         copyData=room.history.FramesData[startIndex:]
      }
      room.history.Mutex.RUnlock()
      
      num:=len(copyData)
      lastFrameIndex:=copyData[num-1].FramesData[0].FrameIndex
      //log.Debug("timeSleepWriteMsg_0 num:%v",num)
      per:=1500

      if num>per{
        minLen:=10
        for i:=0;i<per;i++{
            player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(copyData[i]))
        }
        i:=1
        for {
            len_chan:=player.Agent.GetWriteChanlen()
            
            if len_chan < minLen{
               if i>=num/per{
                  //log.Debug("timeSleepWriteMsg_/ end,i=%v , num/per=%v",i,num/per)
                  break
               }
               
               for j:=0;j<per;j++{
                  player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(copyData[i*per+j]))
               }
               i++
            }
            // else{
            //     log.Debug("len_chan %v",len_chan)
            // }
        }
        if num%per!=0{
           lastIndex:=num-num%per
           for j:=0;j<num%per;j++{
               player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(copyData[lastIndex+j]))
               if j==num%per-1{
                  //log.Debug("timeSleepWriteMsg_mod end")
               }
           }
        }
      }else{
          for i:=0;i<num;i++{
              player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(copyData[i]))
          }
      }
      return lastFrameIndex
}

func (room *Room)syncData(connUUID string,player *datastruct.Player){
     
 
      lastFrameIndex:=room.timeSleepWriteMsg(player,0)
      
      var length int
      room.Mutex.Lock()
      if lastFrameIndex+1 == room.currentFrameIndex {//数据帧是从0开始，服务器计算帧是从1开始
               room.onlineSyncPlayers=append(room.onlineSyncPlayers,*player)
               log.Debug("syncData GetCreateAction")
               room.GetCreateAction(player.GameData.PlayId,datastruct.DefaultReliveFrameIndex,player.NickName)
               length=len(room.players)
               room.Mutex.Unlock()
               room.updateRobotRelive(length)
      }else{
               room.Mutex.Unlock()
               ok:=true
               for ok {
                   if _, ok = <-room.unlockedData.startSync; ok {
                       startIndex:=lastFrameIndex+1
                       lastFrameIndex:=room.timeSleepWriteMsg(player,startIndex)
                       isSyncFinished:=false
                       room.Mutex.Lock()
                       if lastFrameIndex+1 == room.currentFrameIndex {
                           isSyncFinished = true
                           room.onlineSyncPlayers=append(room.onlineSyncPlayers,*player)
                           log.Debug("syncData Channel GetCreateAction")
                           room.GetCreateAction(player.GameData.PlayId,datastruct.DefaultReliveFrameIndex,player.NickName)
                           length=len(room.players)
                       }
                       room.Mutex.Unlock()
                       if isSyncFinished{
                           room.updateRobotRelive(length)
                           log.Debug("Channel SyncFinished")
                           break
                       }
                   }
               }
            }
}

func(room *Room)Join(connUUID string,player datastruct.Player,force bool) bool{
    isOn:=false
    syncFinished:=false
    currentFrameIndex:=-1
    room.Mutex.Lock()
    if force{
       syncFinished,currentFrameIndex=room.IsSyncFinished(connUUID,&player)
       isOn = true
    }else{
       isOn=room.IsOn
       if isOn{
          syncFinished,currentFrameIndex=room.IsSyncFinished(connUUID,&player)
       }
    }
    room.Mutex.Unlock()
    if currentFrameIndex==FirstFrameIndex{
       room.getEnergyExpended(connUUID)
       room.createTicker()
    }
    if isOn&&!syncFinished{
       go room.syncData(connUUID,&player)
    }
    return isOn
}

func (room*Room)goCreatePoints(num int,maxRangeType msg.EnergyPointType){
     for {
        points:=tools.GetRandomPoint(PerFrameEnergy_A,PerFrameEnergy_B,room.unlockedData.pointData.quadrant)
        isClosed := safeSendPoint(room.unlockedData.points_ch,points)
        if isClosed{
            break
        }
     }
}

func (room*Room)goCreateMovePoint(){
    for {
       isClosed := safeSendRobotMoved(room.unlockedData.rebotMoveAction,tools.GetRandomDirection())
       if isClosed{
           break
       }
    }
}


func(room *Room)createTicker(){
	if !room.unlockedData.isExistTicker{
        room.unlockedData.isExistTicker = true
        room.unlockedData.ticker = time.NewTicker(time_interval*time.Millisecond)
        go room.selectTicker()
    }
    
}

func(room *Room)stopTicker(){
    if room.unlockedData.ticker != nil{
        room.unlockedData.ticker.Stop()
        room.unlockedData.isExistTicker=false
    }
}

func(room *Room) selectTicker(){
     for {
			select {
            case <-room.unlockedData.ticker.C:
				room.ComputeFrameData()
			}
	 }
}

func (room *Room)IsRemoveRoom()(bool,int,[]datastruct.Player,[]datastruct.Player,int,bool,string){
 //判断在线玩家

 isRemove:=false  

 room.Mutex.Lock()
 defer room.Mutex.Unlock()
 
 p_num:=len(room.players)
 onlinePlayers:=room.unlockedData.parentMatch.GetOnlinePlayersPtr()
 offlinePlayersUUID:=make([]string,0,p_num)
 expended_onlineConnUUID:=room.getEnergyExpendedConnUUID()
 for _,connUUID := range room.players{
    tf:=onlinePlayers.IsExist(connUUID)
    if !tf{
        offlinePlayersUUID = append(offlinePlayersUUID,connUUID)
    }else{
        if expended_onlineConnUUID == connUUID{
           expended_onlineConnUUID = datastruct.NULLSTRING
        }
    }
 }

 offlineNum:=len(offlinePlayersUUID)
 onlinePlayersInRoom:=p_num-offlineNum

 if onlinePlayersInRoom == 0{
    room.IsOn = false
    isRemove = true
 }
    
    var currentFrameIndex int
    var online_sync []datastruct.Player
    var offline_sync []datastruct.Player
    
    if !isRemove{
       currentFrameIndex = room.currentFrameIndex //服务器帧是从1开始
       room.currentFrameIndex++

       var offlineSyncPlayersIndex []int
       if offlineNum > 0{
          offlineSyncPlayersIndex=make([]int,0,offlineNum)
       }
       for _,uuid := range offlinePlayersUUID{
           for index,player := range room.onlineSyncPlayers{
             agentData:=player.Agent.UserData().(datastruct.AgentUserData)
             if agentData.ConnUUID == uuid{
              offlineSyncPlayersIndex = append(offlineSyncPlayersIndex,index)
              room.offlineSyncPlayers = append(room.offlineSyncPlayers,player)
             }
           }
           removeOfflineSyncPlayersInRoom(room,offlineSyncPlayersIndex)//remove offline players
       }
       online_sync=make([]datastruct.Player,len(room.onlineSyncPlayers))
       copy(online_sync,room.onlineSyncPlayers)
       offline_sync=make([]datastruct.Player,len(room.offlineSyncPlayers))
       copy(offline_sync,room.offlineSyncPlayers)
       if expended_onlineConnUUID != datastruct.NULLSTRING{
        agentData:=online_sync[0].Agent.UserData().(datastruct.AgentUserData)
        expended_onlineConnUUID = agentData.ConnUUID 
       }
    }

 syncNotFinishedPlayers:=onlinePlayersInRoom-len(online_sync)
 isRemoveHistory:=false
 if syncNotFinishedPlayers == 0&&!room.IsOn{
    isRemoveHistory = true
 }
 
 return isRemove,currentFrameIndex,online_sync,offline_sync,syncNotFinishedPlayers,isRemoveHistory,expended_onlineConnUUID
}

func (room *Room)ComputeFrameData(){
     isRemove,currentFrameIndex,online_sync,offline_sync,syncNotFinishedPlayers,isRemoveHistory,expended_onlineConnUUID:=room.IsRemoveRoom()
     if isRemove{
        room.removeFromRooms()
        return
     }

    var frame_content msg.SC_RoomFrameDataContent
     
    frame_content.FramesData = make([]msg.FrameData,0,1)
    var frame_data msg.FrameData
    frame_data.FrameIndex = currentFrameIndex

    frame_data.PlayerFrameData=make([]interface{},0,len(online_sync)+len(offline_sync))

    if currentFrameIndex == FirstFrameIndex{//已保存在历史消息中，清空初始化的能量点
        frame_data.CreateEnergyPoints=room.unlockedData.pointData.firstFramePoint
        room.unlockedData.pointData.firstFramePoint=room.unlockedData.pointData.firstFramePoint[:0]
     }else{
        var points []msg.EnergyPoint
        select {
        case points = <-room.unlockedData.points_ch:
        default:
         points=nil
        }
        expended:=room.getEnergyExpended(expended_onlineConnUUID)
        room.energyData.SetPower(expended)
        if points != nil && len(points)>0 && room.energyData.IsCreatePower(){
            frame_data.CreateEnergyPoints = points
        }
     }
   
     room.robots.Mutex.Lock()
     removeRobotsId:=make([]int,0,len(room.robots.robots))
     for _,robot:= range room.robots.robots{
        action_type,action:=room.getRobotAction(robot,currentFrameIndex)
        if action != nil{
            if action_type==msg.Death{
                died:=action.(*PlayerDied)
                action=died.Action
                frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,died.Points...)
                if !robot.IsRelive{
                   removeRobotsId=append(removeRobotsId,robot.Id)
                }
            }
            frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
         }
     }
     for _,v:= range removeRobotsId{
        delete(room.robots.robots,v)
     }
     room.robots.Mutex.Unlock()
     
     for _,player := range online_sync{
         action_type,action:=room.playersData.GetValue(player.NickName,player.GameData.PlayId,currentFrameIndex,room)
         if action != nil{
             if action_type==msg.Death{
                died:=action.(*PlayerDied)
                action=died.Action
                frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,died.Points...)
             }
             frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
         }
     }
     
     for _,player := range offline_sync{
        connUUID:=player.Agent.UserData().(datastruct.AgentUserData).ConnUUID
        action_type,action:=room.playersData.GetOfflineAction(player.NickName,player.GameData.PlayId,currentFrameIndex,room,connUUID)
        if action != nil{
            if action_type==msg.Death{
               died:=action.(*PlayerDied)
               action=died.Action
               frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,died.Points...)
            }
            frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
        }
     }
 
     frame_content.FramesData = append(frame_content.FramesData,frame_data)
     msg:=msg.GetRoomFrameDataMsg(&frame_content)

   
     for _,player := range online_sync{
         player.Agent.WriteMsg(msg)
     }
     
    
     if !isRemoveHistory{
        room.history.Mutex.Lock()
        room.history.FramesData = append(room.history.FramesData,&frame_content)
        room.history.Mutex.Unlock()
        
        for i:=0;i<syncNotFinishedPlayers;i++{
            isClosed:=safeSendSync(room.unlockedData.startSync,struct{}{})
            if isClosed{
                break
            }
        }
     }else{
        if room.history!=nil{
            room.history.Mutex.Lock()
            room.history.FramesData=room.history.FramesData[:0]
            room.history.Mutex.Unlock()
            room.history = nil
        }
     }
     //log.Debug("Compute FramesData:%v,",frame_content.FramesData)
}

func removeOfflineSyncPlayersInRoom(room *Room,removeIndex []int){
    rm_count:=0
    for index,v := range removeIndex {
        if index!=0{
           v = v-rm_count
        }
        room.onlineSyncPlayers=append(room.onlineSyncPlayers[:v], room.onlineSyncPlayers[v+1:]...)
        rm_count++;
    }
}


func (room *Room)createRoomUnlockedData(r_type RoomDataType,connUUIDs []string,r_id string,parentMatch ParentMatch,rebotsNum int,leastPeople int){
    unlockedData:=new(RoomUnlockedData)
    unlockedData.points_ch = make(chan []msg.EnergyPoint,2)
    unlockedData.rebotMoveAction = make(chan msg.Point,leastPeople-1+MaxPeopleInRoom-1)
    unlockedData.startSync = make(chan struct{},MaxPeopleInRoom-1)
    unlockedData.pointData = room.createEnergyPointData(room.gameMap.width,room.gameMap.height)
    unlockedData.allowList = connUUIDs
    unlockedData.roomId = r_id
    unlockedData.isExistTicker = false
    unlockedData.parentMatch = parentMatch
    unlockedData.rebotsNum = rebotsNum
    unlockedData.leastPeople = leastPeople
    unlockedData.roomType = r_type
    room.unlockedData = unlockedData
    go room.goCreatePoints(1,msg.TypeB)
    go room.goCreateMovePoint()
}

func (room *Room)createHistoryFrameData(){
    history:=new(HistoryFrameData)
    history.Mutex = new(sync.RWMutex)
    rs:=MaxPlayingTime/(time_interval*time.Millisecond)
    history.FramesData = make([]*msg.SC_RoomFrameDataContent,0,rs);
    room.history = history
}

func (room *Room)createEnergyPowerData(){
    energyData:=new(EnergyPowerData)
    energyData.EnableCreateEnergyPower = MaxEnergyPower - InitEnergyPower
    room.energyData = energyData
}

func (room *Room)createRobotData(num int,isRelive bool){
    robots:=new(RobotData)
    robots.Mutex = new(sync.RWMutex)
    robots.robots = make(map[int]*datastruct.Robot)
    if num > 0{
        robots.names = db.Module.GetRobotNames(num)
        i:=0;
        for _,v := range robots.names{
            robot:=tools.CreateRobot(v,i,isRelive,room.unlockedData.pointData.quadrant,datastruct.DefaultReliveFrameIndex)
            robots.robots[i]=robot
            i++
        }
    }
    room.robots = robots
}

func (room *Room)createEnergyExpend(){
    expend:=new(EnergyExpend)
    expend.Mutex = new(sync.Mutex)
    expend.Expended = 0
    room.energyExpend = expend
}

func (room *Room)createPlayersDiedData(){
    diedData:=new(PlayersDiedData)
    diedData.Mutex = new(sync.Mutex)
    diedData.Data = make(map[int]int)
    room.diedData=diedData
}


func (diedData *PlayersDiedData)isRemovePlayerId(current_pid int,frameIndex int,room *Room) bool{
     tf:=false
     if current_pid < room.unlockedData.rebotsNum{
        room.robots.Mutex.RLock()
        _,ok:=room.robots.robots[current_pid]
        if !ok{
           tf = true
        }
        room.robots.Mutex.RUnlock()
     }else{
        room.playersData.Mutex.RLock()
        _,ok:=room.playersData.Data[current_pid]
        if !ok{
            tf = true
            
         }
        room.playersData.Mutex.RUnlock()
     }
     if tf{
        return true
     }else{
        last_frameIndex,ok:=diedData.Data[current_pid]
        if ok{
               if frameIndex<last_frameIndex||(frameIndex>last_frameIndex&&frameIndex-offsetFrames<last_frameIndex){
                  //log.Debug("HandleDiedData_1 last_frameIndex:%v,frameIndex:%v",last_frameIndex,frameIndex)
                  return true
               }
        }
     }
     return false
}

func (diedData *PlayersDiedData)Append(values []msg.PlayerDiedData){
    for _,v := range values{
         p_id:=v.PlayerId
         frameIndex:=v.FrameIndex
         diedData.Data[p_id]=frameIndex
    }
}


func (room *Room)getMovePoint() msg.Point{
    var point msg.Point
    select {
    case point = <-room.unlockedData.rebotMoveAction:
    default:
        point=msg.DefaultDirection
    }
    return point
}


func safeSendRobotMoved(ch chan msg.Point, value msg.Point) (closed bool) {
    defer func() {
        if recover() != nil {
            closed = true
        }
	}()
    ch <- value // panic if ch is closed
    return false // <=> closed = false; return
}

func safeCloseRobotMoved(ch chan msg.Point) (justClosed bool) {
	defer func() {
        if recover() != nil {
            justClosed = false
        }
	}()
	close(ch) // panic if ch is closed
    return true
}

func safeSendPoint(ch chan []msg.EnergyPoint, value []msg.EnergyPoint) (closed bool) {
    defer func() {
        if recover() != nil {
            closed = true
        }
	}()
    ch <- value // panic if ch is closed
    return false // <=> closed = false; return
}

func safeClosePoint(ch chan []msg.EnergyPoint) (justClosed bool) {
	defer func() {
        if recover() != nil {
            justClosed = false
        }
	}()
	close(ch) // panic if ch is closed
    return true
}

func safeSendSync(ch chan struct{}, value struct{}) (closed bool) {
    defer func() {
        if recover() != nil {
            closed = true
        }
	}()
    ch <- value // panic if ch is closed
    return false // <=> closed = false; return
}

func safeCloseSync(ch chan struct{}) (justClosed bool) {
	defer func() {
        if recover() != nil {
            justClosed = false
        }
	}()
	close(ch) // panic if ch is closed
    return true
}


func NewPlayersFrameData() *PlayersFrameData {
	return &PlayersFrameData{
		Mutex: new(sync.RWMutex),
		Data:   make(map[int]*PlayerActionData),//key is play_id
	}
}

func (data *PlayersFrameData)Set(k int,v *PlayerActionData){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
	data.Data[k] = v
}

func (data *PlayersFrameData)CheckValue(k int) (*PlayerActionData,bool){
    data.Mutex.RLock()
    defer data.Mutex.RUnlock()
    actionData, ok := data.Data[k]
    return actionData,ok
}


func (power *EnergyPowerData)IsCreatePower()bool{
     tf:=false
     if power.EnableCreateEnergyPower>=PerFramePower{
        power.EnableCreateEnergyPower-=PerFramePower
        tf = true
     }
     return tf
}

func (power *EnergyPowerData)SetPower(num int){
    power.EnableCreateEnergyPower += num
    if power.EnableCreateEnergyPower>MaxEnergyPower{
       power.EnableCreateEnergyPower = MaxEnergyPower 
    }
}

func (data *PlayersFrameData)GetValue(name string,pid int,currentFrameIndex int,room *Room)(msg.ActionType,interface{}){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
    var v interface{}
    actionData,ok:= data.Data[pid]
    rs_actionType:=msg.NullAction
    v = nil
    if ok{
        switch actionData.ActionType {
          case msg.Create:
               p_relive:=actionData.Data.(*msg.PlayerRelive)
               if p_relive.ReLiveFrameIndex == datastruct.DefaultReliveFrameIndex || p_relive.ReLiveFrameIndex == currentFrameIndex{
                  v = p_relive.Action
                  rs_actionType = msg.Create
                  action:=msg.GetCreatePlayerMoved(pid,msg.DefaultDirection.X,msg.DefaultDirection.Y,msg.DefaultSpeed)
                  actionData.ActionType = action.Action
                  actionData.Data = action
               }else{
                  v = nil
               }
          case msg.Death:
             v=actionData.Data
             rs_actionType = msg.Death
             isCreate:=false
             reliveFrameIndex:=datastruct.DefaultReliveFrameIndex
             switch room.unlockedData.roomType{
             case SinglePersonMatching:
                  isCreate = true
                  reliveFrameIndex=currentFrameIndex+offsetFrames
             case EndlessMode:
                  delete(data.Data,pid)
             }
             if isCreate{
                room.relive(actionData,pid,reliveFrameIndex,name)
             }
          case msg.Move:
             v= *(actionData.Data.(*msg.PlayerMoved))
             rs_actionType = msg.Move    
        }
    }
    return rs_actionType,v
}

func (data *PlayersFrameData)GetOfflineAction(name string,pid int,currentFrameIndex int,room *Room,connUUID string)(msg.ActionType,interface{}){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
    var v interface{}
    actionData,ok:= data.Data[pid]
    rs_actionType:=msg.NullAction
    v = nil
    if ok{
        switch actionData.ActionType {
          case msg.Create:
               p_relive:=actionData.Data.(*msg.PlayerRelive)
               if p_relive.ReLiveFrameIndex == datastruct.DefaultReliveFrameIndex || p_relive.ReLiveFrameIndex == currentFrameIndex{
                  v = p_relive.Action
                  rs_actionType = msg.Create
                  point:=room.getMovePoint()
                  action:=msg.GetCreatePlayerMoved(pid,point.X,point.Y,msg.DefaultSpeed)
                  actionData.ActionType = action.Action
                  actionData.Data = action
               }else{
                  v = nil
               }
          case msg.Death:
             v=actionData.Data
             rs_actionType = msg.Death
             isCreate:=false
             reliveFrameIndex:=datastruct.DefaultReliveFrameIndex
             switch room.unlockedData.roomType{
               case SinglePersonMatching:
                    isCreate = true
                    reliveFrameIndex=currentFrameIndex+offsetFrames
               case EndlessMode:
                    isExist:=room.CheckLeftlist(connUUID)
                    if isExist{
                       room.removePlayer(connUUID)
                    }
             }
             if isCreate{
                room.relive(actionData,pid,reliveFrameIndex,name)
             }
          case msg.Move:
            rs_actionType = msg.Move
            switch actionData.Data.(type){
            case *msg.PlayerMoved:
                 
                 move_action:=actionData.Data.(*msg.PlayerMoved)
                 v= *move_action
                 offlinemove_action:=tools.CreateOfflinePlayerMoved(currentFrameIndex,move_action)
                 actionData.ActionType = offlinemove_action.Action.Action
                 actionData.Data = offlinemove_action
                 
            case *msg.OfflinePlayerMoved:
                  
                 offlinemove_action:=actionData.Data.(*msg.OfflinePlayerMoved)
                 startIndex:=offlinemove_action.StartFrameIndex
                 directionInterval:=offlinemove_action.DirectionInterval
                 speedInterval:=offlinemove_action.SpeedInterval

                 var ptr_action *msg.PlayerMoved
                  
                 if (currentFrameIndex-startIndex)*time_interval % (directionInterval*1000) == 0 {
                     lastSpeed:=offlinemove_action.Action.Speed
                     point:=room.getMovePoint()
                     ptr_action = msg.GetCreatePlayerMoved(pid,point.X,point.Y,lastSpeed)
                 }else{
                     ptr_action = &offlinemove_action.Action
                 }

                 if (currentFrameIndex-startIndex)*time_interval % (speedInterval*1000) == 0{
                    speedDuration:= tools.GetRandomSpeedDuration()
                    offlinemove_action.StopSpeedFrameIndex = currentFrameIndex-startIndex+speedDuration*(1000/time_interval)
                    ptr_action.Speed = tools.GetRandomSpeed()
                }
    
                if offlinemove_action.StopSpeedFrameIndex != 0 && currentFrameIndex-startIndex == offlinemove_action.StopSpeedFrameIndex{
                    offlinemove_action.StopSpeedFrameIndex = 0
                    ptr_action.Speed = msg.DefaultSpeed
                }
                
                offlinemove_action.Action = *ptr_action
                actionData.ActionType = ptr_action.Action
                actionData.Data = offlinemove_action
                v = *ptr_action
            }

        }
    }
    return rs_actionType,v
}





func (room *Room)getRobotAction(robot *datastruct.Robot,currentFrameIndex int)(msg.ActionType,interface{}){
     var rs interface{}
     current_action:=robot.Action
     
     var rs_type msg.ActionType
     switch current_action.(type){
       case *msg.PlayerRelive:
            p_relive:=current_action.(*msg.PlayerRelive)
            
            if p_relive.ReLiveFrameIndex == datastruct.DefaultReliveFrameIndex || p_relive.ReLiveFrameIndex == currentFrameIndex{
                 rs = p_relive.Action
                 rs_type = msg.Create
                 point:=room.getMovePoint()
                 action:=msg.GetCreatePlayerMoved(robot.Id,point.X,point.Y,msg.DefaultSpeed)
                 robot.Action = action
            }else{
                rs = nil
            }

       case *msg.PlayerMoved:
            var ptr_action *msg.PlayerMoved
            if currentFrameIndex*time_interval % (robot.DirectionInterval*1000) == 0 {
                action:=current_action.(*msg.PlayerMoved)
                lastSpeed:=action.Speed
                point:=room.getMovePoint()
                msg.UpdatePlayerMoved(action,point.X,point.Y,lastSpeed)
                ptr_action = action
            }else{
                ptr_action = current_action.(*msg.PlayerMoved)
            }
            
            if currentFrameIndex*time_interval % (robot.SpeedInterval*1000) == 0{
                speedDuration:= tools.GetRandomSpeedDuration()
                robot.StopSpeedFrameIndex = currentFrameIndex+speedDuration*(1000/time_interval)
                ptr_action.Speed = tools.GetRandomSpeed()
            }

            if robot.StopSpeedFrameIndex != 0 && currentFrameIndex == robot.StopSpeedFrameIndex{
                robot.StopSpeedFrameIndex = 0
                ptr_action.Speed = msg.DefaultSpeed
            }
            
            robot.Action=ptr_action
            rs=*ptr_action
            rs_type = msg.Move
            
        case *PlayerDied:
            rs = current_action //PlayerDied
            rs_type = msg.Death
            if robot.IsRelive {
                action:=tools.GetCreateRobotAction(robot.Id,room.unlockedData.pointData.quadrant,offsetFrames+currentFrameIndex,robot.NickName)
                robot.Action=action
            }
     }
	 return rs_type,rs
}

func (room *Room)EnergyExpended(uuid string,expended int){
      room.energyExpend.Mutex.Lock()
      defer room.energyExpend.Mutex.Unlock()
      if uuid == room.energyExpend.ConnUUID{
         room.energyExpend.Expended += expended
      }
}

func (room *Room)getEnergyExpended(uuid string) int {
    room.energyExpend.Mutex.Lock()
    defer room.energyExpend.Mutex.Unlock()
    if uuid != datastruct.NULLSTRING{
       room.energyExpend.ConnUUID = uuid
    }
    rs := room.energyExpend.Expended
    room.energyExpend.Expended = 0
    return rs
}

func (room *Room)getEnergyExpendedConnUUID() string {
    room.energyExpend.Mutex.Lock()
    defer room.energyExpend.Mutex.Unlock()
    connUUID:=room.energyExpend.ConnUUID
    return connUUID
}

func (room *Room)updateRobotRelive(playersNum int){
      if playersNum > 1&&playersNum<=room.unlockedData.leastPeople{
         room.robots.Mutex.RLock()
         for _,v := range room.robots.robots{
             if v.IsRelive{
                v.IsRelive = false
                break 
             }
         }
         room.robots.Mutex.RUnlock()
      }
}

func (room *Room)IsEnableUpdatePlayerAction(PlayerId int) bool{
    if v,ok:=room.playersData.CheckValue(PlayerId);ok{
        if v.ActionType == msg.Create||v.ActionType == msg.Death{
           return false
        }
    }
    return true
}

func (room *Room)GetPlayerMovedMsg(PlayerId int,moveData *msg.CS_MoveData){
      room.playersData.Mutex.RLock()
      defer room.playersData.Mutex.RUnlock()
      actionData,ok:=room.playersData.Data[PlayerId]
      if ok{
         action:=msg.GetCreatePlayerMoved(PlayerId,moveData.MsgContent.X,moveData.MsgContent.Y,moveData.MsgContent.Speed)
         actionData.ActionType = action.Action
         if action.X == 0 && action.Y == 0{
            switch actionData.Data.(type){
                   case *msg.PlayerMoved:
                        last_action:=actionData.Data.(*msg.PlayerMoved)
                        action.X = last_action.X
                        action.Y = last_action.Y
                   default:
                      action.X = msg.DefaultDirection.X
                      action.Y = msg.DefaultDirection.Y
            }
         }
         actionData.Data = action
      }
}
func (room *Room)HandleDiedData(values []msg.PlayerDiedData){
    room.diedData.Mutex.Lock()
    if len(room.diedData.Data)>0{
            removeIndexs:=make([]int,0,len(values))
            room.Mutex.RLock()
            currentFrameIndex:=room.currentFrameIndex-1
            room.Mutex.RUnlock()
            for index,v := range values{
                isRemove:=false
                if v.FrameIndex>currentFrameIndex{
                   isRemove = true
                   log.Debug("HandleDiedData_0 v.FrameIndex:%v,currentFrameIndex:%v",v.FrameIndex,currentFrameIndex)
                }else{
                   isRemove =room.diedData.isRemovePlayerId(v.PlayerId,v.FrameIndex,room)

                }
                if isRemove {
                   removeIndexs = append(removeIndexs,index)//2秒内死亡,去重
                }
            }
            rm_count:=0
            for index,v := range removeIndexs {
                if index!=0{
                    v = v-rm_count
                 }
                 values=append(values[:v], values[v+1:]...)
                 rm_count++;
            }
            room.diedData.Append(values)
     }else{
        room.diedData.Append(values)
     }
     room.diedData.Mutex.Unlock()
     
     
     for _,v := range values{
         p_id:=v.PlayerId
        
        //  points:=v.Points

        //  arr:=make([]msg.EnergyPoint,0,len(points))

        //  for _,point := range points{
        //     x:= point.X
        //     y:= point.Y
        //     new_x:=int(x)
        //     new_y:=int(y)
        //     var e_point msg.EnergyPoint
        //     e_point.Type = int(msg.TypeC)
        //     e_point.X = new_x
        //     e_point.Y = new_y
        //     arr = append(arr,e_point)
        //  }
         
         var action msg.PlayerIsDied
         action.Action = msg.Death
         action.PlayerId = p_id
         
         p_died:=new(PlayerDied)
         p_died.Points = v.Points
         p_died.Action = action
   
        
         if p_id < room.unlockedData.rebotsNum{
            room.robots.Mutex.RLock()
            robot,ok:=room.robots.robots[p_id]
            if ok{
                robot.Action = p_died 
            }
            room.robots.Mutex.RUnlock()
         }else{
            room.playersData.Mutex.RLock()
            defer room.playersData.Mutex.RUnlock()
            action_data,ok:=room.playersData.Data[p_id]
            if ok{
               action_data.Data = p_died
               action_data.ActionType = msg.Death
            }
         }
     }
}
func (room *Room)GetAllowList()[]string{
     return room.unlockedData.allowList 
}

func (room *Room)gameStart(){
    if room.unlockedData.roomType == SinglePersonMatching{
        time.AfterFunc(MaxPlayingTime,func(){
            content:=new(msg.SC_GameOverDataContent)
            content.RoomId = room.unlockedData.roomId
            msg:=msg.GetGameOverMsg(content)
            
            room.Mutex.RLock()
            for _,player := range room.onlineSyncPlayers{
                player.Agent.WriteMsg(msg)
            }
            room.Mutex.RUnlock()

            room.removeFromRooms()
            
            log.Debug("----------Game Over----------")
        })
    }
    time.AfterFunc(RoomCloseTime,func(){
        room.Mutex.Lock()
        room.IsOn = false
        room.Mutex.Unlock()
    })
}

func (room *Room)createLeftList(cap int){
    leftList:=new(LeftList)
    leftList.Mutex = new(sync.Mutex)
    leftList.Data = make([]string,0,cap)
    room.leftList = leftList
}

func (room *Room)AddPlayerleft(connUUID string){
    room.leftList.Mutex.Lock()
    defer room.leftList.Mutex.Unlock()
    room.leftList.Data = append(room.leftList.Data,connUUID)
}

func (room *Room)CheckLeftlist(connUUID string)bool{
    isExist:=false
    rm_index:=-1
    room.leftList.Mutex.Lock()
    defer room.leftList.Mutex.Unlock()
    for index,v := range room.leftList.Data{
        if connUUID == v{
            rm_index = index
            isExist = true
            break
        }
    }
    if isExist{
        room.leftList.Data=append(room.leftList.Data[:rm_index], room.leftList.Data[rm_index+1:]...)
    }
    return isExist
}

func (room *Room)removePlayer(connUUID string){
    room.Mutex.Lock()
    defer room.Mutex.Unlock()
    rm_index:=datastruct.NULLID
    for index,v := range room.players{
        if connUUID == v{
            rm_index=index
            break
        }
    }
    if rm_index!=datastruct.NULLID{
        room.players=append(room.players[:rm_index], room.players[rm_index+1:]...)
    }
}

func (room *Room)PlayerRelive(pid int,playername string){
     actionData,tf:=room.playersData.CheckValue(pid)
     if !tf{
        actionData=new(PlayerActionData)
        room.relive(actionData,pid,datastruct.DefaultReliveFrameIndex,playername)
        room.playersData.Set(pid,actionData)
     }
}
func (room *Room)relive(actionData *PlayerActionData,pid int,reliveFrameIndex int,playername string){
     randomIndex:=tools.GetRandomQuadrantIndex()
     point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
     action:=msg.GetCreatePlayerAction(pid,point.X,point.Y,reliveFrameIndex,playername)
     actionData.ActionType = action.Action.Action
     actionData.Data = action
}