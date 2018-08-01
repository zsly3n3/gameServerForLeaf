package inviteModeMatch

import (
	"fmt"
	"server/tools"
	"server/datastruct"
	"sync"
	"github.com/name5566/leaf/gate"
	"server/msg"
	"server/thirdParty"
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
	waitRoom.roomID = tools.UniqueIdFromInt()
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
	data.Avatar = u_data.Extra.Avatar
	data.NickName = u_data.Extra.PlayName
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
			v.agent.WriteMsg(msg.GetInWaitRoomMsg(datastruct.NotFull,waitRoom.roomID,v.data.IsMaster,players))
		}
		waitRoom.mutex.Unlock()
	}else{
	   waitRoom.mutex.Unlock()	
       a.WriteMsg(msg.GetInWaitRoomStateMsg(datastruct.Full,waitRoom.roomID))
	}
}

func (waitRoom *WaitRoom)Left(connUUID string,waitRooms *WaitRooms) bool{
	isExist:= true
	waitRoom.mutex.Lock()
	rm_key:=-1
	isMaster:=-1
	rm_nickname:=""
	for k,p_data:= range waitRoom.players{
		u_data:=p_data.agent.UserData().(datastruct.AgentUserData)
		if u_data.ConnUUID == connUUID{
			rm_key = k
			isMaster = p_data.data.IsMaster
			rm_nickname = p_data.data.NickName
			break
		}
	}
    if rm_key == -1{
		waitRoom.mutex.Unlock()
		return false
	}
	length:=len(waitRoom.players)
	if length <= 1{
		deleteWaitRoom(waitRooms,waitRoom.roomID)
	}else{
		delete(waitRoom.players,rm_key)
		if isMaster == 1{
		  waitRoom.updateMasterSeat()
		}
		players:=waitRoom.getPlayersData()
		notice_str := rm_nickname + leaveStr
		for _,v := range waitRoom.players{
		  v.agent.WriteMsg(msg.GetInWaitRoomMsg(datastruct.NotFull,waitRoom.roomID,v.data.IsMaster,players))
		  v.agent.WriteMsg(msg.GetNotifyMsg(notice_str))
		}	
	}
	waitRoom.mutex.Unlock()
	return isExist
}

func (waitRoom *WaitRoom)FirePlayer(seat int,waitRooms *WaitRooms) (bool,string){
	isExist:=true
	rm_connUUID:=datastruct.NULLSTRING
	waitRoom.mutex.Lock()
    v,tf:= waitRoom.players[seat]
	if !tf{
	    waitRoom.mutex.Unlock()
        return false,rm_connUUID
	}
	u_data:=v.agent.UserData().(datastruct.AgentUserData)
	rm_connUUID = u_data.ConnUUID
	length:=len(waitRoom.players)
	if length <= 1{
		deleteWaitRoom(waitRooms,waitRoom.roomID)
	}else{
		v.agent.WriteMsg(msg.GetIsFiredMsg())
		delete(waitRoom.players,seat)
		players:=waitRoom.getPlayersData()
		notice_str := v.data.NickName + leaveStr
		for _,v := range waitRoom.players{
		  v.agent.WriteMsg(msg.GetInWaitRoomMsg(datastruct.NotFull,waitRoom.roomID,v.data.IsMaster,players))
		  v.agent.WriteMsg(msg.GetNotifyMsg(notice_str))
		}
	}
	waitRoom.mutex.Unlock()
	return isExist,rm_connUUID
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

func (waitRoom *WaitRoom)IfCanStartGame(connUUID string) bool {
	tf := false
	waitRoom.mutex.RLock()
	defer waitRoom.mutex.RUnlock()
	if len(waitRoom.players) > 1{
	  for _,v := range waitRoom.players{
		u_data:=v.agent.UserData().(datastruct.AgentUserData)
		if u_data.ConnUUID == connUUID{
			if v.data.IsMaster == 1 {
			   tf = true
			}
			break
		}
	  }
	}
	return tf
}

func (waitRoom *WaitRoom)IsPermit(connUUID string,seat int)bool{
	tf:=false
	waitRoom.mutex.RLock()
	defer waitRoom.mutex.RUnlock() 
	for _,v := range waitRoom.players{
		u_data:=v.agent.UserData().(datastruct.AgentUserData)
		if u_data.ConnUUID == connUUID{
		   if v.data.IsMaster == 1 && v.data.Seat != seat {
			  tf = true
		   }
		   break
		}
	}
	return tf
}

func (waitRoom *WaitRoom)SendInviteQRCode(qrcode string){
	fmt.Println("qrcode:",qrcode)
	for _,v := range waitRoom.players{
		v.agent.WriteMsg(msg.GetInviteQRCodeMsg(qrcode))
	}
}

func deleteWaitRoom(waitRooms *WaitRooms,w_id string){
	waitRooms.Delete(w_id)
	deleteQRCode(w_id)
}

func deleteQRCode(w_id string){
	go thirdParty.RemoveQRCode(w_id)
}