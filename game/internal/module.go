package internal

import (
	"github.com/name5566/leaf/module"
	"server/base"
	"server/game/internal/match/singleMatch"
	"server/game/internal/match/endlessModeMatch"
)


var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	ptr_singleMatch=singleMatch.NewSingleMatch()
	ptr_endlessModeMatch=endlessModeMatch.NewEndlessModeMatch()
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	
}

func (m *Module) OnDestroy() {
}










