package internal

import (
	"github.com/name5566/leaf/gate"
	"server/datastruct"
)

const SinglePersonMatchingKey="SinglePersonMatching"
const CloseAgentKey="CloseAgent"


func init() {
	skeleton.RegisterChanRPC(SinglePersonMatchingKey, rpcMatchingPlayers)
	skeleton.RegisterChanRPC(CloseAgentKey, removeOnlinePlayer)
}


func rpcMatchingPlayers(args []interface{}) {
	p_uuid := args[0].(string)
	a := args[1].(gate.Agent)
	uid:= args[2].(int)
	singlePersonMatchingPlayers(p_uuid,a,uid)
}


func removeOnlinePlayer(args []interface{}){
	a := args[0].(gate.Agent)
	u_data:=a.UserData()
	if u_data != nil{
		au_data:=u_data.(datastruct.AgentUserData)
		connUUID:=au_data.ConnUUID
		removePlayer(connUUID,au_data.GameMode)
	}
}
