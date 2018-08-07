package internal

import (
	"server/msg"
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
	robotPaths []map[int]msg.Point
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
// func (m *Module) GetRobotName()(int,string){
// 	return handleGetRobotName(dbEngine)
// }
func (m *Module) UpdateRobotNamesState(names map[int]string){
	 handleUpdateRobotNamesState(names,dbEngine)
}

func (m *Module) GetRobotPaths()[]map[int]msg.Point{
	 return robotPaths
}

func (m *Module) GetMaxScoreInEndlessMode(uid int)(int,int){
	return handleGetMaxScoreInEndlessMode(uid)
}

func (m *Module) UpdateMaxScoreInEndlessMode(uid int,score int,killNum int){
	handleUpdateMaxScoreInEndlessMode(uid,score,killNum)
}

func (m *Module) GetMaxScoreInSinglePersonMode(uid int)(int,int){
	return handleGetMaxScoreInSinglePersonMode(uid)
}

func (m *Module) UpdateMaxScoreInSinglePersonMode(uid int,score int,killNum int){
	handleUpdateMaxScoreInSinglePersonMode(uid,score,killNum)
}

func (m *Module) GetMaxScoreInviteMode(uid int)(int,int){
 	return handleGetMaxScoreInviteMode(uid)
}

func (m *Module) UpdateMaxScoreInviteMode(uid int,score int,killNum int){
	handleUpdateMaxScoreInviteMode(uid,score,killNum)
}

func (m *Module) AddFragmentNum(uid int,fragmentNum int){
	hanleAddFragmentNum(uid,fragmentNum)
}

func (m *Module) AddGameIntegral(uid int,integral int){
	hanleAddGameIntegral(uid,integral)
}

func (m *Module) GetSnakeLengthRank(uid int,mode datastruct.GameModeType,rankStart int,rankEnd int)(*datastruct.PlayerScore,[]datastruct.PlayerRankScoreData){
	return hanleLengthRank(uid,mode,rankStart,rankEnd)
}

func (m *Module) GetKillNumRank(uid int,mode datastruct.GameModeType,rankStart int,rankEnd int)(*datastruct.PlayerKillNum,[]datastruct.PlayerRankKillNumData){
	return hanleKillNumRank(uid,mode,rankStart,rankEnd)
}





