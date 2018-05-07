package internal

import (
	"server/datastruct"
	"github.com/name5566/leaf/module"
	"server/base"
	"sync"
)

const Pool_Num int =10
const Pool_Capacity int =30

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	hall  *datastruct.Hall
	onlinePlayers  *datastruct.OnlinePlayers //在线玩家统计
	matchingPools *[Pool_Num]*datastruct.MatchingPool
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	hall = createHall()
	onlinePlayers=createOnlinePlayers()
	matchingPools=createMatchingPools()
}

func (m *Module) OnDestroy() {

}


func createHall()*datastruct.Hall{
    new_hall:= new(datastruct.Hall)
    new_hall.Mutex=new(sync.RWMutex)
    new_hall.Rooms=make([]*datastruct.Room,0)    
    return new_hall
}
func createOnlinePlayers()*datastruct.OnlinePlayers{
    op:=datastruct.NewOnlinePlayers()
	return op
}

func createMatchingPools()*[Pool_Num]*datastruct.MatchingPool{
	var balannew_poolsce [Pool_Num]*datastruct.MatchingPool
	for i := 0; i < Pool_Num; i++ {
		balannew_poolsce[i]=createMatchingPool()
	}
    return &balannew_poolsce
}

func createMatchingPool()*datastruct.MatchingPool{
	new_pool:= new(datastruct.MatchingPool)
	new_pool.Mutex=new(sync.RWMutex)
	new_pool.Pool=make([]int,0,Pool_Capacity)
	return new_pool
}





