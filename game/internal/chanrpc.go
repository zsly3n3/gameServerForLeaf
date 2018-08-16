package internal

import (
	"github.com/name5566/leaf/gate"
	"server/datastruct"
	"server/game/internal/match"
	"server/game/internal/match/inviteModeMatch"
	"server/thirdParty"
	"github.com/name5566/leaf/log"
	"server/tools"
)

const MatchingKey="Matching"
const GetInviteQRCodeKey="GetInviteQRCode"
const CloseAgentKey="CloseAgent"

func init() {
	skeleton.RegisterChanRPC(MatchingKey, rpcMatchingPlayers)
	skeleton.RegisterChanRPC(GetInviteQRCodeKey, rpcGetQRCode)
	skeleton.RegisterChanRPC(CloseAgentKey, removeOnlinePlayer)
}

func rpcMatchingPlayers(args []interface{}) {
	ptr_match := args[0].(match.ParentMatch)
	connUUID := args[1].(string)
	a := args[2].(gate.Agent)
	uid:= args[3].(int)
	r_id := ptr_match.Matching(connUUID,a,uid)
	switch ptr_match.(type){
	 case *inviteModeMatch.InviteModeMatch:
	    ChanRPC.Go(GetInviteQRCodeKey,r_id)
	}
}

func rpcGetQRCode(args []interface{}){
	r_id := args[0].(string)
	rs:=thirdParty.GetQRCode(r_id)
    sendInviteQRCode(r_id,rs)
}

func removeOnlinePlayer(args []interface{}){
	a := args[0].(gate.Agent)
	u_data:=tools.GetUserData(a)
	if u_data != nil{
		au_data:=u_data
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
	   a.Close()
	   a.Destroy()
	   log.Debug("client Close & Destroy")
	}
}
