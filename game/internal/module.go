package internal

import (
	"github.com/name5566/leaf/module"
	"server/base"
	"server/game/internal/match/singleMatch"
	"server/game/internal/match/endlessModeMatch"
	"server/game/internal/match/inviteModeMatch"
	"server/datastruct"
	"github.com/name5566/leaf/gate"
)

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	ptr_singleMatch=singleMatch.NewSingleMatch()
	ptr_endlessModeMatch=endlessModeMatch.NewEndlessModeMatch()
	ptr_inviteModeMatch=inviteModeMatch.NewInviteModeMatch()
	onlinePlayersData = datastruct.CreateOnlinePlayersData()
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	
}

func (m *Module) OnDestroy() {
}

/*多次登录,踢掉关键的在线玩家*/
func (m *Module)Relogin(loginName string,a gate.Agent){
     relogin(loginName,a)
}








