package internal

import (
	//"fmt"
	"github.com/name5566/leaf/gate"
	"server/datastruct"
)

func init() {
	skeleton.RegisterChanRPC("MatchingPlayers", rpcMatchingPlayers)
	skeleton.RegisterChanRPC("CloseAgent", removeOnlinePlayer)
}


func rpcMatchingPlayers(args []interface{}) {
	p_uuid := args[0].(string)
	matchingPlayers(p_uuid)
}


func removeOnlinePlayer(args []interface{}){
	a := args[0].(gate.Agent)
	u_data:=a.UserData()
	if u_data != nil{
		au_data:=u_data.(datastruct.AgentUserData)
		removePlayer(au_data.ConnUUID)
	}
}

func removeFromMatchingPool(){
	 
	//slice.Remove
}
