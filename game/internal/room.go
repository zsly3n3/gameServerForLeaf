package internal

import (
	"server/msg"
	"server/datastruct"
	"sync"
    "github.com/name5566/leaf/log"
    "time"
    "server/tools"
)

type RoomDataType int //房间类型,匹配类型还是邀请类型
const (
	Matching RoomDataType = iota
	Invite
)

const min_MapHeight = 15
const min_MapWidth = 20  
const time_interval = 50 //50毫秒
var map_factor = 5*40

const MaxPeopleInRoom = 20 //每个房间最大人数
const RoomCloseTime = 15.0*time.Second//玩家最大等待时间多少秒

const FirstFrameIndex = 0//第一帧索引

type Room struct {

    Mutex *sync.RWMutex //读写互斥量
    IsOn bool //玩家是否能进入的开关
    RoomType RoomDataType//房间类型
    AllowList []string//允许列表
    RoomId string
    gameMap *GameMap //游戏地图
    
    players []string//玩家列表
    robots  map[string]datastruct.Robot//机器人列表map[string]

    pointData *EnergyPointData//能量点数据
    
    currentFrameIndex int//记录当前第几帧

    ticker *time.Ticker

    points_ch chan []msg.Point

    /*
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
     point map[int][]msg.Point
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
    room.points_ch = make(chan []msg.Point)
    room.currentFrameIndex = FirstFrameIndex
    room.IsOn = true
    room.ticker = nil
    go room.selectTicker()
    room.AllowList = connUUIDs
    room.RoomType = r_type
    room.Mutex = new(sync.RWMutex)
    room.RoomId = r_id
    // room.players = tools.NewSafeMap()
    // room.playersData = tools.NewSafeMap()
    room.createGameMap(map_factor)
    room.createEnergyPointData()
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
                room.stopTicker()
                rooms.Delete(room.RoomId)
            }
        })
       case Invite:
        log.Debug("create Invite Room")
    }
    return room
}


func (room *Room)createGameMap(fac int){
    g_map:=new(GameMap)
    g_map.height = min_MapHeight*fac
    g_map.width = min_MapWidth*fac
    room.gameMap = g_map
}

func (room *Room)createEnergyPointData(){
    p_data:=new(EnergyPointData)
    room.pointData = p_data
    diff:=200
    num:=2
    length:=room.gameMap.height-diff
    width:=room.gameMap.width-diff
    p_data.quadrant = make([]msg.Quadrant,0,4)
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(length,width,1))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(length,width,2))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(length,width,3))
    p_data.quadrant=append(p_data.quadrant,tools.CreateQuadrant(length,width,4))

    p_data.point = make(map[int][]msg.Point)
    p_data.point[FirstFrameIndex]=getPoints(num,msg.TypeB,room.pointData.quadrant)//第一帧生成能量点
    go room.goCreatePoints(num,msg.TypeB)
}

func getPoints(num int,maxRangeType int,quadrant []msg.Quadrant) []msg.Point{
    rs_slice:=make([]msg.Point,0,len(quadrant)*num)
    for _,v := range quadrant{
        tmp:=tools.GetRandomPoint(v,num,maxRangeType)
        rs_slice=append(rs_slice,tmp...)
    }
    return rs_slice
}


func(room *Room)Join(player *datastruct.Player){
    user_data:=player.Agent.UserData().(datastruct.AgentUserData)
    connUUID:=user_data.ConnUUID
    removeFromMatchActionPool(connUUID)
    length:=len(room.players)
   
    if length == MaxPeopleInRoom - 1 {
       room.IsOn = false
    }
    
    var content msg.SC_InitRoomDataContent
    content.MapHeight = room.gameMap.height
    content.MapWidth = room.gameMap.width
    content.CurrentFrameIndex = room.currentFrameIndex
    content.Interval = time_interval
    player.Agent.WriteMsg(msg.GetInitRoomDataMsg(content))


    if room.currentFrameIndex == FirstFrameIndex{
        //room.createTicker()
        var frame_content msg.SC_RoomFrameDataContent
        frame_content.FramesData=make([]msg.FrameData,0,4)
        var frame_data msg.FrameData
        frame_data.FrameIndex = FirstFrameIndex
        frame_data.CreateEnergyPoints = room.pointData.point[FirstFrameIndex]
        frame_content.FramesData=append(frame_content.FramesData,frame_data)
        
        player.Agent.WriteMsg(msg.GetRoomFrameDataMsg(&frame_content))
        
    }else{


    }

    room.players=append(room.players,connUUID)
}

func (room*Room)goCreatePoints(num int,maxRangeType int){
     for{
         room.points_ch <- getPoints(num,msg.TypeB,room.pointData.quadrant)
     }
}

func(room *Room)createTicker(){
	if room.ticker == nil{
        room.ticker = time.NewTicker(time_interval*time.Millisecond)
    }
}
func(room *Room) stopTicker(){
    if room.ticker != nil{
        room.ticker.Stop()
        room.ticker=nil
    }
}

func(room *Room) selectTicker(){
     for {
		 if room.ticker != nil{
			select {
			case <-room.ticker.C:
				room.ComputeFrameData()
			}
		 }
	 }
}

func (room *Room)ComputeFrameData(){
     var points []msg.Point
     room.Mutex.Lock()
     room.currentFrameIndex++
     p_num:=len(room.players)
     onlinePlayersInRoom:=make([]datastruct.Player,0,p_num)
     offlinePlayersInRoom:=make([]int,0,p_num)
     for index,uuid := range room.players{
        player,tf:=onlinePlayers.Get(uuid)
        if tf{
            onlinePlayersInRoom=append(onlinePlayersInRoom,player)
        }else {
            offlinePlayersInRoom=append(offlinePlayersInRoom,index)
        }
     }
     removeOfflinePlayersInRoom(room,offlinePlayersInRoom)//remove offline players
     select {
      case points = <-room.points_ch:
      default:
        points=nil
     }
     room.Mutex.Unlock()

     var frame_content msg.SC_RoomFrameDataContent
     frame_content.FramesData=make([]msg.FrameData,0,1)
     var frame_data msg.FrameData
     frame_data.FrameIndex = room.currentFrameIndex

     if points != nil{
        frame_data.CreateEnergyPoints = points 
     }
     frame_content.FramesData=append(frame_content.FramesData,frame_data)
     
     for _,player := range onlinePlayersInRoom{
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
        room.players=append(room.players[:v], room.players[v+1:]...)
        rm_count++;
    }
}
