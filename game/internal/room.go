package internal

import (
	"fmt"
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
const time_interval = 50 //50毫秒
var map_factor = 5*40

const MaxPeopleInRoom = 20 //每个房间最大人数
const RoomCloseTime = 15.0*time.Second//玩家最大等待时间多少秒

const FirstFrameIndex = 0//第一帧索引

type Room struct {
    Mutex *sync.RWMutex //读写互斥量
    IsOn bool //玩家是否能进入的开关

    gameMap *GameMap //游戏地图
    

    roomData *RoomData
    unlockedData *RoomUnlockedData

    //robots *RobotData
}

type RoomUnlockedData struct {
    isExistTicker bool
    ticker *time.Ticker
     points_ch chan []msg.Point
     pointData *EnergyPointData
     AllowList []string//允许列表
     RoomType RoomDataType//房间类型
     RoomId string
}

type RobotData struct {
    Mutex *sync.RWMutex //读写互斥量
    robots map[string]datastruct.Robot//机器人列表map[string]
    //robotsData *tools.SafeMap//机器人数据map[string]*RebotFramesData
}

type RoomData struct {
    Mutex *sync.RWMutex //读写互斥量
    currentFrameIndex int//记录当前第几帧
    players []string//玩家列表

    //history
    /*
    playersData+robotsData == history
    playersData *tools.SafeMap//玩家数据map[string]*PlayerFramesData
    robotsData *tools.SafeMap//机器人数据map[string]*RebotFramesData
    */
}

type PlayerFramesData struct {
    Mutex *sync.RWMutex 
    FramesData []interface{}//比如存玩家第1帧的动作事件,eventdata
}

type RebotFramesData struct {
    Mutex *sync.RWMutex 
    FramesData []interface{} //eventdata
}

type GameMap struct{
    height int
    width int
}

type EnergyPointData struct{
     quadrant []msg.Quadrant
     firstFramePoint []msg.Point //第一帧的能量点数据
}


/*以下为玩家事件*/
type CreatePlayer struct {//玩家的创建
     point msg.Point    
}

type PlayerIsDied struct {//玩家的死亡
     point msg.Point
}

type PlayerMoved struct {//玩家的移动
     point msg.Point
     //方向
}


func createRoom(connUUIDs []string,r_type RoomDataType,r_id string)*Room{
    room := new(Room)
    room.Mutex = new(sync.RWMutex)
    room.createGameMap(map_factor)
    room.createRoomData()
    room.createRoomUnlockedData(connUUIDs,r_type,r_id)
    room.IsOn = true
    
    
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
            room.Mutex.Unlock()
            room.roomData.Mutex.RLock()
            length:= len(room.roomData.players)
            if length <=0{
                isRemove = true
            }
            room.roomData.Mutex.RUnlock()
            if isRemove{
                room.stopTicker()
                rooms.Delete(room.unlockedData.RoomId)
            }
        })
       case Invite:
        log.Debug("create Invite Room")
    }
    return room
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
    // p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,2))
    // p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,3))
    // p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(width,height,4))
    //p_data.firstFramePoint=getPoints(num,msg.TypeB,p_data.quadrant)//第一帧生成能量点
    tools.TestPoint()
    p_data.firstFramePoint= tools.TestRandomPoint(msg.TypeB)
    
    go room.goCreatePoints(num,msg.TypeB)
    return p_data
}

func getPoints(num int,maxRangeType int,quadrant []msg.Quadrant) []msg.Point{
    // rs_slice:=make([]msg.Point,0,len(quadrant)*num)
    // for _,v := range quadrant{
    //     tmp:=tools.GetRandomPoint(v,num,maxRangeType)
    //     rs_slice=append(rs_slice,tmp...)
    // }
    // return rs_slice

    return tools.TestRandomPoint(msg.TypeB)
}


func(room *Room)Join(connUUID string,a gate.Agent){

    var content msg.SC_InitRoomDataContent
    content.MapHeight = room.gameMap.height
    content.MapWidth = room.gameMap.width
    content.Interval = time_interval
    
    var frame_content msg.SC_RoomFrameDataContent
    frame_content.FramesData=make([]msg.FrameData,0,4)
    var frame_data msg.FrameData

    room.roomData.Mutex.Lock()
    content.CurrentFrameIndex = room.roomData.currentFrameIndex
    length:=len(room.roomData.players)
    if length == MaxPeopleInRoom - 1 {
       room.IsOn = false
    }
    room.roomData.players=append(room.roomData.players,connUUID)
    
    log.Debug("Join GetInitRoomDataMsg")
    a.WriteMsg(msg.GetInitRoomDataMsg(content))

    
    //存入容器,定时发送

    //player actions and points
    if content.CurrentFrameIndex == FirstFrameIndex{
        room.createTicker()
        frame_data.FrameIndex = FirstFrameIndex
        frame_data.CreateEnergyPoints = room.unlockedData.pointData.firstFramePoint
        frame_content.FramesData=append(frame_content.FramesData,frame_data)
        log.Debug("Join GetRoomFrameDataMsg")
        a.WriteMsg(msg.GetRoomFrameDataMsg(&frame_content))
        
    }else{


    }

    room.roomData.Mutex.Unlock()
}

func (room*Room)goCreatePoints(num int,maxRangeType int){
     for{
         room.unlockedData.points_ch <- getPoints(num,msg.TypeB,room.unlockedData.pointData.quadrant)
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

func (room *Room)ComputeFrameData(){
     var currentFrameIndex int
     room.roomData.Mutex.Lock()
     room.roomData.currentFrameIndex++
     currentFrameIndex = room.roomData.currentFrameIndex
     room.roomData.Mutex.Unlock()

     var frame_content msg.SC_RoomFrameDataContent
     frame_content.FramesData=make([]msg.FrameData,0,1)
     var frame_data msg.FrameData
     frame_data.FrameIndex = currentFrameIndex

     var points []msg.Point
     select {
     case points = <-room.unlockedData.points_ch:
     default:
      points=nil
    }

     if points != nil&&len(points)>0{
        frame_data.CreateEnergyPoints = points 
     }
     frame_content.FramesData=append(frame_content.FramesData,frame_data)
     


     room.roomData.Mutex.Lock()
     p_num:=len(room.roomData.players)
     onlinePlayersInRoom:=make([]datastruct.Player,0,p_num)
     offlinePlayersInRoom:=make([]int,0,p_num)
     for index,uuid := range room.roomData.players{
        player,tf:=onlinePlayers.Get(uuid)
        if tf{
            onlinePlayersInRoom=append(onlinePlayersInRoom,player)
        }else {
            offlinePlayersInRoom=append(offlinePlayersInRoom,index)
        }
     }
     removeOfflinePlayersInRoom(room,offlinePlayersInRoom)//remove offline players
     room.roomData.Mutex.Unlock()
     
     for _,player := range onlinePlayersInRoom{
         fmt.Println(frame_content)
         player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(&frame_content))
     }
    

     //save SC_RoomFrameDataContent with FrameIndex

}

func removeOfflinePlayersInRoom(room *Room,removeIndex []int){
    rm_count:=0
    for index,v := range removeIndex {
        if index!=0{
           v = v-rm_count
        }
        room.roomData.players=append(room.roomData.players[:v], room.roomData.players[v+1:]...)
        rm_count++;
    }
}

func (room *Room)createRoomData(){
    roomData:=new(RoomData)
    roomData.Mutex = new(sync.RWMutex)
    roomData.currentFrameIndex = FirstFrameIndex
    roomData.players = make([]string,0,MaxPeopleInRoom)
    room.roomData = roomData
}

func (room *Room)createRoomUnlockedData(connUUIDs []string,r_type RoomDataType,r_id string){
    unlockedData:=new(RoomUnlockedData)
    unlockedData.points_ch = make(chan []msg.Point,2)
    unlockedData.pointData = room.createEnergyPointData(room.gameMap.width,room.gameMap.height)
    unlockedData.AllowList = connUUIDs
    unlockedData.RoomId = r_id
    unlockedData.RoomType = r_type
    unlockedData.isExistTicker = false
    room.unlockedData = unlockedData
}