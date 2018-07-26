package singleMatch

import (
	"server/datastruct"
	"sync"
	"time"
	"github.com/name5566/leaf/gate"
	"server/db"
	"server/tools"
	"server/msg"
	"server/game/internal/match"
	"github.com/name5566/leaf/log"
)

const LeastPeople = 10


/*单人匹配*/
type SingleMatch struct {
	Times time.Duration
	MaxWaitTime time.Duration
	Pool_Capacity int
	ticker *time.Ticker
	isExistTicker bool
	rooms *match.Rooms
	onlinePlayers *datastruct.OnlinePlayers
	singleMatchPool *SingleMatchingPool
	actionPool *match.MatchActionPool
}

func NewSingleMatch()*SingleMatch{
	singleMatch:=new(SingleMatch)
	singleMatch.init()
	return singleMatch
}

func (singleMatch *SingleMatch)init(){
	singleMatch.isExistTicker = false
	singleMatch.Times = 1*time.Second //定时器多少时间执行一次
	singleMatch.MaxWaitTime = 5*time.Second//玩家最大等待时间多少秒
	singleMatch.Pool_Capacity = 10 //满足有多少个人就开始游戏
	singleMatch.onlinePlayers = datastruct.NewOnlinePlayers()
	singleMatch.singleMatchPool = newSingleMatchingPool(singleMatch.Pool_Capacity)
	singleMatch.actionPool = match.NewMatchActionPool(singleMatch.Pool_Capacity)
	singleMatch.rooms = match.NewRooms()
}

func (match *SingleMatch)addPlayer(connUUID string,a gate.Agent,uid int){
	match.addOnlinePlayer(connUUID,a,uid)
	match.actionPool.AddInMatchActionPool(connUUID)
}

func (match *SingleMatch)RemovePlayer(connUUID string){
	match.onlinePlayers.Delete(connUUID)
	match.actionPool.RemoveFromMatchActionPool(connUUID)
}

func (match *SingleMatch)RemovePlayerFromMatchingPool(connUUID string){
	  match.RemovePlayer(connUUID)
	  match.singleMatchPool.Mutex.Lock()
	  defer match.singleMatchPool.Mutex.Unlock()
	  rm_index:=-1
	  for index,v := range match.singleMatchPool.Pool{
		  if connUUID == v{
			  rm_index = index
			  break
		  }
	  }
	  if rm_index != -1{
		 match.singleMatchPool.Pool=append(match.singleMatchPool.Pool[:rm_index], match.singleMatchPool.Pool[rm_index+1:]...)
	  }
}

func (match *SingleMatch)addOnlinePlayer(connUUID string,a gate.Agent,uid int){
	match.onlinePlayers.Lock.Lock()
	 defer match.onlinePlayers.Lock.Unlock()
	 v, ok := match.onlinePlayers.Bm[connUUID];
	 if !ok {
		 user:=db.Module.GetUserInfo(uid)
		 player:=datastruct.CreatePlayer(user)
		 player.Agent = a
		 match.onlinePlayers.Bm[connUUID]=*player
	 }else{
		 v.GameData.EnterType = datastruct.NULLWay
		 v.GameData.RoomId = datastruct.NULLSTRING
		 v.GameData.PlayId = datastruct.NULLID
		 match.onlinePlayers.Bm[connUUID]=v
	 }
}
func (match *SingleMatch)CheckActionPool(connUUID string) bool{
	  return match.actionPool.Check(connUUID)
}

func (match *SingleMatch)Matching(connUUID string, a gate.Agent,uid int){
	match.addPlayer(connUUID,a,uid)
	
	//willEnterRoom 是否将要加入了房间
	r_id,willEnterRoom:=match.rooms.GetFreeRoomId()
    
	if !willEnterRoom{
	   match.singleMatchPool.Mutex.Lock()
	   defer match.singleMatchPool.Mutex.Unlock()
	   num:=len(match.singleMatchPool.Pool)
	   if num<LeastPeople{
		  match.singleMatchPool.Pool=append(match.singleMatchPool.Pool,connUUID)
		  match.createTicker()
		  if num == LeastPeople-1{
			//check player is online or offline
			//offline player is removed from pool
			//if all online create room
			removeIndex,_:=match.getOfflinePlayers()
			rm_num:=len(removeIndex)
			if rm_num<=0{//池中没有离线玩家,则创建房间
				match.cleanPoolAndCreateRoom()
			}else{
				match.removeOfflinePlayersInPool(removeIndex)
			}
		  }
	   }
	}else{
		 player,tf:=match.onlinePlayers.GetAndUpdateState(connUUID,datastruct.FreeRoom,r_id)
		 if tf{
		 	player.Agent.WriteMsg(msg.GetMatchingEndMsg(r_id))
		 }
	}
}

func (match *SingleMatch)createTicker(){
    if !match.isExistTicker {
		match.isExistTicker = true
		match.ticker = time.NewTicker(match.Times)
        go match.selectTicker()
    } 
}

func (match *SingleMatch)stopTicker(){
    if match.ticker != nil{
	   match.ticker.Stop() 
       match.isExistTicker = false
    }
}

func (match *SingleMatch)selectTicker(){
    for{
        select {
         case <-match.ticker.C:
            match.computeMatchingTime()
        }
    }
}

func (match *SingleMatch)getOfflinePlayers() ([]int, map[string]datastruct.Player){
    tmp_map:=match.onlinePlayers.Items()
	
    online_map:=make(map[string]datastruct.Player)
    
    removeIndex:=make([]int,0,LeastPeople)
    
    var online_player datastruct.Player
    online_key:=datastruct.NULLSTRING
    
    for index,v := range match.singleMatchPool.Pool{
        isOnline:=false
        for key,player :=range tmp_map{
            if key == v{
                isOnline=true
                online_key = key
                online_player = player
                break
            }
        }
        if isOnline{
            online_map[online_key]= online_player
            delete(tmp_map, online_key)//移除对比过的数据,减少空间复杂度
        }else{
            removeIndex=append(removeIndex,index)//保存离线玩家
        }
    }
    return removeIndex,online_map
}

func (match *SingleMatch)removeOfflinePlayersInPool(removeIndex []int){
    rm_count:=0
    for index,v := range removeIndex {
        if index!=0{
           v = v-rm_count
        }
        match.singleMatchPool.Pool=append(match.singleMatchPool.Pool[:v], match.singleMatchPool.Pool[v+1:]...)
        rm_count++
    }
}

func (match *SingleMatch)cleanPoolAndCreateRoom(){
	match.stopTicker()
    arr:=make([]string,len(match.singleMatchPool.Pool))
    copy(arr,match.singleMatchPool.Pool)
	match.singleMatchPool.Pool=match.singleMatchPool.Pool[:0]//clean pool
    go match.createMatchingTypeRoom(arr)
}

func (singleMatch *SingleMatch)createMatchingTypeRoom(playerUUID []string){
	log.Debug("单人匹配完成，创建房间")
    r_uuid:=tools.UniqueId()
	players:=singleMatch.onlinePlayers.GetsAndUpdateState(playerUUID,datastruct.FromMatchingPool,r_uuid)
    room:=match.CreateRoom(match.SinglePersonMatching,playerUUID,r_uuid,singleMatch,LeastPeople,20)
    singleMatch.rooms.Set(r_uuid,room)
    for _,play := range players{
        play.Agent.WriteMsg(msg.GetMatchingEndMsg(r_uuid))
    }
}

func (match *SingleMatch)computeMatchingTime(){
	match.singleMatchPool.Mutex.Lock()
    defer match.singleMatchPool.Mutex.Unlock()
    num:=len(match.singleMatchPool.Pool)
    if num >0{
        removeIndex,online_map:=match.getOfflinePlayers()
        rm_num:=len(removeIndex)
        if rm_num>0{//删除池中离线玩家
		   match.removeOfflinePlayersInPool(removeIndex)
        }
        now_t := time.Now()
        for _,player := range online_map{
            rs_sub:=now_t.Sub(player.GameData.StartMatchingTime)
            if rs_sub>=match.MaxWaitTime{
                match.cleanPoolAndCreateRoom()
                break
            }
        }
    }else{
	   match.stopTicker()
    }
}


func (match *SingleMatch)PlayerMoved(r_id string,play_id int,moveData *msg.CS_MoveData){
	ok,room:=match.rooms.Get(r_id)
    if ok&&room.IsEnableUpdatePlayerAction(play_id){
	//    log.Debug("r_id:%v,data:%v",r_id,moveData.MsgContent)	
       room.GetPlayerMovedMsg(play_id,moveData)
    }
}

func (singleMatch *SingleMatch)PlayerJoin(connUUID string,joinData *msg.CS_PlayerJoinRoom){
	player,tf:=singleMatch.onlinePlayers.CheckAndCleanState(connUUID,datastruct.NULLWay,datastruct.NULLSTRING)
    if tf{
        r_id := joinData.MsgContent.RoomID
        if player.GameData.EnterType == datastruct.FreeRoom&&player.GameData.RoomId==r_id{
		   ok,room:=singleMatch.rooms.Get(r_id)
		   if ok{
			isOn:=room.Join(connUUID,player,false)
			if isOn{
			  log.Debug("通过遍历空闲房间进入")
			  singleMatch.actionPool.RemoveFromMatchActionPool(connUUID)	
			}else{
			  go singleMatch.handleRoomOff(player.Agent,connUUID,player.Uid)
			}
		   }else{
			go singleMatch.handleRoomOff(player.Agent,connUUID,player.Uid)		
		   }
        }else if player.GameData.EnterType == datastruct.FromMatchingPool{
			ok,room:=singleMatch.rooms.Get(r_id)
			if ok{
				isExist:=false
				for _,v:=range room.GetAllowList(){
					if v == connUUID{
						isOn:=room.Join(connUUID,player,true)
						if isOn{
						  log.Debug("通过匹配池进入")
						  singleMatch.actionPool.RemoveFromMatchActionPool(connUUID)	
						}else{
						  go singleMatch.handleRoomOff(player.Agent,connUUID,player.Uid)
						}
						isExist = true
						break
					}
				}
				if !isExist{
				   player.Agent.WriteMsg(msg.GetJoinInvalidMsg())
				}
			}
        }else{
            player.Agent.WriteMsg(msg.GetJoinInvalidMsg())
        }
    }
}

func (match *SingleMatch)handleRoomOff(a gate.Agent,connUUID string,uid int){
    a.WriteMsg(msg.GetReMatchMsg())
    match.Matching(connUUID,a,uid)
}

func (match *SingleMatch)EnergyExpended(expended int,agentUserData datastruct.AgentUserData){
       connUUID:=agentUserData.ConnUUID
	   r_id:=agentUserData.Extra.RoomID
	   ok,room:=match.rooms.Get(r_id)
	   if ok{
		room.EnergyExpended(connUUID,expended)
	   }
}

func (singleMatch *SingleMatch)PlayersDied(r_id string,values []datastruct.PlayerDiedData){
	ok,room:=singleMatch.rooms.Get(r_id)
	if ok{
		room.HandleDiedData(values)
	}
}

func (match *SingleMatch)RemoveRoomWithID(uuid string){
	match.rooms.Delete(uuid)
}
func (match *SingleMatch)GetOnlinePlayersPtr() *datastruct.OnlinePlayers{
     return match.onlinePlayers
}
func (match *SingleMatch)PlayerLeftRoom(r_id string,connUUID string){
	 match.RemovePlayer(connUUID)
}


/*单人匹配池*/
type SingleMatchingPool struct {
	 Mutex *sync.RWMutex //读写互斥量
	 Pool  []string //存放玩家uuid
}

func newSingleMatchingPool(poolCapacity int)*SingleMatchingPool{
	singleMatchingPool:=new(SingleMatchingPool)
	singleMatchingPool.init(poolCapacity)
	return singleMatchingPool
}

func (pool *SingleMatchingPool)init(poolCapacity int){
	  pool.Mutex = new(sync.RWMutex)
	  pool.Pool = make([]string,0,poolCapacity)
}
