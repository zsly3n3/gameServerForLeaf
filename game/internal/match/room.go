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

// type RoomDataType int //房间类型,匹配类型还是邀请类型

// const (
//     SinglePersonMatching RoomDataType = iota + 1
//     EndlessMode
// 	Invite
// )

const min_MapWidth = 24
const min_MapHeight = 18
const time_interval = 50//50毫秒
var map_factor = 200

const RoomCloseTime = 15*time.Second//房间入口关闭时间

var MaxPlayingTime time.Duration

const FirstFrameIndex = 0//第一帧索引

const MaxEnergyPower = 2000 //全场最大能量值
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
    
    pid_auto int //每进入一个玩家 自增1
    
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
    robotPaths *RobotPaths
}

type RobotPaths struct {
    Mutex *sync.Mutex
    RobotPath map[int]struct{} //key为pathIndex,正在使用中
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
    Points []datastruct.EnergyPoint
    Action msg.PlayerIsDied
    AddEnergy int
}


type HistoryFrameData struct {
    Mutex *sync.RWMutex //读写互斥量
    FramesData []*msg.SC_RoomFrameDataContent
    IsClean bool
}

type RoomUnlockedData struct {
     parentMatch ParentMatch
     leastPeople int
     maxPeopleInRoom int //每个房间最大人数
     isExistTicker bool
     ticker *time.Ticker
     points_ch chan []datastruct.EnergyPoint
     pointData *EnergyPointData
     allowList []string//允许列表
     roomType datastruct.GameModeType//房间类型
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
     firstFramePoint []datastruct.EnergyPoint //第一帧的能量点数据
}

type EnergyPowerData struct {
     Mutex *sync.Mutex //读写互斥量
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
    PathIndex int //离线玩家使用    
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

func CreateRoom(r_type datastruct.GameModeType,connUUIDs []string,r_id string,parentMatch ParentMatch,leastPeople int,maxPeopleInRoom int)*Room{
    //测试
    if r_type == datastruct.SinglePersonMode || r_type == datastruct.InviteMode{
        MaxPlayingTime = 15 * time.Second
    }
    room := new(Room)
    room.Mutex = new(sync.RWMutex)
    room.pid_auto = 0
    rebotsNum:=leastPeople-len(connUUIDs)
    room.createGameMap(map_factor)
    room.createRoomUnlockedData(r_type,connUUIDs,r_id,parentMatch,rebotsNum,leastPeople,maxPeopleInRoom)
    room.createHistoryFrameData()
    room.createEnergyPowerData()
    room.createEnergyExpend()
    room.createPlayersDiedData()
    room.createLeftList(maxPeopleInRoom)
    
    room.currentFrameIndex = FirstFrameIndex
    room.onlineSyncPlayers = make([]datastruct.Player,0,maxPeopleInRoom)
    room.offlineSyncPlayers = make([]datastruct.Player,0,maxPeopleInRoom-1)
    
    room.IsOn = true
    room.players = make([]string,0,maxPeopleInRoom)
    room.playersData = NewPlayersFrameData()

    room.CreateRobotPaths()
    
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
     if len(room.robots.names) > 0{
        db.Module.UpdateRobotNamesState(room.robots.names)
     }
     room.unlockedData.parentMatch.RemoveRoomWithID(room.unlockedData.roomId)
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
    
    play_id:= room.pid_auto +room.unlockedData.rebotsNum
    
    if length == room.unlockedData.maxPeopleInRoom - 1 {
       room.IsOn = false
    }
    room.players=append(room.players,connUUID)
    room.pid_auto++
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
        room.GetCreateAction(play_id,datastruct.DefaultReliveFrameIndex,player.NickName,player.Avatar)
        syncFinished = true
    }else{
        content.CurrentFrameIndex = room.currentFrameIndex-1
        room.sendInitRoomDataToAgent(player,&content,play_id)
    }
    return syncFinished,content.CurrentFrameIndex
}

func (room *Room)GetCreateAction(play_id int,reliveFrameIndex int,playername string,playeravatar string)*msg.PlayerRelive{
     if play_id == datastruct.NULLID{
        panic("GetCreateAction play_id error")
     }
     randomIndex:=tools.GetRandomQuadrantIndex()
     point:=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
     action:=msg.GetCreatePlayerAction(play_id,point.X,point.Y,reliveFrameIndex,playername,0,playeravatar)
     actionData:=new(PlayerActionData)
     actionData.ActionType = action.Action.Action
     actionData.Data = action
     actionData.PathIndex = -1
     room.playersData.Set(play_id,actionData)//添加action 到 lastFrameIndex+1
     return action
}

func (room *Room)sendInitRoomDataToAgent(player *datastruct.Player,content *msg.SC_InitRoomDataContent,play_id int){
     if room.unlockedData.roomType != datastruct.EndlessMode {
        //测试
        content.GameTime = 15 * 1000 - room.currentFrameIndex*50
     }

     content.GameMode = int(room.unlockedData.roomType)
     msg_initRoom:=msg.GetInitRoomDataMsg(*content)
     player.Agent.WriteMsg(msg_initRoom)
     
     agentData:=player.Agent.UserData().(datastruct.AgentUserData)
     connUUID:=agentData.ConnUUID
     uid:=agentData.Uid
     rid:=room.unlockedData.roomId
     mode:=agentData.GameMode
     
     player.GameData.PlayId = play_id
     var extra datastruct.ExtraUserData
     extra.Avatar = agentData.Extra.Avatar
     extra.PlayName = agentData.Extra.PlayName
     extra.WaitRoomID = datastruct.NULLSTRING
     extra.RoomID = rid
     extra.IsSettle = false
     tools.ReSetAgentUserData(uid,mode,play_id,player.Agent,connUUID,extra)
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
               log.Debug("normal SyncFinished")
               room.GetCreateAction(player.GameData.PlayId,datastruct.DefaultReliveFrameIndex,player.NickName,player.Avatar)
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
                           room.GetCreateAction(player.GameData.PlayId,datastruct.DefaultReliveFrameIndex,player.NickName,player.Avatar)
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
    room.Mutex.Lock()
    if !room.IsOn{
       room.Mutex.Unlock()
       return false
    }
    isOn:=false
    syncFinished:=false
    currentFrameIndex:=-1
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

func (room*Room)goCreatePoints(maxRangeType msg.EnergyPointType){
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
           log.Debug("room.onlineSyncPlayers:%v",room.onlineSyncPlayers)
           removeOfflineSyncPlayersInRoom(room,offlineSyncPlayersIndex)//remove offline players
       }
       online_sync=make([]datastruct.Player,len(room.onlineSyncPlayers))
       copy(online_sync,room.onlineSyncPlayers)
       offline_sync=make([]datastruct.Player,len(room.offlineSyncPlayers))
       copy(offline_sync,room.offlineSyncPlayers)
       if expended_onlineConnUUID != datastruct.NULLSTRING && len(online_sync)>0{
        agentData:=online_sync[0].Agent.UserData().(datastruct.AgentUserData)
        expended_onlineConnUUID = agentData.ConnUUID
       }
    }

 syncNotFinishedPlayers:=onlinePlayersInRoom-len(online_sync)
 isRemoveHistory:=false
 log.Debug("rid:%v,onlinePlayersInRoom:%v,syncNotFinishedPlayers:%v",room.unlockedData.roomId,onlinePlayersInRoom,syncNotFinishedPlayers)
 if syncNotFinishedPlayers == 0&&!room.IsOn{
    log.Debug("rid:%v,room.IsOn:%v",room.unlockedData.roomId,room.IsOn)
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
        var points []datastruct.EnergyPoint
        select {
        case points = <-room.unlockedData.points_ch:
        default:
         points=nil
        }
        frame_data.CreateEnergyPoints = make([]datastruct.EnergyPoint,0)
        expended:=room.getEnergyExpended(expended_onlineConnUUID) 
        //log.Debug("expended:%v",expended)
        room.energyData.SetPower(expended)
        if points != nil && len(points)>0 && room.energyData.IsCreatePower(){
          //log.Debug("CreateEnergyPoints:")
          frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,points...)
        }
     }
     
     room.robots.Mutex.Lock()
     removeRobotsId:=make([]int,0,len(room.robots.robots))
     for _,robot:= range room.robots.robots{
        action_type,action:=room.getRobotAction(robot,currentFrameIndex)
        // if currentFrameIndex < 10 && currentFrameIndex > 0{
        //     log.Debug("robot 1 x:",action.(*msg.PlayerMoved).X,",y:",action.(*msg.PlayerMoved).Y)
        // }
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
     
     rm_action_ids:=make([]int,0)//需要删除的动作id

     for _,player := range online_sync{
         connUUID:=player.Agent.UserData().(datastruct.AgentUserData).ConnUUID
         pid:=player.GameData.PlayId
         action_type,action:=room.playersData.GetValue(player.Avatar,player.NickName,pid,currentFrameIndex,room,connUUID)
         log.Debug("name:%v,action_type:%v",player.NickName,action_type)
         if action != nil{
             if action_type==msg.Death{
                died:=action.(*PlayerDied)
                action=died.Action
                frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,died.Points...)
                if room.unlockedData.roomType == datastruct.EndlessMode{
                    rm_action_ids = append(rm_action_ids,pid)
                }
             }
             frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
         }
     }
     
     for _,player := range offline_sync{
        connUUID:=player.Agent.UserData().(datastruct.AgentUserData).ConnUUID
        pid:=player.GameData.PlayId
        action_type,action:=room.playersData.GetOfflineAction(player.Avatar,player.NickName,player.GameData.PlayId,currentFrameIndex,room,connUUID)
        if action != nil{
            if action_type==msg.Death{
               died:=action.(*PlayerDied)
               action=died.Action
               frame_data.CreateEnergyPoints = append(frame_data.CreateEnergyPoints,died.Points...)
               if room.unlockedData.roomType == datastruct.EndlessMode{
                rm_action_ids = append(rm_action_ids,pid)
               }
            }
            frame_data.PlayerFrameData = append(frame_data.PlayerFrameData,action)
        }
     }
     
     frame_content.FramesData = append(frame_content.FramesData,frame_data)
     msg:=msg.GetRoomFrameDataMsg(&frame_content)
    
    
    for _,player := range online_sync{
         player.Agent.WriteMsg(msg)
    }
     
     room.playersData.DeleteDatas(rm_action_ids)
     
     room.history.Mutex.Lock()
     defer room.history.Mutex.Unlock()
     isClean:=room.history.IsClean
  
     if !isRemoveHistory&&!isClean{
        room.history.FramesData = append(room.history.FramesData,&frame_content)
        for i:=0;i<syncNotFinishedPlayers;i++{
            isClosed:=safeSendSync(room.unlockedData.startSync,struct{}{})
            if isClosed{
                break
            }
        }
     }else{ 
        if !isClean{
            room.history.FramesData=room.history.FramesData[:0]
            room.history.IsClean = true
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


func (room *Room)createRoomUnlockedData(r_type datastruct.GameModeType,connUUIDs []string,r_id string,parentMatch ParentMatch,rebotsNum int,leastPeople int,maxPeopleInRoom int){
    unlockedData:=new(RoomUnlockedData)
    unlockedData.points_ch = make(chan []datastruct.EnergyPoint,2)
    unlockedData.rebotMoveAction = make(chan msg.Point,leastPeople-1+maxPeopleInRoom-1)
    unlockedData.startSync = make(chan struct{},maxPeopleInRoom-1)
    unlockedData.pointData = room.createEnergyPointData(room.gameMap.width,room.gameMap.height)
    unlockedData.allowList = connUUIDs
    unlockedData.roomId = r_id
    unlockedData.isExistTicker = false
    unlockedData.parentMatch = parentMatch
    unlockedData.rebotsNum = rebotsNum
    unlockedData.leastPeople = leastPeople
    unlockedData.maxPeopleInRoom = maxPeopleInRoom
    unlockedData.roomType = r_type
    room.unlockedData = unlockedData
    go room.goCreatePoints(msg.TypeB)
    go room.goCreateMovePoint()
}

func (room *Room)createHistoryFrameData(){
    history:=new(HistoryFrameData)
    history.Mutex = new(sync.RWMutex)
    rs:=MaxPlayingTime/(time_interval*time.Millisecond)
    history.FramesData = make([]*msg.SC_RoomFrameDataContent,0,rs)
    history.IsClean = false
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
    if num > 0{
        robots.names = db.Module.GetRobotNames(num)
        i:=0;
        for _,v := range robots.names{
           pathIndex:= room.getRandPathIndex(-1)
           pt:=room.getRobotPath(pathIndex,0)
           robot:=tools.CreateRobot(v,i,isRelive,room.unlockedData.pointData.quadrant,datastruct.DefaultReliveFrameIndex,pt)
           robot.PathIndex = pathIndex
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

func (diedData *PlayersDiedData)Append(values []datastruct.PlayerDiedData){
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

func safeSendPoint(ch chan []datastruct.EnergyPoint, value []datastruct.EnergyPoint) (closed bool) {
    defer func() {
        if recover() != nil {
            closed = true
        }
	}()
    ch <- value // panic if ch is closed
    return false // <=> closed = false; return
}

func safeClosePoint(ch chan []datastruct.EnergyPoint) (justClosed bool) {
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

func (data *PlayersFrameData)GetValue(avatar string,name string,pid int,currentFrameIndex int,room *Room,connUUID string)(msg.ActionType,interface{}){
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
             case datastruct.InviteMode:
                  fallthrough
             case datastruct.SinglePersonMode:
                  isCreate = true
                  reliveFrameIndex=currentFrameIndex+offsetFrames
             case datastruct.EndlessMode:
                  log.Debug("GetValue Death %v",name)
                  room.playerExit(connUUID,pid)
             }
             if isCreate{
                room.relive(actionData,pid,reliveFrameIndex,name,actionData.Data.(*PlayerDied).AddEnergy,false,avatar)
             }
          case msg.Move:
             v= *(actionData.Data.(*msg.PlayerMoved))
             rs_actionType = msg.Move
            //测试
            //  test:=*(actionData.Data.(*msg.PlayerMoved))
            //  log.Debug("r_id:%v,x:%v,y:%v,frameIndex:%v",room.unlockedData.roomId,test.X,test.Y,currentFrameIndex)
        }
    }
    return rs_actionType,v
}

func (data *PlayersFrameData)GetOfflineAction(avatar string,name string,pid int,currentFrameIndex int,room *Room,connUUID string)(msg.ActionType,interface{}){
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
                
                  var point msg.Point
                  if actionData.PathIndex == -1{
                     point =room.getMovePoint() 
                  }else{
                     point = msg.Point{
                         X:0,
                         Y:0,
                     }
                  }
                  action:=msg.GetCreatePlayerMoved(pid,point.X,point.Y,msg.DefaultSpeed)
                  offlinemove_action:=tools.CreateOfflinePlayerMoved(action,1)
                  actionData.ActionType = offlinemove_action.Action.Action
                  actionData.Data = offlinemove_action
               }else{
                  v = nil
               }
          case msg.Death:
             v=actionData.Data
             rs_actionType = msg.Death
             isCreate:=false
             reliveFrameIndex:=datastruct.DefaultReliveFrameIndex
             switch room.unlockedData.roomType{
               case datastruct.InviteMode:
                    fallthrough            
               case datastruct.SinglePersonMode:
                    isCreate = true
                    reliveFrameIndex=currentFrameIndex+offsetFrames
               case datastruct.EndlessMode:
                    isExist:=room.CheckLeftlist(connUUID)
                    if isExist{
                       log.Debug("GetOfflineAction Death %v",name)
                       room.playerExit(connUUID,pid)
                    }
             }
             if isCreate{
                room.relive(actionData,pid,reliveFrameIndex,name,actionData.Data.(*PlayerDied).AddEnergy,true,avatar)
             }
          case msg.Move:
            rs_actionType = msg.Move
            switch actionData.Data.(type){
            case *msg.PlayerMoved:
                 
                 move_action:=actionData.Data.(*msg.PlayerMoved)
                 v= *move_action
                 offlinemove_action:=tools.CreateOfflinePlayerMoved(move_action,-1)
                 actionData.ActionType = offlinemove_action.Action.Action
                 actionData.Data = offlinemove_action
                 
            case *msg.OfflinePlayerMoved:
                 
                 offlinemove_action:=actionData.Data.(*msg.OfflinePlayerMoved)

                 ptr_action := &offlinemove_action.Action

                
                 var point msg.Point
                 pathIndex:=actionData.PathIndex
                 if pathIndex == -1||offlinemove_action.MoveStep==-1{
                    point = room.getMovePoint() 
                 }else{
                    point=room.getRobotPath(pathIndex,offlinemove_action.MoveStep)
                    offlinemove_action.MoveStep++
                 }
                 ptr_action.Speed = 1
                 ptr_action.X = point.X
                 ptr_action.Y = point.Y
                //  startIndex:=offlinemove_action.StartFrameIndex
                //  directionInterval:=offlinemove_action.DirectionInterval
                //  speedInterval:=offlinemove_action.SpeedInterval
                //  var ptr_action *msg.PlayerMoved
                  
                //  if (currentFrameIndex-startIndex)*time_interval % (directionInterval*1000) == 0 {
                //      lastSpeed:=offlinemove_action.Action.Speed
                //      point:=room.getMovePoint()
                //      ptr_action = msg.GetCreatePlayerMoved(pid,point.X,point.Y,lastSpeed)
                //  }else{
                //      ptr_action = &offlinemove_action.Action
                //  }

                //  if (currentFrameIndex-startIndex)*time_interval % (speedInterval*1000) == 0{
                //     speedDuration:= tools.GetRandomSpeedDuration()
                //     offlinemove_action.StopSpeedFrameIndex = currentFrameIndex-startIndex+speedDuration*(1000/time_interval)
                //     ptr_action.Speed = tools.GetRandomSpeed()
                // }
    
                // if offlinemove_action.StopSpeedFrameIndex != 0 && currentFrameIndex-startIndex == offlinemove_action.StopSpeedFrameIndex{
                //     offlinemove_action.StopSpeedFrameIndex = 0
                //     ptr_action.Speed = msg.DefaultSpeed
                // }
                
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
                 //point:=room.getMovePoint()//测试
                 point:=msg.Point{
                     X:0,
                     Y:0,
                 }
                 robot.MoveStep = 1
                 action:=msg.GetCreatePlayerMoved(robot.Id,point.X,point.Y,msg.DefaultSpeed)
                 robot.Action = action
            }else{
                rs = nil
            }
       case *msg.PlayerMoved:
            //var ptr_action *msg.PlayerMoved
            action:=current_action.(*msg.PlayerMoved)
            action.Speed = 1
            pt:=room.getRobotPath(robot.PathIndex,robot.MoveStep)
            action.X = pt.X
            action.Y = pt.Y
            robot.MoveStep++
            //ptr_action = action
            /* //测试
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
            */
            robot.Action=action
            rs=action
            rs_type = msg.Move
            
        case *PlayerDied:
            rs = current_action//*PlayerDied
            rs_type = msg.Death
            addEnergy:= current_action.(*PlayerDied).AddEnergy
            room.DeleteRandPathIndex(robot.PathIndex)
            if robot.IsRelive {
                pathIndex:= room.getRandPathIndex(robot.PathIndex)
                robot.PathIndex = pathIndex
                pt:=room.getRobotPath(pathIndex,0)
                action:=tools.GetCreateRobotAction(pt,robot.Id,room.unlockedData.pointData.quadrant,offsetFrames+currentFrameIndex,robot.NickName,addEnergy,tools.GetRobotAvatar())
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
func (room *Room)HandleDiedData(values []datastruct.PlayerDiedData){
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
                   //log.Debug("HandleDiedData_0 v.FrameIndex:%v,currentFrameIndex:%v",v.FrameIndex,currentFrameIndex)
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
         
         var action msg.PlayerIsDied
         action.Action = msg.Death
         action.PlayerId = p_id
         
         p_died:=new(PlayerDied)
         expend:=0
         p_died.Points,expend= tools.CheckScalePoints(v.Points)
         if expend > 0{
            room.energyData.SetPower(-expend)
         }
         p_died.Action = action
         p_died.AddEnergy = v.AddEnergy
         
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
    if room.unlockedData.roomType == datastruct.SinglePersonMode || room.unlockedData.roomType == datastruct.InviteMode{
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

func (room *Room)removePlayer(connUUID string,pid int){
    room.Mutex.Lock()
    defer room.Mutex.Unlock()
    rm_index:=datastruct.NULLID
    for index,v := range room.players{
        if connUUID == v{
            rm_index=index
            break
        }
    }
    
    rm_index1:=datastruct.NULLID
    for index,v := range room.onlineSyncPlayers{
        if pid == v.GameData.PlayId{
            rm_index1=index
            break
        }
    }
    rm_index2:=datastruct.NULLID
    for index,v := range room.offlineSyncPlayers{
        if pid == v.GameData.PlayId{
            rm_index2=index
            break
        }
    }
   
    if rm_index!=datastruct.NULLID{
        room.players=append(room.players[:rm_index], room.players[rm_index+1:]...)
    }
    if rm_index1!=datastruct.NULLID{
        room.onlineSyncPlayers=append(room.onlineSyncPlayers[:rm_index1],room.onlineSyncPlayers[rm_index1+1:]...)
    }
    if rm_index2!=datastruct.NULLID{
        room.offlineSyncPlayers=append(room.offlineSyncPlayers[:rm_index2],room.offlineSyncPlayers[rm_index2+1:]...)
    }
}

func (room *Room)HandlePlayerRelive(pid int,playername string,playeravatar string){
     actionData,tf:=room.playersData.CheckValue(pid)
     if !tf{
        actionData=new(PlayerActionData)
        room.relive(actionData,pid,datastruct.DefaultReliveFrameIndex,playername,0,false,playeravatar)
        actionData.PathIndex = -1
        room.playersData.Set(pid,actionData)
     }
}
func (room *Room)relive(actionData *PlayerActionData,pid int,reliveFrameIndex int,playername string,addEnergy int,isRobot bool,playeravatar string){
     var point msg.Point
     pathIndex:=-1
     if isRobot{
        pathIndex = room.getRandPathIndex(-1)
        point=room.getRobotPath(pathIndex,0)
     }else{
        randomIndex:=tools.GetRandomQuadrantIndex()
        point=tools.GetCreatePlayerPoint(room.unlockedData.pointData.quadrant[randomIndex],randomIndex) 
     }
     action:=msg.GetCreatePlayerAction(pid,point.X,point.Y,reliveFrameIndex,playername,addEnergy,playeravatar)
     actionData.ActionType = action.Action.Action
     actionData.Data = action
     actionData.PathIndex = pathIndex
}

func (room *Room)getRobotPath(index int,step int)msg.Point{
    paths := db.Module.GetRobotPaths()
    path := paths[index]
    v,tf:= path[step]
    var pt msg.Point
    if tf{
       pt = v
    }else{
       pt = room.getMovePoint()
    }
    return pt
}

func (room *Room)CreateRobotPaths(){
    paths:=new(RobotPaths)
    paths.Mutex = new(sync.Mutex)
    paths.RobotPath = make(map[int]struct{})
    room.robotPaths = paths
}

func (room *Room)getRandPathIndex(lastIndex int) int{
     paths := db.Module.GetRobotPaths()
     randMap:=make(map[int]struct{})
     for i:=0;i<len(paths);i++{
         randMap[i]=struct{}{}
     }
     room.robotPaths.Mutex.Lock()
     defer room.robotPaths.Mutex.Unlock()
     for k,_ := range room.robotPaths.RobotPath{
         delete(randMap,k)
     }
     rs:=-1
     switch len(randMap){
     case 0:
        randMap:=make(map[int]struct{})
        for i:=0;i<len(paths);i++{
            randMap[i]=struct{}{}
        }
        rand_slice:=make([]int,0,len(randMap)) 
        for k,_ := range randMap{
          rand_slice = append(rand_slice,k)                
        }
        rs=tools.GetRandomFromSlice(rand_slice)
         //panic("路径地图数不能小于机器人个数")
        return rs
     case 1:
        for k,_ := range randMap{
            rs = k
        }
     default:
        delete(randMap,lastIndex)
        if len(randMap) <= 1{
           for k,_ := range randMap{
               rs = k
           }
        }else{
           rand_slice:=make([]int,0,len(randMap)) 
           for k,_ := range randMap{
             rand_slice = append(rand_slice,k)                
           }
           rs=tools.GetRandomFromSlice(rand_slice)
        }
     }
     room.robotPaths.RobotPath[rs]=struct{}{}
     return rs
}

func (room *Room)DeleteRandPathIndex(k int){
    room.robotPaths.Mutex.Lock()
    defer room.robotPaths.Mutex.Unlock()
    delete(room.robotPaths.RobotPath,k)
}

func (room *Room)playerExit(connUUID string,pid int){
    room.unlockedData.parentMatch.RemovePlayer(connUUID)
    room.removePlayer(connUUID,pid)
}

func (data *PlayersFrameData)DeleteDatas(pids []int){
     data.Mutex.Lock()
     data.Mutex.Unlock()
     for _,v:=range pids{
        delete(data.Data,v)
     }
}
