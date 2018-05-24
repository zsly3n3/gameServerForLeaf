package internal

import (
	//"fmt"
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
const time_interval = 50*200 //50毫秒
var map_factor = 5*40

const MaxPeopleInRoom = 20 //每个房间最大人数
const RoomCloseTime = 15.0*time.Second//玩家最大等待时间多少秒

const FirstFrameIndex = 0//第一帧索引

const MaxPlayingTime = 5*time.Minute

type Room struct {
    Mutex *sync.RWMutex //读写互斥量
    IsOn bool //玩家是否能进入的开关
    players []string//玩家列表
  
    currentFrameIndex int//记录当前第几帧
    onlineSyncPlayers []datastruct.Player//同步完成的在线玩家列表,第0帧进来的玩家就已存在同步列表中
    playersData *PlayersFramesData//玩家数据

    gameMap *GameMap //游戏地图
    
    unlockedData *RoomUnlockedData
    history *HistoryFrameData
    
    //robots *RobotData
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
    
}


type PlayersFramesData struct {
     Mutex *sync.RWMutex //读写互斥量
     Data map[string]*PlayerFramesData
}

type PlayerFramesData struct {
     Mutex *sync.RWMutex //读写互斥量
     //SaveLastNum int //保存最后一次的个数,看是否有更新
     Data []interface{}//目前只能存一个动作,之后可能改进为每个单位存一组动作
}

type RobotData struct {
    Mutex *sync.RWMutex //读写互斥量
    robots map[string]datastruct.Robot//机器人列表map[string]
    //robotsData *tools.SafeMap//机器人数据map[string]*RebotFramesData
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

type GameMap struct{
    height int
    width int
}

type EnergyPointData struct{
     quadrant []msg.Quadrant
     firstFramePoint []msg.EnergyPoint //第一帧的能量点数据
}



func createRoom(connUUIDs []string,r_type RoomDataType,r_id string)*Room{
    room := new(Room)
    room.Mutex = new(sync.RWMutex)
    room.createGameMap(map_factor)
    room.createHistoryFrameData()
    room.currentFrameIndex = FirstFrameIndex
    room.onlineSyncPlayers = make([]datastruct.Player,0,MaxPeopleInRoom)
    room.createRoomUnlockedData(connUUIDs,r_type,r_id)
    room.IsOn = true
    room.players = make([]string,0,MaxPeopleInRoom)
    room.playersData = NewPlayersFramesData()
    switch r_type{
       case Matching:
        log.Debug("create Matching Room")
        // room.robots = tools.NewSafeMap()
        // room.robotsData = tools.NewSafeMap()
        //create robots
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
    num:=1
    width=width-diff
    height=height-diff
    
    p_data.quadrant = make([]msg.Quadrant,0,4)
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,1))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,2))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,3))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,4))
    p_data.firstFramePoint=getPoints(num,msg.TypeB,p_data.quadrant)//第一帧生成能量点
  
    go room.goCreatePoints(num,msg.TypeB)
    return p_data
}

func getPoints(num int,maxRangeType int,quadrant []msg.Quadrant) []msg.EnergyPoint{
    rs_slice:=make([]msg.EnergyPoint,0,len(quadrant)*num)
    for _,v := range quadrant{
        tmp:=tools.GetRandomPoint(v,num,maxRangeType)
        rs_slice=append(rs_slice,tmp...)
    }
    return rs_slice
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

    var frame_content msg.SC_RoomFrameDataContent

   
    var frame_data msg.FrameData
    content.CurrentFrameIndex = room.currentFrameIndex
    syncFinished:=false

    if content.CurrentFrameIndex == FirstFrameIndex{
        room.SendInitRoomDataToAgent(player.Agent,&content)
        room.onlineSyncPlayers=append(room.onlineSyncPlayers,player)
        frame_data.FrameIndex = FirstFrameIndex
        frame_data.CreateEnergyPoints = room.unlockedData.pointData.firstFramePoint
        frame_content.FramesData = make([]msg.FrameData,0,1)
        
        frame_data.PlayerFrameData=make([]interface{},0,1)
        randomIndex:=tools.GetRandomQuadrantIndex()
        point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
        action:=msg.GetCreatePlayerAction(player.Id,point.X,point.Y)
        frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
        
        frame_content.FramesData = append(frame_content.FramesData,frame_data)
        
        
        
        //创建第一帧的动作
        p_FramesData:=NewPlayerFramesData()
        p_FramesData.Set(action)//添加action 到 lastFrameIndex+1
        room.playersData.Set(connUUID,p_FramesData)
        
        syncFinished = true
    }else{
        room.SendInitRoomDataToAgent(player.Agent,&content)
    }
    return syncFinished,content.CurrentFrameIndex
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

     randomIndex:=tools.GetRandomQuadrantIndex()
     point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex)
     action:=msg.GetCreatePlayerAction(player.Id,point.X,point.Y)
     p_FramesData:=NewPlayerFramesData()
     
     
     room.Mutex.Lock()
     if lastFrameIndex == room.currentFrameIndex { 
        room.onlineSyncPlayers=append(room.onlineSyncPlayers,player)
        p_FramesData.Set(action)//添加action 到 lastFrameIndex+1
        room.playersData.Set(connUUID,p_FramesData)
        log.Debug("Normal SyncFinished")
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
                if lastFrameIndex == room.currentFrameIndex {
                    isSyncFinished = true
                    room.onlineSyncPlayers=append(room.onlineSyncPlayers,player)
                    p_FramesData.Set(action)//添加action 到 lastFrameIndex+1
                    room.playersData.Set(connUUID,p_FramesData)
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

func (room*Room)goCreatePoints(num int,maxRangeType int){
     for {
        isClosed := safeSendPoint(room.unlockedData.points_ch,getPoints(num,msg.TypeB,room.unlockedData.pointData.quadrant))
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
            
            room.currentFrameIndex++
            currentFrameIndex = room.currentFrameIndex //服务器帧是从1开始
            currentFrameIndex--
            
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

    var points []msg.EnergyPoint
    select {
     case points = <-room.unlockedData.points_ch:
     default:
      points=nil
    }
    
     if points != nil&&len(points)>0{
        frame_data.CreateEnergyPoints = points 
     }

     
    
     
     frame_data.PlayerFrameData=make([]interface{},0,len(online_sync)+len(offline_sync))
     for _,player := range online_sync{
         connUUID:=player.Agent.UserData().(datastruct.AgentUserData).ConnUUID
         p_FramesData:=room.playersData.Get(connUUID)
         frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,p_FramesData.Get(player.Id))
     }
     
     // for _,player := range offline_sync{
        
     // }
    
     
     frame_content.FramesData = append(frame_content.FramesData,frame_data)
     
     
     
     for _,player := range online_sync{
         player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(&frame_content))
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

func NewPlayersFramesData() *PlayersFramesData {
	return &PlayersFramesData{
		Mutex: new(sync.RWMutex),
		Data:   make(map[string]*PlayerFramesData),
	}
}

func NewPlayerFramesData() *PlayerFramesData {
	return &PlayerFramesData{
        Mutex: new(sync.RWMutex),
		Data:   make([]interface{},0,10),
	}
}


func (data *PlayersFramesData)Set(k string,v *PlayerFramesData){
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
	if _, ok := data.Data[k]; !ok {
		data.Data[k] = v
	}
}


func (data *PlayersFramesData)Get(k string) *PlayerFramesData{
    data.Mutex.RLock()
	defer data.Mutex.RUnlock()
	if val, ok := data.Data[k]; ok {
		return val
    }
    return nil
}


func (data *PlayerFramesData)Set(v interface{}){
    data.Mutex.Lock()
    defer data.Mutex.Unlock()
    data.Data=append(data.Data,v)
}

func (data *PlayerFramesData)Get(pid int)interface{}{
    data.Mutex.Lock()
	defer data.Mutex.Unlock()
    num:=len(data.Data)
    var v interface{}
    if num > 0{
        v=data.Data[num-1]
        data.Data=data.Data[:0] //clean
    }else {
        v=msg.GetCreatePlayerMoved(pid,msg.DefaultDirection.X,msg.DefaultDirection.Y,msg.DefaultSpeed) 
    }
    return v
}