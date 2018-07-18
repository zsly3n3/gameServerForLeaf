package internal
import (  
	"fmt"
    "reflect"  
    "server/datastruct"
    "server/msg"
    "github.com/go-xorm/xorm"
    "server/conf"
    "github.com/name5566/leaf/log"
    "server/tools"
    //"encoding/json"//json封装解析
)

func init() {
    // 向当前模块（game 模块）注册 Test 消息的消息处理函数 handleTest  
    //handler(&msg.Test{}, handleTest)
}

// 异步处理  
func handler(m interface{}, h interface{}) {  
    skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handleUserLogin(arg interface{}) int {
    data := arg.(*msg.CS_UserLogin)
    rs_id:=-1
    user := &datastruct.User{LoginName:data.MsgContent.LoginName}
    has, _ := dbEngine.Get(user)
    if !has{
        user.Avatar=data.MsgContent.Avatar
        user.NickName=data.MsgContent.NickName
        _,err:=dbEngine.Insert(user)
        if err!= nil{
           user.Id = rs_id
           log.Debug("Insert user error:%v",err.Error())
        }
    }
    rs_id=user.Id
    return rs_id
}

func handleCreateDB()*xorm.Engine{
    dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",conf.Server.DB_UserName,conf.Server.DB_Pwd,conf.Server.DB_IP,conf.Server.DB_Name)
	engine, err:= xorm.NewEngine("mysql", dsn)
	errhandle(err)
	err=engine.Ping()
	errhandle(err)
	//日志打印SQL
    engine.ShowSQL(true)
	//设置连接池的空闲数大小
	engine.SetMaxIdleConns(10)
    resetDB(engine)
    initData(engine)
    dbEngine = engine
    return engine
}

func resetDB(engine *xorm.Engine){
    user:=&datastruct.User{}
    robotName:=&datastruct.RobotName{}
	err:=engine.DropTables(user,robotName)
    errhandle(err)
	err=engine.CreateTables(user,robotName)
    errhandle(err)
    names:=tools.GetRobotNames()
    _, err = engine.Insert(&names)
    errhandle(err)
}

func initData(engine *xorm.Engine){
    robotName:=&datastruct.RobotName{}
    robotName.State = 0
    engine.Cols("state").Update(robotName)
}

func errhandle(err error){
	if err != nil {
		log.Fatal("db error is %v", err.Error())
	}
}

func handleGetUserInfo(uid int)*datastruct.User{
    var user datastruct.User
    _,err:=dbEngine.Id(uid).Get(&user)
    if err !=nil{
        log.Error("handleGetUserInfo err:%v",err)
    }
    return &user
}

func getRobotNames(num int,engine *xorm.Engine)([]datastruct.RobotName,int){
    session := engine.NewSession()
    defer session.Close()
    err := session.Begin()
    names:= make([]datastruct.RobotName,0,num)
    table:=new(datastruct.RobotName)
    count, _ := session.Where("id >?", 0).Count(table)
    err = session.Where("state = ?", 0).Limit(num, 0).Find(&names)
    if err != nil {
       return nil,int(count)
    }
    if len(names) > 0{
       sql := "update robot_name set state=1 where id in ("
       for _,v := range names{
        sql+= fmt.Sprintf("%d",v.Id) + ","
       }
       sql=sql[0:len(sql)-1]
       sql +=")"
       _,err = session.Exec(sql)
       if err != nil {
        session.Rollback()
        return nil,int(count)
       }
       err = session.Commit()
       if err != nil {
           return nil,int(count)
       }
    }
    return names,int(count)
}

func handleGetRobotNames(num int,engine *xorm.Engine)map[int]string{
     names,count:=getRobotNames(num,engine)
     length:=len(names)
     rs:=num-length
     if rs > 0 {
        for i:=0;i<rs;i++{
            rand:=tools.GetRandID(names,count)
            var name datastruct.RobotName
            engine.Id(rand).Get(&name)
            names = append(names,name)
        }
     }
     rsMap:=make(map[int]string)//key:id,value:name
     for _,v := range names{
         rsMap[v.Id] = v.Name
     }
     return rsMap
}
// func handleGetRobotName(engine *xorm.Engine)(int,string){
//     names,count:=getRobotNames(1,engine)
//     if len(names) == 0 {
//        rand:=tools.GetOnceRandID(count)
//        var name datastruct.RobotName
//        engine.Id(rand).Get(&name)
//        return name.Id,name.Name
//     }
//     return names[0].Id,names[0].Name
// }

func handleUpdateRobotNamesState(names map[int]string,engine *xorm.Engine){
    session := engine.NewSession()
    defer session.Close()
    session.Begin()
    sql := "update robot_name set state=0 where id in ("
    for k,_ := range names{
        sql+= fmt.Sprintf("%d",k) + ","
    }
    sql=sql[0:len(sql)-1]
    sql +=")"
    _,err := session.Exec(sql)
    if err != nil {
        session.Rollback()
        return
    }
    session.Commit()
}
// func handleUpdateRobotNameState(n_id int,engine *xorm.Engine){
//     session := engine.NewSession()
//     defer session.Close()
//     session.Begin()
//     sql := "update robot_name set state=0 where id="
//     sql+= fmt.Sprintf("%d",n_id)
//     _,err := session.Exec(sql)
//     if err != nil {
//         session.Rollback()
//         return
//     }
//     session.Commit()
// }

