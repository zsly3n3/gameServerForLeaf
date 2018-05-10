package internal
import (
	//"fmt"
	"server/db"
    "reflect"  
    "server/msg"
    "server/datastruct"
    "github.com/name5566/leaf/gate"  
    "github.com/name5566/leaf/log"
    "github.com/name5566/leaf/network/json"
    "sync"
    "server/tools"
    "time"
)
func init() {  
    //向当前模块（game 模块）注册 CS_PlayerMatching 消息 
    handler(&msg.CS_PlayerMatching{}, handlePlayerMatching)
}  
// 异步处理  
func handler(m interface{}, h interface{}) {  
    skeleton.RegisterChanRPC(reflect.TypeOf(m), h)  
}

// 消息处理  
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
    if _,tf:=onlinePlayers.Check(connUUID);tf{//已在线
        //fmt.Println("已在线")
        //removePlayer(connUUID)
        //tools.ReSetAgentUserData(a,uid)
        msgHeader.MsgName = "SC_PlayerOnline"
        a.WriteMsg(&msg.SC_PlayerOnline{
            MsgHeader:msgHeader,
        })
        return
    }

    user:=db.Module.GetUserInfo(uid)
    player:=createPlayer(user)
    player.Agent = a
    onlinePlayers.Set(connUUID,*player) //add onlinePlayer

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

    willEnterRoom:=false //是否将要加入了房间

    if !willEnterRoom{
	   pools:=*matchingPools
	   m_pool:=pools[0]
       m_pool.Mutex.Lock()
       defer m_pool.Mutex.Unlock()
	   num:=len(m_pool.Pool)
	   if num<LeastPeople{
        m_pool.Pool=append(m_pool.Pool,p_uuid)
        createTicker()
        if num == LeastPeople-1{
            //check player is online or offline
            //offline player remove from pool
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
    }

            
	
	
	 //查找相应的房间,如果没有合适的房间,创建新的房间
    // isJoined:=false
    // hall.Mutex.RLock() 
    // for _, room := range hall.Rooms {
    //     room.Mutex.RLock()
    //     if room.IsOn{
    //        //add player
    //     }
    //     room.Mutex.RUnlock()
    // }
    // hall.Mutex.RUnlock()
}

func cleanPoolAndCreateRoom(m_pool *datastruct.MatchingPool){
    stopTicker()
    arr := m_pool.Pool[:] //copy data
    m_pool.Pool=m_pool.Pool[:0]//clean pool
    go createRoom(arr)
}

func createRoom(playerUUID []string){
    //生成房间数据
    //匹配房间的权限,邀请好友的玩家进入不到

    var msgHeader json.MsgHeader
    msgHeader.MsgName = "SC_PlayerRoomData"

    var msgContent msg.SC_PlayerRoomDataContent
    msgContent.RoomID =tools.UniqueId()
  
     players:=onlinePlayers.Gets(playerUUID)
     for _,play := range players{
        play.Agent.WriteMsg(&msg.SC_PlayerRoomData{
            MsgHeader:msgHeader,
            MsgContent:msgContent,
        })
     }
}

func createPlayer(user *datastruct.User) *datastruct.Player{
    var player datastruct.Player
    player.Avatar=user.Avatar
    player.Id=user.Id
    player.NickName=user.NickName
    player.Mutex = new(sync.RWMutex)
    player.GameData = new(datastruct.PlayerGameData)
    player.GameData.StartMatchingTime = time.Now()
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
    pools:=*matchingPools
    m_pool:=pools[0]
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
    online_key:=""
    
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

