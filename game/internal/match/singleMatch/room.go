package singleMatch

import (
	"server/msg"
	"server/datastruct"
	"sync"
    "github.com/name5566/leaf/log"
    "time"
    "server/tools"
)

// type RoomDataType int //房间类型,匹配类型还是邀请类型

// const (
// 	SinglePersonMatching RoomDataType = iota
// 	Invite
// )


const min_MapWidth = 20
const min_MapHeight = 15
const time_interval = 50//50毫秒
var map_factor = 5*40



const MaxPeopleInRoom = 20 //每个房间最大人数
const RoomCloseTime = 15.0*time.Second//房间入口关闭时间

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
}

type PlayersDiedData struct {
    MaxCleanTime time.Duration
    Mutex *sync.Mutex //读写互斥量
    Data []PlayersDiedDataForTime
}

type PlayersDiedDataForTime struct {
    DiedTime time.Time
    Players  []int //play_id
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
     parentMatch *SingleMatch
     isExistTicker bool
     ticker *time.Ticker
     points_ch chan []msg.EnergyPoint
     pointData *EnergyPointData
     AllowList []string//允许列表
     //RoomType RoomDataType//房间类型
     RoomId string
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
     Data map[int]PlayerActionData
}

type PlayerActionData struct {
    ActionType msg.ActionType
    Data interface{}//目前只能存一个动作,之后可能改进为每个单位存一组动作
}

type RobotData struct {
    Mutex *sync.RWMutex //读写互斥量
    robots map[int]*datastruct.Robot//机器人列表map[string]
    //for map , create actions ,  read
    //isrelive false,remove,  write
    //get robot, set isrelive = false, write
    
}




func CreateRoom(connUUIDs []string,r_id string,parentMatch *SingleMatch)*Room{
    room := new(Room)
    room.Mutex = new(sync.RWMutex)
    rebotsNum:=LeastPeople-len(connUUIDs)
    room.createGameMap(map_factor)
    room.createRoomUnlockedData(connUUIDs,r_id,parentMatch,rebotsNum)
    room.createHistoryFrameData()
    room.createEnergyPowerData()
    room.createEnergyExpend()
    room.createPlayersDiedData()
    
    room.currentFrameIndex = FirstFrameIndex
    room.onlineSyncPlayers = make([]datastruct.Player,0,MaxPeopleInRoom)
    room.offlineSyncPlayers = make([]datastruct.Player,0,MaxPeopleInRoom-1)
    
    room.IsOn = true
    room.players = make([]string,0,MaxPeopleInRoom)
    room.playersData = NewPlayersFrameData()

        log.Debug("create Matching Room")
        //测试
        room.createRobotData(1,true)
        //room.createRobotData(rebotsNum,true)
        
        time.AfterFunc(RoomCloseTime,func(){
            isRemove:=false
            room.Mutex.Lock()
            room.IsOn = false
            length:= len(room.players)
            if length <=0{
                isRemove = true
            }
            room.Mutex.Unlock()
            if isRemove{
                room.removeFromRooms()
            }
        })
      
    return room
}

func (room *Room)removeFromRooms(){
     room.stopTicker()
     safeCloseRobotMoved(room.unlockedData.rebotMoveAction)
     safeClosePoint(room.unlockedData.points_ch)
     safeCloseSync(room.unlockedData.startSync)
     room.unlockedData.parentMatch.removeRoomWithID(room.unlockedData.RoomId)
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
    if length == MaxPeopleInRoom - 1 {
       room.IsOn = false
    }
    room.players=append(room.players,connUUID)
    
    var content msg.SC_InitRoomDataContent
    content.MapHeight = room.gameMap.height
    content.MapWidth = room.gameMap.width
    content.Interval = time_interval
    
    syncFinished:=false
    
    play_id:=len(room.players)+room.unlockedData.rebotsNum



    content.PlayId = play_id

    if room.currentFrameIndex == FirstFrameIndex{
       
        content.CurrentFrameIndex = FirstFrameIndex
        
       
        room.SendInitRoomDataToAgent(player,&content,play_id)

        room.onlineSyncPlayers=append(room.onlineSyncPlayers,*player)
        room.updateRobotRelive(length) 
        

        var frame_data msg.FrameData
        var frame_content msg.SC_RoomFrameDataContent

        frame_data.FrameIndex = FirstFrameIndex
        frame_data.CreateEnergyPoints = room.unlockedData.pointData.firstFramePoint
        frame_content.FramesData = make([]msg.FrameData,0,1)
        
        frame_data.PlayerFrameData=make([]interface{},0,1)

        
        action:=room.GetCreateAction(play_id,msg.DefaultReliveFrameIndex)
        frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
        
        frame_content.FramesData = append(frame_content.FramesData,frame_data)
            
        syncFinished = true

    }else{
        content.CurrentFrameIndex = room.currentFrameIndex-1
        room.SendInitRoomDataToAgent(player,&content,play_id)
    }
    return syncFinished,content.CurrentFrameIndex
}

func (room *Room)GetCreateAction(play_id int,reliveFrameIndex int)msg.PlayerRelive{
     if play_id == datastruct.NULLID{
        panic("GetCreateAction play_id error")
     }
     randomIndex:=tools.GetRandomQuadrantIndex()
     point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
     action:=msg.GetCreatePlayerAction(play_id,point.X,point.Y,reliveFrameIndex)
     var actionData PlayerActionData
     actionData.ActionType = action.Action.Action
     actionData.Data = action
     room.playersData.Set(play_id,actionData)//添加action 到 lastFrameIndex+1
     return action
}

func (room *Room)SendInitRoomDataToAgent(player *datastruct.Player,content *msg.SC_InitRoomDataContent,play_id int){
     player.Agent.WriteMsg(msg.GetInitRoomDataMsg(*content))
     agentData:=player.Agent.UserData().(datastruct.AgentUserData)
     connUUID:=agentData.ConnUUID
     uid:=agentData.Uid
     rid:=room.unlockedData.RoomId
     mode:=agentData.GameMode
     player.GameData.PlayId = play_id
     tools.ReSetAgentUserData(player.Agent,connUUID,uid,rid,mode,play_id)
}

func (room *Room)syncData(connUUID string,player *datastruct.Player){
     
    room.history.Mutex.RLock()
     copyData:=make([]*msg.SC_RoomFrameDataContent,len(room.history.FramesData))
     copy(copyData,room.history.FramesData)
     room.history.Mutex.RUnlock()

     num:=len(copyData)
     
     for _,data := range copyData{
        player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(data))
     }
     lastFrameIndex:=copyData[num-1].FramesData[0].FrameIndex
 
     var length int
     room.Mutex.Lock()
     if lastFrameIndex+1 == room.currentFrameIndex {//数据帧是从0开始，服务器计算帧是从1开始
        room.onlineSyncPlayers=append(room.onlineSyncPlayers,*player)
        
        room.GetCreateAction(player.GameData.PlayId,msg.DefaultReliveFrameIndex)
        length=len(room.players)
        room.Mutex.Unlock()
        room.updateRobotRelive(length)
     }else{
        room.Mutex.Unlock()
        ok:=true
        for ok {
            if _, ok = <-room.unlockedData.startSync; ok {
                room.history.Mutex.RLock()
                copyData:=room.history.FramesData[lastFrameIndex+1:]
                room.history.Mutex.RUnlock()
                num:=len(copyData)
                for _,data := range copyData{
                   player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(data))
                }
                lastFrameIndex:=copyData[num-1].FramesData[0].FrameIndex
                isSyncFinished:=false
                room.Mutex.Lock()
                if lastFrameIndex+1 == room.currentFrameIndex {
                    isSyncFinished = true
                    room.onlineSyncPlayers=append(room.onlineSyncPlayers,*player)
                   
                    room.GetCreateAction(player.GameData.PlayId,msg.DefaultReliveFrameIndex)
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
        time.AfterFunc(MaxPlayingTime,func(){
            
            content:=new(msg.SC_GameOverDataContent)
            content.RoomId = room.unlockedData.RoomId
            msg:=msg.GetGameOverMsg(content)
            
            room.Mutex.RLock()
            for _,player := range room.onlineSyncPlayers{
                player.Agent.WriteMsg(msg)
            }
            room.Mutex.RUnlock()

            room.removeFromRooms()

            log.Debug("----------Game Over----------")
        })
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
 onlinePlayers:=room.unlockedData.parentMatch.onlinePlayers
 offlinePlayersUUID:=make([]string,0,p_num)
 expended_onlineConnUUID:=room.getEnergyExpendedConnUUID()
 for _,connUUID := range room.players{
    tf:=onlinePlayers.IsExist(connUUID)
    if !tf{
        offlinePlayersUUID =append(offlinePlayersUUID,connUUID)
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
                died:=action.(PlayerDied)
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
         action_type,action:=room.playersData.GetValue(player.GameData.PlayId,currentFrameIndex,room)
         if action != nil{
             if action_type==msg.Death{
                died:=action.(PlayerDied)
                action=died.Action
                frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,died.Points...)
             }
             frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
         }
     }
     
     for _,player := range offline_sync{
        action_type,action:=room.playersData.GetOfflineAction(player.GameData.PlayId,currentFrameIndex,room)
        if action != nil{
            if action_type==msg.Death{
               died:=action.(PlayerDied)
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
     
     log.Debug("Compute FramesData:%v,",frame_content.FramesData)
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


func (room *Room)createRoomUnlockedData(connUUIDs []string,r_id string,parentMatch *SingleMatch,rebotsNum int){
    unlockedData:=new(RoomUnlockedData)
    unlockedData.points_ch = make(chan []msg.EnergyPoint,2)
    unlockedData.rebotMoveAction = make(chan msg.Point,LeastPeople-1+MaxPeopleInRoom-1)
    unlockedData.startSync = make(chan struct{},MaxPeopleInRoom-1)
    unlockedData.pointData = room.createEnergyPointData(room.gameMap.width,room.gameMap.height)
    unlockedData.AllowList = connUUIDs
    unlockedData.RoomId = r_id
    unlockedData.isExistTicker = false
    unlockedData.parentMatch = parentMatch
    unlockedData.rebotsNum = rebotsNum
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
    for i:=0;i<num;i++{
        robot:=tools.CreateRobot(i,isRelive,room.unlockedData.pointData.quadrant,msg.DefaultReliveFrameIndex)
        robots.robots[robot.Id]=robot
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
    diedData.MaxCleanTime = 2*time.Second//容器间隔清空时间
    diedData.Mutex = new(sync.Mutex)
    diedData.Data = make([]PlayersDiedDataForTime,0,2*(1000/time_interval))
    room.diedData=diedData
}

func (diedData *PlayersDiedData)Add(values []msg.PlayerDiedData,room *Room){
    now_t := time.Now()
    diedData.Mutex.Lock()
    if len(diedData.Data)>0{
        earliest:=diedData.Data[0].DiedTime
        rs_sub:=now_t.Sub(earliest)
        if rs_sub>=diedData.MaxCleanTime{
            diedData.Data=diedData.Data[:0]
            diedData.Append(values,now_t)   
        }else{
            removeIndexs:=make([]int,0,len(values))
            for index,v := range values{
                current_pid:=v.PlayerId
                isRemove:=diedData.isRemovePlayerId(current_pid)
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
            diedData.Append(values,now_t)
        }
     }else{
        diedData.Append(values,now_t)
     }
     diedData.Mutex.Unlock()
     
     
     for _,v := range values{
         p_id:=v.PlayerId
        
         points:=v.Points

         arr:=make([]msg.EnergyPoint,0,len(points))

         for _,point := range points{
            x:= point.X
            y:= point.Y
            new_x:=int(x)
            new_y:=int(y)
            var e_point msg.EnergyPoint
            e_point.Type = int(msg.TypeC)
            e_point.X = new_x
            e_point.Y = new_y
            arr = append(arr,e_point)
         }
         
         var action msg.PlayerIsDied
         action.Action = msg.Death
         action.PlayerId = p_id

         var p_died PlayerDied
         p_died.Points = arr
         p_died.Action = action
   
        
         if p_id <= room.unlockedData.rebotsNum{
            room.robots.Mutex.RLock()
            room.robots.robots[p_id].Action = p_died
            room.robots.Mutex.RUnlock()
         }else{
            var action_data PlayerActionData 
            action_data.Data = p_died
            action_data.ActionType = msg.Death
            room.playersData.Set(p_id,action_data)
         }
     }
}


func (diedData *PlayersDiedData)isRemovePlayerId(current_pid int) bool{
     for _,dataForTime := range diedData.Data{
        for _,p_id := range dataForTime.Players{
              if current_pid == p_id{
                  return true
              }
        }
     }
     return false
}

func (diedData *PlayersDiedData)Append(values []msg.PlayerDiedData,now_t time.Time){
    var dataForTime PlayersDiedDataForTime
    dataForTime.DiedTime = now_t
    dataForTime.Players = make([]int,0,len(values))
    for _,v := range values{
         p_id:=v.PlayerId
         dataForTime.Players = append(dataForTime.Players,p_id)
    }
    diedData.Data=append(diedData.Data,dataForTime)
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
		Data:   make(map[int]PlayerActionData),//key is play_id
	}
}

func (data *PlayersFrameData)Set(k int,v PlayerActionData){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
	data.Data[k] = v
}

func (data *PlayersFrameData)CheckValue(k int) (PlayerActionData,bool){
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

func (data *PlayersFrameData)GetValue(pid int,currentFrameIndex int,room *Room)(msg.ActionType,interface{}){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
    var v interface{}
    actionData, ok := data.Data[pid]
    if ok{
        switch actionData.ActionType {
          case msg.Create:
               p_relive:=actionData.Data.(msg.PlayerRelive)
               if p_relive.ReLiveFrameIndex == msg.DefaultReliveFrameIndex || p_relive.ReLiveFrameIndex == currentFrameIndex{
                  v = p_relive.Action
                  delete(data.Data, pid)
               }else{
                  v = nil
               }
          case msg.Death:
             v=actionData.Data
             randomIndex:=tools.GetRandomQuadrantIndex()
             point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
             action:=msg.GetCreatePlayerAction(pid,point.X,point.Y,offsetFrames+currentFrameIndex)
             var actionData PlayerActionData
             actionData.ActionType = action.Action.Action
             actionData.Data = action
             data.Data[pid] = actionData
          case msg.Move:
             v=actionData.Data          
        }
        
    }else{
      action:=msg.GetCreatePlayerMoved(pid,msg.DefaultDirection.X,msg.DefaultDirection.Y,msg.DefaultSpeed)
      var actionData PlayerActionData
      actionData.ActionType = action.Action
      actionData.Data = action
      data.Data[pid] = actionData
      v=action
    }
    return actionData.ActionType,v
}

func (data *PlayersFrameData)GetOfflineAction(pid int,currentFrameIndex int,room *Room)(msg.ActionType,interface{}){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
    var v interface{}
    actionData, ok := data.Data[pid]
    if ok{
        switch actionData.ActionType {
          case msg.Create:
               p_relive:=actionData.Data.(msg.PlayerRelive)
               if p_relive.ReLiveFrameIndex == msg.DefaultReliveFrameIndex || p_relive.ReLiveFrameIndex == currentFrameIndex{
                  v = p_relive.Action
                  point:=room.getMovePoint()
                  action:=msg.GetCreatePlayerMoved(pid,point.X,point.Y,msg.DefaultSpeed)
                  var actionData PlayerActionData
                  actionData.ActionType = action.Action
                  actionData.Data = action
                  data.Data[pid]=actionData
               }else{
                  v = nil
               }
          case msg.Death:
             v=actionData.Data
             randomIndex:=tools.GetRandomQuadrantIndex()
             point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
             action:=msg.GetCreatePlayerAction(pid,point.X,point.Y,offsetFrames+currentFrameIndex)
             var actionData PlayerActionData
             actionData.ActionType = action.Action.Action
             actionData.Data = action
             data.Data[pid] = actionData
          case msg.Move:
            
            switch actionData.Data.(type){
            case msg.PlayerMoved:
                 v=actionData.Data
                 move_action:=actionData.Data.(msg.PlayerMoved)
                 offlinemove_action:=tools.CreateOfflinePlayerMoved(currentFrameIndex,&move_action)
                 var actionData PlayerActionData
                 actionData.ActionType = offlinemove_action.Action.Action
                 actionData.Data = *offlinemove_action
                 data.Data[pid] = actionData
                 
            case msg.OfflinePlayerMoved:
                 
                 offlinemove_action:=actionData.Data.(msg.OfflinePlayerMoved)
                 startIndex:=offlinemove_action.StartFrameIndex
                 directionInterval:=offlinemove_action.DirectionInterval
                 speedInterval:=offlinemove_action.SpeedInterval

                 var ptr_action *msg.PlayerMoved
                 
                 if (currentFrameIndex-startIndex)*time_interval % (directionInterval*1000) == 0 {
                     lastSpeed:=offlinemove_action.Action.Speed
                     point:=room.getMovePoint()
                     action:=msg.GetCreatePlayerMoved(pid,point.X,point.Y,lastSpeed)
                     ptr_action = &action
                 }else{
                     action:=offlinemove_action.Action
                     ptr_action = &action
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
                var actionData PlayerActionData
                actionData.ActionType = ptr_action.Action
                actionData.Data = offlinemove_action
                data.Data[pid]=actionData
                v = *ptr_action
            }
        }
        
    }else{
      action:=msg.GetCreatePlayerMoved(pid,msg.DefaultDirection.X,msg.DefaultDirection.Y,msg.DefaultSpeed)
      var actionData PlayerActionData
      actionData.ActionType = action.Action
      actionData.Data = action
      data.Data[pid] = actionData
      v=action
    }
    return actionData.ActionType,v
}





func (room *Room)getRobotAction(robot *datastruct.Robot,currentFrameIndex int)(msg.ActionType,interface{}){
     var rs interface{}
     current_action:=robot.Action
     var rs_type msg.ActionType
     switch current_action.(type){
       case msg.PlayerRelive:
            p_relive:=current_action.(msg.PlayerRelive)
            rs_type = p_relive.Action.Action
            if p_relive.ReLiveFrameIndex == msg.DefaultReliveFrameIndex || p_relive.ReLiveFrameIndex == currentFrameIndex{
                 rs = p_relive.Action
                 point:=room.getMovePoint()
                 action:=msg.GetCreatePlayerMoved(robot.Id,point.X,point.Y,msg.DefaultSpeed)
                 robot.Action = action
            }else{
                rs = nil
            }

       case msg.PlayerMoved:
            var ptr_action *msg.PlayerMoved
            
            if currentFrameIndex*time_interval % (robot.DirectionInterval*1000) == 0 {
                lastSpeed:=current_action.(msg.PlayerMoved).Speed
                point:=room.getMovePoint()
                action:=msg.GetCreatePlayerMoved(robot.Id,point.X,point.Y,lastSpeed)
                ptr_action = &action
                
            }else{
                action:=current_action.(msg.PlayerMoved)
                ptr_action = &action
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
            
            robot.Action=*ptr_action
            rs=robot.Action
            rs_type = msg.Move
            
        case PlayerDied:
            rs = current_action //PlayerDied
            rs_type = msg.Death
            if robot.IsRelive {
                action:=tools.GetCreateRobotAction(robot.Id,room.unlockedData.pointData.quadrant,offsetFrames+currentFrameIndex)
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
      if playersNum > 1&&playersNum<=LeastPeople{
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
      
      action:=msg.GetCreatePlayerMoved(PlayerId,moveData.MsgContent.X,moveData.MsgContent.Y,moveData.MsgContent.Speed)
      var actionData PlayerActionData
      actionData.ActionType = action.Action
      
      room.playersData.Mutex.Lock()
      defer room.playersData.Mutex.Unlock()
      if action.X == 0 && action.Y == 0{
          last_actionData:=room.playersData.Data[PlayerId]
          switch last_actionData.Data.(type){
                 case msg.PlayerMoved:
                      last_action:=last_actionData.Data.(msg.PlayerMoved)
                      action.X = last_action.X
                      action.Y = last_action.Y
                 default:
                    action.X = msg.DefaultDirection.X
                    action.Y = msg.DefaultDirection.Y
          }
         
      }
      actionData.Data = action
      room.playersData.Data[PlayerId] = actionData
}