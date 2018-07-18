package internal

import (
	"github.com/name5566/leaf/module"
	"server/base"
	"server/game/internal/match/singleMatch"
	"server/game/internal/match/endlessModeMatch"
	"server/game/internal/match/inviteModeMatch"
)

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	ptr_singleMatch=singleMatch.NewSingleMatch()
	ptr_endlessModeMatch=endlessModeMatch.NewEndlessModeMatch()
	ptr_inviteModeMatch=inviteModeMatch.NewInviteModeMatch()
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	
}

func (m *Module) OnDestroy() {
}










