package endlessModeMatch

import (
	"server/datastruct"
	"github.com/name5566/leaf/gate"
	"server/db"
	"server/tools"
	"server/msg"
	"github.com/name5566/leaf/log"
)

const LeastPeople = 10


/*无尽模式*/
type EndlessModeMatch struct {
	rooms *Rooms
	onlinePlayers *datastruct.OnlinePlayers
}

func NewEndlessModeMatch()*EndlessModeMatch{
	endlessModeMatch:=new(EndlessModeMatch)
	endlessModeMatch.init()
	return endlessModeMatch
}

func (endlessModeMatch *EndlessModeMatch)init(){
	endlessModeMatch.onlinePlayers = datastruct.NewOnlinePlayers()
	endlessModeMatch.rooms = NewRooms()
}

func (match *EndlessModeMatch)addPlayer(connUUID string,a gate.Agent,uid int){
	match.addOnlinePlayer(connUUID,a,uid)
}

func (match *EndlessModeMatch)RemovePlayer(connUUID string){
	match.onlinePlayers.Delete(connUUID)
}

func (match *EndlessModeMatch)addOnlinePlayer(connUUID string,a gate.Agent,uid int){
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
	 }
}


func (match *EndlessModeMatch)Matching(connUUID string, a gate.Agent,uid int){
	
	match.addPlayer(connUUID,a,uid)
	
	//lock
	r_id,willEnterRoom:=match.rooms.GetFreeRoomId()
	if !willEnterRoom{
	    match.createRoom()
	}
	//unlock
	
	if willEnterRoom{
		player,tf:=match.onlinePlayers.GetAndUpdateState(connUUID,datastruct.FreeRoom,r_id)
		if tf{
			player.Agent.WriteMsg(msg.GetMatchingEndMsg(r_id))
		}
	}
}

func (match *EndlessModeMatch)createRoom(playerUUID []string){
    r_uuid:=tools.UniqueId()
    players:=match.onlinePlayers.GetsAndUpdateState(playerUUID,datastruct.FromMatchingPool,r_uuid)
    room:=CreateRoom(playerUUID,r_uuid,match)
    match.rooms.Set(r_uuid,room)
    for _,play := range players{
        play.Agent.WriteMsg(msg.GetMatchingEndMsg(r_uuid))
    }
}



func (match *SingleMatch)removeRoomWithID(uuid string){
	match.rooms.Delete(uuid)
}

func (match *SingleMatch)PlayerMoved(r_id string,play_id int,moveData *msg.CS_MoveData){
	ok,room:=match.rooms.Get(r_id)
    if ok&&room.IsEnableUpdatePlayerAction(play_id){
       room.GetPlayerMovedMsg(play_id,moveData)
    }
}


func (match *SingleMatch)PlayerJoin(connUUID string,joinData *msg.CS_PlayerJoinRoom){
	player,tf:=match.onlinePlayers.CheckAndCleanState(connUUID,datastruct.NULLWay,datastruct.NULLSTRING)
    if tf{
        r_id := joinData.MsgContent.RoomID
        if player.GameData.EnterType == datastruct.FreeRoom&&player.GameData.RoomId==r_id{
		   ok,room:=match.rooms.Get(r_id)
		   if ok{
			isOn:=room.Join(connUUID,player,false)
			if isOn{
			   log.Debug("通过遍历空闲房间进入")
			   match.actionPool.RemoveFromMatchActionPool(connUUID)
			}else{
			   go match.handleRoomOff(player.Agent,connUUID,player.Uid)
			}
		   }
          
        }else if player.GameData.EnterType == datastruct.FromMatchingPool{
			ok,room:=match.rooms.Get(r_id)
			if ok{
				for _,v:=range room.unlockedData.AllowList{
					if v == connUUID{
						log.Debug("通过匹配池进入")
						room.Join(connUUID,player,true)
						match.actionPool.RemoveFromMatchActionPool(connUUID)
						break
					}
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
	   r_id:=agentUserData.RoomID
	   ok,room:=match.rooms.Get(r_id)
	   if ok{
		room.EnergyExpended(connUUID,expended)
	   }
}

func (match *SingleMatch)PlayersDied(r_id string,values []msg.PlayerDiedData){
	ok,room:=match.rooms.Get(r_id)
	if ok{
	  room.diedData.Add(values,room)
	}
}



















