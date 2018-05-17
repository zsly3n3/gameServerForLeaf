package internal
import (
	"server/db"
    "reflect"  
    "server/msg"
    "server/datastruct"
    "github.com/name5566/leaf/gate"  
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/network/json"
    "server/tools"
    "time"
)

// 异步处理  
func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
    handleMsg(&msg.CS_PlayerMatching{}, handlePlayerMatching)
    handleMsg(&msg.CS_PlayerCancelMatching{}, handleCancelMatching)
    handleMsg(&msg.CS_PlayerJoinRoom{}, handlePlayerJoinRoom)
}

func handlePlayerJoinRoom(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID
    player,tf:=onlinePlayers.CheckAndCleanState(connUUID,datastruct.EmptyWay,datastruct.NULLSTRING)
    if tf{
        m := args[0].(*msg.CS_PlayerJoinRoom)
        r_id := m.MsgContent.RoomID
        if player.GameData.EnterType == datastruct.FreeRoom&&player.GameData.RoomId==r_id{
           room:=rooms.Get(r_id)
           room.Mutex.Lock()
           defer room.Mutex.Unlock()
           if room.IsOn{
              room.Join(&player)
              log.Debug("通过遍历空闲房间进入")
           }else{
               go handleRoomOff(a,connUUID)
           }
        }else if player.GameData.EnterType == datastruct.FromMatchingPool{
            room:=rooms.Get(r_id)
            for _,v:=range room.AllowList{
                if v == connUUID{
                    log.Debug("通过匹配池进入")
                    room.Join(&player)
                    break
                }
            }
        }else{
            a.WriteMsg(msg.GetJoinInvalidMsg())
        }
    }
}

func handleRoomOff(a gate.Agent,connUUID string){
    a.WriteMsg(msg.GetReMatchMsg())
    matchingPlayers(connUUID)
}

func handleCancelMatching(args []interface{}){
    a := args[1].(gate.Agent)
    if !tools.IsValid(a.UserData()){
       return
    }
    agentUserData := a.UserData().(datastruct.AgentUserData)
    connUUID:=agentUserData.ConnUUID
    removeFromMatchActionPool(connUUID)
    m_pool:=getPool()
    m_pool.Mutex.Lock()
    defer m_pool.Mutex.Unlock()
    rm_index:=-1
    for index,v := range m_pool.Pool{
        if connUUID == v{
            rm_index = index
            break
        }
    }
    if rm_index != -1{
        m_pool.Pool=append(m_pool.Pool[:rm_index], m_pool.Pool[rm_index+1:]...)
    }
}
//收到匹配消息的时候加入池，主动离开和自动离开在池中删除，
//完成匹配后，在池中删除

func handlePlayerMatching(args []interface{}) {
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
    
    if checkMatchActionPool(connUUID){//已在匹配中
        msgHeader.MsgName = "SC_PlayerAlreadyMatching"
        a.WriteMsg(&msg.SC_PlayerAlreadyMatching{
            MsgHeader:msgHeader,
        })
        return
    }
    
    //不存在则加入在线玩家列表
    if PlayerIsExist(connUUID,a,uid){
       
    }

    addInMatchActionPool(connUUID) 
    ChanRPC.Go("MatchingPlayers",connUUID) //玩家匹配
    
    msgHeader.MsgName = "SC_PlayerMatching"

    var msgContent msg.SC_PlayerMatchingContent
    msgContent.IsMatching =true

    a.WriteMsg(&msg.SC_PlayerMatching{  
        MsgHeader:msgHeader,
        MsgContent:msgContent,
    })
}  

func matchingPlayers(p_uuid string){
    //willEnterRoom 是否将要加入了房间
    
    r_id,willEnterRoom:=rooms.GetFreeRoomId()

    if !willEnterRoom{
	   m_pool:=getPool()
       m_pool.Mutex.Lock()
       defer m_pool.Mutex.Unlock()
	   num:=len(m_pool.Pool)
	   if num<LeastPeople{
        m_pool.Pool=append(m_pool.Pool,p_uuid)
        createTicker()
        if num == LeastPeople-1{
            //check player is online or offline
            //offline player is removed from pool
            //if all online create room
            removeIndex,_:=getOfflinePlayers(m_pool)
            rm_num:=len(removeIndex)
            if rm_num<=0{//池中没有离线玩家,则创建房间
                cleanPoolAndCreateRoom(m_pool)
            }else{
                removeOfflinePlayersInPool(m_pool,removeIndex)
            }
        }
       }
    }else{
        player,tf:=onlinePlayers.GetAndUpdateState(p_uuid,datastruct.FreeRoom,r_id)
        if tf{
            player.Agent.WriteMsg(msg.GetMatchingEndMsg(r_id))
        }
    }
}

func cleanPoolAndCreateRoom(m_pool *datastruct.MatchingPool){
    stopTicker()
    arr := m_pool.Pool[:] //copy data
    m_pool.Pool=m_pool.Pool[:0]//clean pool
    go createMatchingTypeRoom(arr)
}

func createMatchingTypeRoom(playerUUID []string){
    r_uuid:=tools.UniqueId()
    players:=onlinePlayers.GetsAndUpdateState(playerUUID,datastruct.FromMatchingPool,r_uuid)
    room:=createRoom(playerUUID,Matching,r_uuid)
    rooms.Set(r_uuid,room)
    for _,play := range players{
        play.Agent.WriteMsg(msg.GetMatchingEndMsg(r_uuid))
    }
}



func createPlayer(user *datastruct.User) *datastruct.Player{
    var player datastruct.Player
    player.Avatar=user.Avatar
    player.Id=user.Id
    player.NickName=user.NickName
    var game_data datastruct.PlayerGameData
    game_data.StartMatchingTime = time.Now()
    game_data.EnterType = datastruct.EmptyWay
    game_data.RoomId = datastruct.NULLSTRING
    player.GameData = game_data
    return &player
}

func removePlayer(key string){
    onlinePlayers.Delete(key)   
}

func createTicker(){
	if ticker == nil{
        ticker = time.NewTicker(times)
    }
}
func stopTicker(){
    if ticker != nil{
        ticker.Stop()
        ticker=nil
    }
}

func selectTicker(){
     for {
		 if ticker != nil{
			select {
			case <-ticker.C:
				computeMatchingTime()
			}
		 }
	 }
}

func computeMatchingTime(){
    m_pool:=getPool()
    m_pool.Mutex.Lock()
    defer m_pool.Mutex.Unlock()
    
    num:=len(m_pool.Pool)
    if num >0{
        removeIndex,online_map:=getOfflinePlayers(m_pool)
        rm_num:=len(removeIndex)
        if rm_num>0{//删除池中离线玩家
           removeOfflinePlayersInPool(m_pool,removeIndex)
        }
        now_t := time.Now()
        
        for _,player := range online_map{
            rs_sub:=now_t.Sub(player.GameData.StartMatchingTime)
            if rs_sub>=MaxWaitTime{
                cleanPoolAndCreateRoom(m_pool)
                break
            }
        }
    }else{
      stopTicker()
    }
}
func getOfflinePlayers(m_pool *datastruct.MatchingPool) ([]int, map[string]datastruct.Player){
    tmp_map:=onlinePlayers.Items()
    
    online_map:=make(map[string]datastruct.Player)
    
    removeIndex:=make([]int,0,LeastPeople)
    
    var online_player datastruct.Player
    online_key:=datastruct.NULLSTRING
    
    for index,v := range m_pool.Pool{
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

func removeOfflinePlayersInPool(m_pool *datastruct.MatchingPool,removeIndex []int){
    rm_count:=0
    for index,v := range removeIndex {
        if index!=0{
           v = v-rm_count
        }
        m_pool.Pool=append(m_pool.Pool[:v], m_pool.Pool[v+1:]...)
        rm_count++;
    }
}

func getPool()*datastruct.MatchingPool{
    pools:=matchingPools
    m_pool:=pools[0]
    return m_pool
}

func checkMatchActionPool(uuid string) bool{
    tf:=false 
    matchActionPool.Mutex.RLock()
    defer  matchActionPool.Mutex.RUnlock()
    for _,v:=range matchActionPool.Pool{
        if v==uuid{
            tf = true
            break
        }
    }
    return tf
}

func PlayerIsExist(k string,a gate.Agent,uid int) bool {
	onlinePlayers.Lock.Lock()
    defer onlinePlayers.Lock.Unlock()
    v, ok := onlinePlayers.Bm[k];
	if !ok {
        user:=db.Module.GetUserInfo(uid)
        player:=createPlayer(user)
        player.Agent = a
        onlinePlayers.Bm[k]=*player
	}else{
        v.GameData.EnterType = datastruct.EmptyWay
        v.GameData.RoomId = datastruct.NULLSTRING
    }
	return ok
}

func addInMatchActionPool(p_uuid string){
    matchActionPool.Mutex.Lock()
    matchActionPool.Pool=append(matchActionPool.Pool,p_uuid)
    matchActionPool.Mutex.Unlock()
    log.Debug("addInMatchActionPool:",p_uuid)
}

func removeFromMatchActionPool(p_uuid string){
    matchActionPool.Mutex.Lock()
    defer matchActionPool.Mutex.Unlock()
    rm_index:=-1
    for index,v := range matchActionPool.Pool{
        if v==p_uuid{
            rm_index = index
            log.Debug("removeFromMatchActionPool:",v)
            break
        }
    }
    if rm_index>=0{
        matchActionPool.Pool=append(matchActionPool.Pool[:rm_index], matchActionPool.Pool[rm_index+1:]...)
    }
}