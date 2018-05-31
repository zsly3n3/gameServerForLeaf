package internal

import (
	"github.com/name5566/leaf/module"
	"server/base"
    "server/game/internal/match/singleMatch"
)


var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	//singleMatch = singleMatch.NewSingleMatch()
	ptr_singleMatch=singleMatch.NewSingleMatch()
	
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	
}

func (m *Module) OnDestroy() {
}










