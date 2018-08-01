package inviteModeMatch

import (
	"server/datastruct"
	"github.com/name5566/leaf/gate"
	"server/db"
	"server/msg"
	"server/tools"
	"server/game/internal/match"
	"github.com/name5566/leaf/log"
)

const MaxPeople = 10

type InviteModeMatch struct {
	rooms *match.Rooms
	waitRooms *WaitRooms
	onlinePlayers *datastruct.OnlinePlayers
	actionPool *match.MatchActionPool
}

func NewInviteModeMatch()*InviteModeMatch{
	inviteModeMatch:=new(InviteModeMatch)
	inviteModeMatch.init()
	return inviteModeMatch
}

func (inviteModeMatch *InviteModeMatch)init(){
	inviteModeMatch.onlinePlayers = datastruct.NewOnlinePlayers()
	inviteModeMatch.rooms = match.NewRooms()
	inviteModeMatch.waitRooms = NewWaitRooms()
	inviteModeMatch.actionPool = match.NewMatchActionPool(0)
}

func (match *InviteModeMatch)addPlayer(connUUID string,a gate.Agent,uid int) {
	  match.addOnlinePlayer(connUUID,a,uid)
	  match.actionPool.AddInMatchActionPool(connUUID)
}

func (match *InviteModeMatch)RemovePlayer(connUUID string){
	match.onlinePlayers.Delete(connUUID)
	match.actionPool.RemoveFromMatchActionPool(connUUID)
}

func (match *InviteModeMatch)CheckActionPool(connUUID string) bool{
	return match.actionPool.Check(connUUID)
}


func (match *InviteModeMatch)addOnlinePlayer(connUUID string,a gate.Agent,uid int) {
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


func (match *InviteModeMatch)Matching(connUUID string,a gate.Agent,uid int)string{
	match.addPlayer(connUUID,a,uid)

	//创建等待房间,房主即第一个玩家
	waitRoom:=CreateWaitRoom(a,MaxPeople)
	r_id:=waitRoom.roomID
	match.waitRooms.Set(r_id,waitRoom)
	
	userData:=a.UserData().(datastruct.AgentUserData)
	var data datastruct.PlayerInWaitRoom
	data.NickName = userData.Extra.PlayName
	data.Avatar = userData.Extra.Avatar
	data.IsMaster = 1
	data.Seat = 0
	players:=[]datastruct.PlayerInWaitRoom{data}
	
	var extra datastruct.ExtraUserData
	extra.Avatar = userData.Extra.Avatar
	extra.PlayName = userData.Extra.PlayName
	extra.RoomID = userData.Extra.RoomID
	extra.WaitRoomID = r_id
	tools.ReSetAgentUserData(uid,datastruct.InviteMode,userData.PlayId,a,connUUID,extra)
	a.WriteMsg(msg.GetInWaitRoomMsg(datastruct.NotFull,r_id,1,players))
	return r_id
}

func (match *InviteModeMatch)JoinWaitRoom(w_id string,a gate.Agent,uid int,connUUID string){
	 tf,waitRoom:=match.waitRooms.Get(w_id)
	 if tf{
		userData:=a.UserData().(datastruct.AgentUserData)
		var extra datastruct.ExtraUserData
		extra.Avatar = userData.Extra.Avatar
		extra.PlayName = userData.Extra.PlayName
		extra.RoomID = userData.Extra.RoomID
		extra.WaitRoomID = w_id
		tools.ReSetAgentUserData(uid,datastruct.InviteMode,userData.PlayId,a,connUUID,extra)
		match.addPlayer(connUUID,a,uid)
		waitRoom.Join(a)
	 }else{
		a.WriteMsg(msg.GetInWaitRoomStateMsg(datastruct.NotExist,w_id)) 
	 }
}

func (match *InviteModeMatch)LeftWaitRoom(w_id string,connUUID string){
	tf,waitRoom:=match.waitRooms.Get(w_id)
	if tf {
	   isExist:=waitRoom.Left(connUUID,match.waitRooms)
	   if isExist{
		  match.RemovePlayer(connUUID)
	   }
	}
}

func (match *InviteModeMatch)MasterFirePlayer(w_id string,connUUID string,seat int){
	tf,waitRoom:=match.waitRooms.Get(w_id)
	if tf && waitRoom.IsPermit(connUUID,seat){
		isExist,rm_connUUID:=waitRoom.FirePlayer(seat,match.waitRooms)
		if isExist{
		   match.RemovePlayer(rm_connUUID)
		}
	}
}

func (match *InviteModeMatch)RemoveRoomWithID(uuid string){
	match.rooms.Delete(uuid)
}
func (match *InviteModeMatch)GetOnlinePlayersPtr() *datastruct.OnlinePlayers{
     return match.onlinePlayers
}

func (match *InviteModeMatch)PlayersDied(r_id string,values []datastruct.PlayerDiedData){
	ok,room:=match.rooms.Get(r_id)
	if ok{
	  room.HandleDiedData(values)
	}
}

func (match *InviteModeMatch)PlayerLeftRoom(r_id string,connUUID string){
	match.RemovePlayer(connUUID)
}

func (match *InviteModeMatch)EnergyExpended(expended int,agentUserData datastruct.AgentUserData){
	connUUID:=agentUserData.ConnUUID
	r_id:=agentUserData.Extra.RoomID
	ok,room:=match.rooms.Get(r_id)
	if ok{
	 room.EnergyExpended(connUUID,expended)
	}
}

func (match *InviteModeMatch)PlayerMoved(r_id string,play_id int,moveData *msg.CS_MoveData){
	ok,room:=match.rooms.Get(r_id)
    if ok&&room.IsEnableUpdatePlayerAction(play_id){
       room.GetPlayerMovedMsg(play_id,moveData)
	}
}

func (match *InviteModeMatch)PlayerJoin(connUUID string,joinData *msg.CS_PlayerJoinRoom){
	player,tf:=match.onlinePlayers.CheckAndCleanState(connUUID,datastruct.NULLWay,datastruct.NULLSTRING)
    if tf{
	   r_id := joinData.MsgContent.RoomID
        if player.GameData.EnterType == datastruct.BeInvited&&player.GameData.RoomId==r_id{
			ok,room:=match.rooms.Get(r_id)
			if ok{
				isExist:=false
				for _,v:=range room.GetAllowList(){
					if v == connUUID{
						isOn:=room.Join(connUUID,player,true)
						if isOn{
						  log.Debug("通过邀请房间进入")
						  match.actionPool.RemoveFromMatchActionPool(connUUID)	
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

func (match *InviteModeMatch)StartGame(w_id string,connUUID string){
	tf,waitRoom:=match.waitRooms.Get(w_id)
	if tf && waitRoom.IfCanStartGame(connUUID){
		deleteWaitRoom(match.waitRooms,w_id)
		waitRoom.mutex.Lock()
		defer waitRoom.mutex.Unlock()
		playersUUID := waitRoom.GetPlayersUUID()
		r_id:=match.createRoom(playersUUID)
		waitRoom.SendMatchingEndMsg(msg.GetMatchingEndMsg(r_id),match.onlinePlayers,r_id)
	}
}

func (inviteModeMatch *InviteModeMatch)createRoom(playersUUID []string)string{
	log.Debug("邀请模式匹配完成，创建房间")
	r_uuid:=tools.UniqueId()
	room:=match.CreateRoom(match.Invite,playersUUID,r_uuid,inviteModeMatch,MaxPeople,MaxPeople)
    inviteModeMatch.rooms.Set(r_uuid,room)
	return r_uuid
}

func (match *InviteModeMatch)SendInviteQRCode(w_id string,qrcode string){
	tf,waitRoom:=match.waitRooms.Get(w_id)
	if tf{
	   waitRoom.SendInviteQRCode(qrcode)
	}
}











