package endlessModeMatch

import (
	"sync"
	"server/datastruct"
	"github.com/name5566/leaf/gate"
	"server/db"
	"server/tools"
	"server/msg"
	"github.com/name5566/leaf/log"
	"server/game/internal/match"
)

const LeastPeople = 10


/*无尽模式*/
type EndlessModeMatch struct {
	rooms *match.Rooms
	onlinePlayers *datastruct.OnlinePlayers
	actionPool *match.MatchActionPool
	mutex *sync.Mutex
}

func NewEndlessModeMatch()*EndlessModeMatch{
	endlessModeMatch:=new(EndlessModeMatch)
	endlessModeMatch.init()
	return endlessModeMatch
}

func (endlessModeMatch *EndlessModeMatch)init(){
	endlessModeMatch.onlinePlayers = datastruct.NewOnlinePlayers()
	endlessModeMatch.rooms = match.NewRooms()
	endlessModeMatch.actionPool = match.NewMatchActionPool(0)
	endlessModeMatch.mutex = new(sync.Mutex)
}

func (match *EndlessModeMatch)addPlayer(connUUID string,a gate.Agent,uid int){
	match.addOnlinePlayer(connUUID,a,uid)
	match.actionPool.AddInMatchActionPool(connUUID)
}

func (match *EndlessModeMatch)RemovePlayer(connUUID string){
	match.onlinePlayers.Delete(connUUID)
	match.actionPool.RemoveFromMatchActionPool(connUUID)
}

func (match *EndlessModeMatch)CheckActionPool(connUUID string) bool{
	return match.actionPool.Check(connUUID)
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
		 match.onlinePlayers.Bm[connUUID]=v
	 }
}


func (match *EndlessModeMatch)Matching(connUUID string, a gate.Agent,uid int){
	log.Debug("无尽模式匹配开始")
	match.addPlayer(connUUID,a,uid)
	
	match.mutex.Lock()
	r_id,willEnterRoom:=match.rooms.GetFreeRoomId()
	if !willEnterRoom{
	    r_id=match.createRoom(connUUID)
	}
	match.mutex.Unlock()
	
	player,tf:=match.onlinePlayers.GetAndUpdateState(connUUID,datastruct.FreeRoom,r_id)
	if tf{
		player.Agent.WriteMsg(msg.GetMatchingEndMsg(r_id))
	}
}

func (endlessModeMatch *EndlessModeMatch)createRoom(connUUID string)string{
	log.Debug("无尽模式匹配完成，创建房间")
	r_uuid:=tools.UniqueId()
	room:=match.CreateRoom(match.EndlessMode,[]string{connUUID},r_uuid,endlessModeMatch,LeastPeople,20)
    endlessModeMatch.rooms.Set(r_uuid,room)
	return r_uuid
}

func (match *EndlessModeMatch)PlayerMoved(r_id string,play_id int,moveData *msg.CS_MoveData){
	ok,room:=match.rooms.Get(r_id)
    if ok&&room.IsEnableUpdatePlayerAction(play_id){
       room.GetPlayerMovedMsg(play_id,moveData)
	}
}

func (match *EndlessModeMatch)PlayerJoin(connUUID string,joinData *msg.CS_PlayerJoinRoom){
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
		   }else{ 
			 go match.handleRoomOff(player.Agent,connUUID,player.Uid)
		   }
		}else{
            player.Agent.WriteMsg(msg.GetJoinInvalidMsg())
        }
    }
}

func (match *EndlessModeMatch)handleRoomOff(a gate.Agent,connUUID string,uid int){
    a.WriteMsg(msg.GetReMatchMsg())
    match.Matching(connUUID,a,uid)
}

func (match *EndlessModeMatch)EnergyExpended(expended int,agentUserData datastruct.AgentUserData){
       connUUID:=agentUserData.ConnUUID
	   r_id:=agentUserData.Extra.RoomID
	   ok,room:=match.rooms.Get(r_id)
	   if ok{
		room.EnergyExpended(connUUID,expended)
	   }
}

func (match *EndlessModeMatch)PlayersDied(r_id string,values []datastruct.PlayerDiedData){
	ok,room:=match.rooms.Get(r_id)
	if ok{
	  room.HandleDiedData(values)
	}
}

func (match *EndlessModeMatch)RemoveRoomWithID(uuid string){
	match.rooms.Delete(uuid)
}
func (match *EndlessModeMatch)GetOnlinePlayersPtr() *datastruct.OnlinePlayers{
     return match.onlinePlayers
}

func (match *EndlessModeMatch)PlayerLeftRoom(r_id string,connUUID string){
	match.RemovePlayer(connUUID)
	ok,room:=match.rooms.Get(r_id)
	if ok{
		room.AddPlayerleft(connUUID)
	}
}

func (match *EndlessModeMatch)PlayerRelive(r_id string,pid int,name string){
	ok,room:=match.rooms.Get(r_id)
	if ok{
	   room.HandlePlayerRelive(pid,name)
	}
}


















