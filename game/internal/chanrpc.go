package internal

import (
	"github.com/name5566/leaf/gate"
	"server/datastruct"
	"server/game/internal/match"
)

const MatchingKey="Matching"
const CloseAgentKey="CloseAgent"

func init() {
	skeleton.RegisterChanRPC(MatchingKey, rpcMatchingPlayers)
	skeleton.RegisterChanRPC(CloseAgentKey, removeOnlinePlayer)
}

func rpcMatchingPlayers(args []interface{}) {
	ptr_match := args[0].(match.ParentMatch)
	connUUID := args[1].(string)
	a := args[2].(gate.Agent)
	uid:= args[3].(int)
	ptr_match.Matching(connUUID,a,uid)
}

func removeOnlinePlayer(args []interface{}){
	a := args[0].(gate.Agent)
	u_data:=a.UserData()
	if u_data != nil{
		au_data:=u_data.(datastruct.AgentUserData)
		connUUID:=au_data.ConnUUID
		mode:=au_data.GameMode
		r_id:=au_data.Extra.RoomID
		w_id:=au_data.Extra.WaitRoomID
		playerLeftRoom(connUUID,mode,r_id)
		if w_id != datastruct.NULLSTRING{
		   leaveWaitRoom(w_id,connUUID)
		}
	}
	if a != nil{
	   a.Destroy()
	}
}
