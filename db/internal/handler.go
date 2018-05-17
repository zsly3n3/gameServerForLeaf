package internal
import (  
	"fmt"
    "reflect"  
    "server/datastruct"
    "server/msg"
    "github.com/go-xorm/xorm"
    "server/conf"
    "github.com/name5566/leaf/log"
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
	
	user:=&datastruct.User{}
	err=engine.DropTables(user)
    errhandle(err)
    
	err=engine.CreateTables(user)
    errhandle(err)
    
    dbEngine = engine
    return engine
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



