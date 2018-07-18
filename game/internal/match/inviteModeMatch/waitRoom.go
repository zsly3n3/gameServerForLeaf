package inviteModeMatch

import (
	"server/tools"
	"server/datastruct"
	"sync"
	"github.com/name5566/leaf/gate"
	"server/msg"
)

const leaveStr = " 退出房间"

type WaitRoom struct {
	mutex *sync.RWMutex //读写互斥量
	roomID string //房间id
	maxPeople int //房间人数
	players map[int]*playerMsgData //根据座位号查询玩家
}

type playerMsgData struct{
	 data  datastruct.PlayerInWaitRoom
	 agent gate.Agent
}

func CreateWaitRoom(a gate.Agent,maxPeople int) *WaitRoom{
	waitRoom:=new(WaitRoom)
	waitRoom.mutex = new(sync.RWMutex)
	waitRoom.roomID = tools.UniqueId()
	waitRoom.maxPeople = maxPeople
    waitRoom.players = make(map[int]*playerMsgData)
    waitRoom.addPlayer(a,1)
	return waitRoom
}

func (waitRoom *WaitRoom)addPlayer(a gate.Agent,isMaster int){
	u_data:=a.UserData().(datastruct.AgentUserData)
	seat:=waitRoom.getSeatNumber()
	var data datastruct.PlayerInWaitRoom
	data.Seat = seat
	data.IsMaster = isMaster
	data.Avatar = u_data.Avatar
	data.NickName = u_data.PlayName
	p_data:=new(playerMsgData)
	p_data.agent = a
	p_data.data = data
	waitRoom.players[seat] = p_data
}

func (waitRoom *WaitRoom)Join(a gate.Agent){
	waitRoom.mutex.Lock()
	length:=len(waitRoom.players)
	if length < waitRoom.maxPeople {
		waitRoom.addPlayer(a,0)
		players:=waitRoom.getPlayersData()
		for _,v := range waitRoom.players{
			v.agent.WriteMsg(msg.GetInWaitRoomMsg(datastruct.NotFull,waitRoom.roomID,players))
		}
		waitRoom.mutex.Unlock()
	}else{
	   waitRoom.mutex.Unlock()	
       a.WriteMsg(msg.GetInWaitRoomStateMsg(datastruct.Full,waitRoom.roomID))
	}
}

func (waitRoom *WaitRoom)Left(connUUID string,waitRooms *WaitRooms,isFired bool){
	waitRoom.mutex.Lock()
	length:=len(waitRoom.players)
	if length <= 1{
		waitRooms.Delete(waitRoom.roomID)
		waitRoom.mutex.Unlock()
	}else{
		rm_key:=-1
		isMaster:=-1
		rm_nickname:=""
		for k,p_data:= range waitRoom.players{
			u_data:=p_data.agent.UserData().(datastruct.AgentUserData)
			if u_data.ConnUUID == connUUID{
				rm_key = k
				isMaster = p_data.data.IsMaster
				rm_nickname = p_data.data.NickName
				if isFired{
				   p_data.agent.WriteMsg(msg.GetIsFiredMsg())
				}
				break
			}
		}
		delete(waitRoom.players,rm_key)
		if isMaster == 1{
			waitRoom.updateMasterSeat()
		}
		players:=waitRoom.getPlayersData()
		notice_str := rm_nickname + leaveStr
		for _,v := range waitRoom.players{
			v.agent.WriteMsg(msg.GetInWaitRoomMsg(datastruct.NotFull,waitRoom.roomID,players))
			v.agent.WriteMsg(msg.GetNotifyMsg(notice_str))
		}
		waitRoom.mutex.Unlock()		
	}
}

func (waitRoom *WaitRoom)getPlayersData()[]datastruct.PlayerInWaitRoom{
	players:=make([]datastruct.PlayerInWaitRoom,0,len(waitRoom.players))
	for _,v := range waitRoom.players{
		players = append(players,v.data)
	}
	return players
}

func (waitRoom *WaitRoom)getSeatNumber() int{
	 rs:=-1
	 for i:=0;i<waitRoom.maxPeople;i++{
		 _,tf:=waitRoom.players[i]
         if !tf{
			 rs = i
			 break
		 }
	 }
	 return rs
}

func (waitRoom *WaitRoom)updateMasterSeat(){
	 for i:=0;i<waitRoom.maxPeople;i++{
		 v,tf:=waitRoom.players[i]
		 if tf{
			v.data.IsMaster = 1
			break
		 }
	 }
}

func (waitRoom *WaitRoom)GetPlayersUUID()[]string{
	 length:=len(waitRoom.players)
	 rs:=make([]string,0,length)
	 for _,p_data:= range waitRoom.players{
		u_data:=p_data.agent.UserData().(datastruct.AgentUserData)
		rs = append(rs,u_data.ConnUUID)
	 }
	 return rs
}

func (waitRoom *WaitRoom)SendMatchingEndMsg(msg *msg.SC_PlayerMatchingEnd,onlinePlayers *datastruct.OnlinePlayers,r_id string){
	for _,v := range waitRoom.players{
		u_data:=v.agent.UserData().(datastruct.AgentUserData)
		player,tf:=onlinePlayers.GetAndUpdateState(u_data.ConnUUID,datastruct.BeInvited,r_id)
		if tf {
			player.Agent.WriteMsg(msg)
		}
	}
}
