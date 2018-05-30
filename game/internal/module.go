package internal

import (
	"github.com/name5566/leaf/module"
	"server/base"
    "server/game/internal/Matching"
)


var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	singleMatch = Matching.NewSingleMatch()
	
	rooms *Rooms
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	rooms = createRooms()
}

func (m *Module) OnDestroy() {
}


func createRooms()*Rooms{
    rms:=NewRooms()
	return rms
}











