package internal

import (
	"server/datastruct"
	"github.com/name5566/leaf/module"
	"server/base"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

var (
	skeleton = base.NewSkeleton()
	ChanRPC  = skeleton.ChanRPCServer
	dbEngine *xorm.Engine
)

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
	dbEngine = handleCreateDB()

}

func (m *Module) OnDestroy() {
	dbEngine.Close()
}


func (m *Module) GetUserInfo (uid int)*datastruct.User{
	 return handleGetUserInfo(uid)
}


func (m *Module) UserLogin (arg interface{}) int{
	return handleUserLogin(arg)
}

func (m *Module) GetRobotNames(num int)map[int]string{
	return handleGetRobotNames(num,dbEngine)
}
func (m *Module) GetRobotName()(int,string){
	return handleGetRobotName(dbEngine)
}
func (m *Module) UpdateRobotNamesState(names map[int]string){
	 handleUpdateRobotNamesState(names,dbEngine)
}
func (m *Module) UpdateRobotNameState(n_id int){
	handleUpdateRobotNameState(n_id,dbEngine)
}


