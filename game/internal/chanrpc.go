package internal

import (
	"github.com/name5566/leaf/gate"
	"server/datastruct"
	//"server/slice"
)

func init() {
	skeleton.RegisterChanRPC("MatchingPlayers", matchingPlayers)
	skeleton.RegisterChanRPC("CloseAgent", removeOnlinePlayer)
}


func matchingPlayers(args []interface{}) {
	p_id := args[0].(int)
	pools:=*matchingPools
	isAppend:= false
	for _, m_pool := range pools {
		m_pool.Mutex.Lock()
		if len(m_pool.Pool)<Pool_Capacity{
			m_pool.Pool=append(m_pool.Pool,p_id)
			isAppend = true
		}
		m_pool.Mutex.Unlock()
		if isAppend{
		   break
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


func removeOnlinePlayer(args []interface{}){
	a := args[0].(gate.Agent)
	index,player:=onlinePlayers.GetWithAddr(a.RemoteAddr().String())
	onlinePlayers.Delete(index)
	if index > 0{
			player.Mutex.RLock()
			switch player.LocationStatus {
			case datastruct.Matching:
				 println("remove from Matching")
				 
			case datastruct.Playing:
				 println("remove from room")
			}
			player.Mutex.RUnlock()
	}
}

func removeFromMatchingPool(){
	 
	//slice.Remove
}
