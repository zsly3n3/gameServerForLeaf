package internal

import (
	"server/msg"
	"server/datastruct"
	"sync"
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/gate" 
    "time"
    "server/tools"
)

type RoomDataType int //房间类型,匹配类型还是邀请类型
const (
	Matching RoomDataType = iota
	Invite
)


const min_MapWidth = 20
const min_MapHeight = 15
const time_interval = 50//50毫秒
var map_factor = 5*40



const MaxPeopleInRoom = 20 //每个房间最大人数
const RoomCloseTime = 15.0*time.Second//玩家最大等待时间多少秒

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

type Room struct {
    Mutex *sync.RWMutex //读写互斥量
    IsOn bool //玩家是否能进入的开关
    players []string//玩家列表
    
    currentFrameIndex int//记录当前第几帧
    onlineSyncPlayers []datastruct.Player//同步完成的在线玩家列表,第0帧进来的玩家就已存在同步列表中
    playersData *PlayersFrameData//玩家数据

    gameMap *GameMap //游戏地图
    
    unlockedData *RoomUnlockedData
    history *HistoryFrameData
    
    energyData *EnergyPowerData

    //interface{} 历史帧能量消耗事件数据，保存消耗后的能量点数据。 用于计算生产能量数据
    robots *RobotData
}

type HistoryFrameData struct {
    Mutex *sync.RWMutex //读写互斥量
    FramesData []*msg.SC_RoomFrameDataContent
}

type RoomUnlockedData struct {
     isExistTicker bool
     ticker *time.Ticker
     points_ch chan []msg.EnergyPoint
     pointData *EnergyPointData
     AllowList []string//允许列表
     RoomType RoomDataType//房间类型
     RoomId string
     startSync chan struct{} //开始同步的管道
     rebotMoveAction chan msg.Point
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
     Mutex *sync.Mutex //读写互斥量
     EnableCreateEnergyPower int //当前可以生成的能量
}

type PlayersFrameData struct {
     Mutex *sync.RWMutex //读写互斥量
     Data map[string] PlayerActionData
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

// type RoomData struct {
//     Mutex *sync.RWMutex //读写互斥量
   
   
 
//     //history
//     /*
//     playersData+robotsData == history
//     playersData *tools.SafeMap//玩家数据map[string]*PlayerFramesData
//     robotsData *tools.SafeMap//机器人数据map[string]*RebotFramesData
//     */
// }



// type RebotFramesData struct {
//     Mutex *sync.RWMutex 
//     FramesData []interface{} //eventdata
// }





func createRoom(connUUIDs []string,r_type RoomDataType,r_id string)*Room{
    room := new(Room)
    room.Mutex = new(sync.RWMutex)
    room.createGameMap(map_factor)
    room.createRoomUnlockedData(connUUIDs,r_type,r_id)
    room.createHistoryFrameData()
    room.createEnergyPowerData()
    
    room.currentFrameIndex = FirstFrameIndex
    room.onlineSyncPlayers = make([]datastruct.Player,0,MaxPeopleInRoom)
    
    room.IsOn = true
    room.players = make([]string,0,MaxPeopleInRoom)
    room.playersData = NewPlayersFrameData()
    switch r_type{
       case Matching:
        log.Debug("create Matching Room")
        room.createRobotData(LeastPeople-len(connUUIDs),true)
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
       case Invite:
        log.Debug("create Invite Room")
    }
    return room
}

func (room *Room)removeFromRooms(){
     room.stopTicker()
     safeCloseRobotMoved(room.unlockedData.rebotMoveAction)
     safeClosePoint(room.unlockedData.points_ch)
     safeCloseSync(room.unlockedData.startSync)
     rooms.Delete(room.unlockedData.RoomId)
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
    
    
    go room.goCreatePoints(1,msg.TypeB)
    go room.goCreateMovePoint()
    return p_data
}



func(room *Room)IsSyncFinished(connUUID string,player datastruct.Player) (bool,int){
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

    if room.currentFrameIndex == FirstFrameIndex{
       
        content.CurrentFrameIndex = FirstFrameIndex

        room.SendInitRoomDataToAgent(player.Agent,&content)
        room.onlineSyncPlayers=append(room.onlineSyncPlayers,player)
        
        var frame_data msg.FrameData
        var frame_content msg.SC_RoomFrameDataContent

        frame_data.FrameIndex = FirstFrameIndex
        frame_data.CreateEnergyPoints = room.unlockedData.pointData.firstFramePoint
        frame_content.FramesData = make([]msg.FrameData,0,1)
        
        frame_data.PlayerFrameData=make([]interface{},0,1)

        
        action:=room.GetCreateAction(connUUID,player.Id)
        frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
        
        frame_content.FramesData = append(frame_content.FramesData,frame_data)
            
        
        syncFinished = true
    }else{
        
        content.CurrentFrameIndex = room.currentFrameIndex-1
        room.SendInitRoomDataToAgent(player.Agent,&content)
    }
    return syncFinished,content.CurrentFrameIndex
}

func (room *Room)GetCreateAction(connUUID string,p_id int)msg.CreatePlayer{
     randomIndex:=tools.GetRandomQuadrantIndex()
     point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
     action:=msg.GetCreatePlayerAction(p_id,point.X,point.Y)
     var actionData PlayerActionData
     actionData.ActionType = action.Action
     actionData.Data = action
     room.playersData.Set(connUUID,actionData)//添加action 到 lastFrameIndex+1
     return action
}


 
func (room *Room)SendInitRoomDataToAgent(a gate.Agent,content *msg.SC_InitRoomDataContent){
     a.WriteMsg(msg.GetInitRoomDataMsg(*content))
     agentData:=a.UserData().(datastruct.AgentUserData)
     tools.UpdateAgentUserData(a,agentData.ConnUUID,agentData.Uid,room.unlockedData.RoomId)
}

func (room *Room)syncData(connUUID string,player datastruct.Player){
     room.history.Mutex.RLock()
     copyData:=make([]*msg.SC_RoomFrameDataContent,len(room.history.FramesData))
     copy(copyData,room.history.FramesData)
    
     room.history.Mutex.RUnlock()
     num:=len(copyData)
     
     for _,data := range copyData{
        player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(data))
     }
     lastFrameIndex:=copyData[num-1].FramesData[0].FrameIndex
 
     room.Mutex.Lock()
     
     if lastFrameIndex+1 == room.currentFrameIndex {//数据帧是从0开始，服务器计算帧是从1开始
        room.onlineSyncPlayers=append(room.onlineSyncPlayers,player)
        room.GetCreateAction(connUUID,player.Id)
        room.Mutex.Unlock()
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
                    room.onlineSyncPlayers=append(room.onlineSyncPlayers,player)
                    room.GetCreateAction(connUUID,player.Id)
                }
                room.Mutex.Unlock()
                if isSyncFinished{
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
       syncFinished,currentFrameIndex=room.IsSyncFinished(connUUID,player)
       isOn = true
    }else{
       isOn=room.IsOn
       if isOn{
          syncFinished,currentFrameIndex=room.IsSyncFinished(connUUID,player)
       }
    }
    room.Mutex.Unlock()
    if currentFrameIndex==FirstFrameIndex{
       room.createTicker()
    }
    if isOn&&!syncFinished{
       go room.syncData(connUUID,player)
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
            room.removeFromRooms()
            //send over msg
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

func (room *Room)IsRemoveRoom()(bool,int,[]datastruct.Player,[]datastruct.Player,int,bool){
 //判断在线玩家
 isRemove:=false
 room.Mutex.Lock()
 defer room.Mutex.Unlock()
 p_num:=len(room.players)

 offlinePlayersUUID:=make([]string,0,p_num)
 for _,connUUID := range room.players{
    tf:=onlinePlayers.IsExist(connUUID)
    if !tf{
        offlinePlayersUUID =append(offlinePlayersUUID,connUUID)
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
    offline_sync:= make([]datastruct.Player,0,MaxPeopleInRoom)

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
              offline_sync=append(offline_sync,player) 
             }
           }
          removeOfflineSyncPlayersInRoom(room,offlineSyncPlayersIndex)//remove offline players
       }
       online_sync=make([]datastruct.Player,len(room.onlineSyncPlayers))
       copy(online_sync,room.onlineSyncPlayers)
    }

 syncNotFinishedPlayers:=onlinePlayersInRoom-len(online_sync)
 isRemoveHistory:=false
 if syncNotFinishedPlayers == 0&&!room.IsOn{
    isRemoveHistory = true
 }
 return isRemove,currentFrameIndex,online_sync,offline_sync,syncNotFinishedPlayers,isRemoveHistory
}

func (room *Room)ComputeFrameData(){
     isRemove,currentFrameIndex,online_sync,offline_sync,syncNotFinishedPlayers,isRemoveHistory:=room.IsRemoveRoom()
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
        if points != nil && len(points)>0 && room.energyData.IsCreatePower(){
            frame_data.CreateEnergyPoints = points
        }
     }

     
     room.robots.Mutex.RLock()
     for _,robot:= range room.robots.robots{
         action:=room.getRobotAction(robot,currentFrameIndex)
         frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
     }
     room.robots.Mutex.RUnlock()


     for _,player := range online_sync{
         connUUID:=player.Agent.UserData().(datastruct.AgentUserData).ConnUUID
         action:=room.playersData.GetValue(connUUID,player.Id)
         frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
     }

    //  for _,player := range offline_sync{
    //     point:=room.getMovePoint()
    //     action:=msg.GetCreatePlayerMoved(player.Id,point.X,point.Y,msg.DefaultSpeed)
    //     frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
    //  }
     
     frame_content.FramesData = append(frame_content.FramesData,frame_data)
     
     for _,player := range online_sync{
         msg:=msg.GetRoomFrameDataMsg(&frame_content)
         player.Agent.WriteMsg(msg)
         log.Debug("ComputeFrameData msgHeader:%v,msgContent:%v",msg.MsgHeader,msg.MsgContent)
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


func (room *Room)createRoomUnlockedData(connUUIDs []string,r_type RoomDataType,r_id string){
    unlockedData:=new(RoomUnlockedData)
    unlockedData.points_ch = make(chan []msg.EnergyPoint,2)
    unlockedData.rebotMoveAction = make(chan msg.Point,LeastPeople-1+MaxPeopleInRoom-1)
    unlockedData.startSync = make(chan struct{},MaxPeopleInRoom-1)
    unlockedData.pointData = room.createEnergyPointData(room.gameMap.width,room.gameMap.height)
    unlockedData.AllowList = connUUIDs
    unlockedData.RoomId = r_id
    unlockedData.RoomType = r_type
    unlockedData.isExistTicker = false
    room.unlockedData = unlockedData
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
    energyData.Mutex = new(sync.Mutex)
    energyData.EnableCreateEnergyPower = MaxEnergyPower - InitEnergyPower
    room.energyData = energyData
}
func (room *Room)createRobotData(num int,isRelive bool){
    robots:=new(RobotData)
    robots.Mutex = new(sync.RWMutex)
    robots.robots = make(map[int]*datastruct.Robot)
    for i:=0;i<num;i++{
        robot:=tools.CreateRobot(i,isRelive,room.unlockedData.pointData.quadrant)
        robots.robots[robot.Id]=robot
    }
    room.robots = robots
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
		Data:   make(map[string]PlayerActionData),
	}
}

func (data *PlayersFrameData)Set(k string,v PlayerActionData){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
	data.Data[k] = v
}

func (data *PlayersFrameData)CheckValue(k string) (PlayerActionData,bool){
    data.Mutex.RLock()
    defer data.Mutex.RUnlock()
    actionData, ok := data.Data[k]
    return actionData,ok
}

func (data *PlayersFrameData)GetValue(k string,pid int)interface{}{
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
    var v interface{}
    actionData, ok := data.Data[k]
    if ok{
        v=actionData.Data
        if actionData.ActionType == msg.Create{
            delete(data.Data, k)
        }
    }else{
      action:=msg.GetCreatePlayerMoved(pid,msg.DefaultDirection.X,msg.DefaultDirection.Y,msg.DefaultSpeed)
      var actionData PlayerActionData
      actionData.ActionType = action.Action
      actionData.Data = action
      data.Data[k] = actionData
      v=action
    }
    return v
}

func (power *EnergyPowerData)IsCreatePower()bool{
     tf:=false
     power.Mutex.Lock()
     if power.EnableCreateEnergyPower>=PerFramePower{
        power.EnableCreateEnergyPower-=PerFramePower
        tf = true
     }
     power.Mutex.Unlock()
     return tf
}

func (power *EnergyPowerData)SetPower(num int){
    power.Mutex.Lock()
    power.EnableCreateEnergyPower += num
    if power.EnableCreateEnergyPower>MaxEnergyPower{
       power.EnableCreateEnergyPower = MaxEnergyPower 
    }
    power.Mutex.Unlock()
}


func (room *Room)getRobotAction(robot *datastruct.Robot,currentFrameIndex int)interface{}{
     var rs interface{}
     current_action:=robot.Action
    
     switch current_action.(type){
       case msg.CreatePlayer:
            rs=current_action
            point:=room.getMovePoint()
            action:=msg.GetCreatePlayerMoved(robot.Id,point.X,point.Y,msg.DefaultSpeed)
            robot.Action = action
       case msg.PlayerMoved:
            var ptr_action *msg.PlayerMoved

            if currentFrameIndex*time_interval % (robot.DirectionInterval*1000) == 0 {
                lastSpeed:=current_action.(msg.PlayerMoved).Speed
                point:=room.getMovePoint()
                action:=msg.GetCreatePlayerMoved(robot.Id,point.X,point.Y,msg.DefaultSpeed)
                action.Speed=lastSpeed
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
       //case msg.PlayerDied:
              //create player
     }
	 return rs
}