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
    "strconv"
    "time"
    "github.com/robfig/cron"
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
    dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4",conf.Server.DB_UserName,conf.Server.DB_Pwd,conf.Server.DB_IP,conf.Server.DB_Name)
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
    insertTestData(engine)
    dbEngine = engine
    return engine
}

func resetDB(engine *xorm.Engine){
    user:=&datastruct.User{}
    robotName:=&datastruct.RobotName{}
    maxScoreInEndlessMode:=&datastruct.MaxScoreInEndlessMode{}
    maxScoreInSinglePersonMode:=&datastruct.MaxScoreInSinglePersonMode{}
    maxScoreInInviteMode:=&datastruct.MaxScoreInInviteMode{}
    skinFragment:=&datastruct.SkinFragment{}
    gameIntegral:=&datastruct.GameIntegral{}
	err:=engine.DropTables(user,robotName,maxScoreInEndlessMode,maxScoreInSinglePersonMode,maxScoreInInviteMode,skinFragment,gameIntegral)
    errhandle(err)
	err=engine.CreateTables(user,robotName,maxScoreInEndlessMode,maxScoreInSinglePersonMode,maxScoreInInviteMode,skinFragment,gameIntegral)
    errhandle(err)
}

func initData(engine *xorm.Engine){
    execStr:=fmt.Sprintf("ALTER DATABASE %s CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci;",conf.Server.DB_Name)
    _,err:=engine.Exec(execStr)
    errhandle(err)
    _,err=engine.Exec("ALTER TABLE user CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;")
    errhandle(err)
    _,err=engine.Exec("ALTER TABLE user CHANGE nick_name nick_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;")
    errhandle(err)

    names:=tools.GetRobotNames()
    _, err = engine.Insert(&names)
    errhandle(err)
    robotPaths = tools.GetRobotPath()
    robotName:=&datastruct.RobotName{}
    robotName.State = 0
    engine.Cols("state").Update(robotName)
    go timerTask(engine)//执行定时任务
}

/*定时任务*/
func timerTask(engine *xorm.Engine){
    c := cron.New()
	spec :="0 0 0 * * 1"
    c.AddFunc(spec, func() {
        table0:="max_score_in_single_person_mode"
        table1:="max_score_in_endless_mode"
        truncate:="truncate table "
        engine.Exec(truncate+table0)
        engine.Exec(truncate+table1)
    })
    c.Start()
}

func errhandle(err error){
	if err != nil {
	   log.Fatal("db error is %v", err.Error())
	}
}

func handleGetMaxScoreInEndlessMode(uid int) (int,int){
    var scoreInfo datastruct.MaxScoreInEndlessMode
    has,err:=dbEngine.Where("uid =?", uid).Get(&scoreInfo)
    score:=0
    killNum:=0
    if err !=nil{
       log.Error("handleGetMaxScoreInEndlessMode err:%v",err)
    }
    if has{
       score = scoreInfo.MaxScore
       killNum = scoreInfo.MaxKillNum
    }
    return score,killNum
}

func handleGetMaxScoreInSinglePersonMode(uid int) (int,int){
    var scoreInfo datastruct.MaxScoreInSinglePersonMode
    has,err:=dbEngine.Where("uid =?", uid).Get(&scoreInfo)
    score:=0
    killNum:=0
    if err !=nil{
       log.Error("handleGetMaxScoreInSinglePersonMode err:%v",err)
    }
    if has{
       score = scoreInfo.MaxScore
       killNum = scoreInfo.MaxKillNum
    }
    return score,killNum
}

func handleGetMaxScoreInviteMode(uid int) (int,int){
    var scoreInfo datastruct.MaxScoreInInviteMode
    has,err:=dbEngine.Where("uid =?", uid).Get(&scoreInfo)
    score:=0
    killNum:=0
    if err !=nil{
       log.Error("handleGetMaxScoreInviteMode err:%v",err)
    }
    if has{
       score = scoreInfo.MaxScore
       killNum = scoreInfo.MaxKillNum
    }
    return score,killNum
}

func handleUpdateMaxScoreInEndlessMode(uid int,score int,killNum int){
    var scoreInfo datastruct.MaxScoreInEndlessMode
    has,_:=dbEngine.Where("uid =?", uid).Get(&scoreInfo)
    timestamp:=time.Now().Unix()
    if !has{
       scoreInfo.Uid = uid
       scoreInfo.MaxScore = score
       scoreInfo.MaxKillNum = killNum
       scoreInfo.UpdateTime = timestamp
       dbEngine.Insert(scoreInfo)
    }else{
       score_str:=fmt.Sprintf("%d",score)
       killNum_str:=fmt.Sprintf("%d",killNum)
       uid_str:=fmt.Sprintf("%d",uid)
       sql := "update max_score_in_endless_mode set max_score=" + score_str + ",max_kill_num="+killNum_str+",update_time="+fmt.Sprintf("%d",timestamp)+" where uid="+uid_str
       dbEngine.Exec(sql)
    }
}

func handleUpdateMaxScoreInSinglePersonMode(uid int,score int,killNum int){
    var scoreInfo datastruct.MaxScoreInSinglePersonMode
    has,_:=dbEngine.Where("uid =?", uid).Get(&scoreInfo)
    timestamp:=time.Now().Unix()
    if !has{
       scoreInfo.Uid = uid
       scoreInfo.MaxScore = score
       scoreInfo.MaxKillNum = killNum
       scoreInfo.UpdateTime = timestamp
       dbEngine.Insert(scoreInfo)
    }else{
       score_str:=fmt.Sprintf("%d",score)
       killNum_str:=fmt.Sprintf("%d",killNum)
       uid_str:=fmt.Sprintf("%d",uid)
       sql := "update max_score_in_single_person_mode set max_score=" + score_str + ",max_kill_num="+killNum_str+",update_time="+fmt.Sprintf("%d",timestamp)+" where uid="+uid_str
       dbEngine.Exec(sql)
    }
}

func handleUpdateMaxScoreInviteMode(uid int,score int,killNum int){
    var scoreInfo datastruct.MaxScoreInInviteMode
    has,_:=dbEngine.Where("uid =?", uid).Get(&scoreInfo)
    if !has{
       scoreInfo.Uid = uid
       scoreInfo.MaxScore = score
       scoreInfo.MaxKillNum = killNum
       dbEngine.Insert(scoreInfo)
    }else{
       score_str:=fmt.Sprintf("%d",score)
       killNum_str:=fmt.Sprintf("%d",killNum)
       uid_str:=fmt.Sprintf("%d",uid)
       sql:= "update max_score_in_invite_mode set max_score=" + score_str + ",max_kill_num="+killNum_str+" where uid="+uid_str
       dbEngine.Exec(sql)
    }
}

func hanleAddFragmentNum(uid int,fragmentNum int){
    if fragmentNum >0{
        var skinFragment datastruct.SkinFragment
        has,_:=dbEngine.Where("uid =?", uid).Get(&skinFragment)
        if !has{
           skinFragment.Uid = uid
           skinFragment.FragmentNum = fragmentNum
           dbEngine.Insert(skinFragment)
        }else{
           fragmentNum=skinFragment.FragmentNum+fragmentNum
           fragmentNum_str:=fmt.Sprintf("%d",fragmentNum)
           uid_str:=fmt.Sprintf("%d",uid)
           sql := "update skin_fragment set fragment_num=" + fragmentNum_str + " where uid="+uid_str
           dbEngine.Exec(sql)
        }   
    }
}

func hanleAddGameIntegral(uid int,integral int){
    if integral >0{
        var gameIntegral datastruct.GameIntegral
        has,_:=dbEngine.Where("uid =?", uid).Get(&gameIntegral)
        if !has{
            gameIntegral.Uid = uid
            gameIntegral.Integral = integral
            dbEngine.Insert(gameIntegral)
        }else{
           integral=gameIntegral.Integral+integral
           integral_str:=fmt.Sprintf("%d",integral)
           uid_str:=fmt.Sprintf("%d",uid)
           sql := "update game_integral set integral=" + integral_str + " where uid="+uid_str
           dbEngine.Exec(sql)
        }   
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

func hanleLengthRank(uid int,mode datastruct.GameModeType,rankStart int,rankEnd int)(*datastruct.PlayerScore,[]datastruct.PlayerRankScoreData){
     desc:="max_score"
     return queryLengthRank(uid,mode,rankStart,rankEnd,desc)
}

func queryLengthRank(uid int,mode datastruct.GameModeType,rankStart int,rankEnd int,desc string)(*datastruct.PlayerScore,[]datastruct.PlayerRankScoreData){
    table,offset,rows:=getTableOffsetRows(mode,rankStart,rankEnd)
    playersData:=getPlayersMaxData(table,desc,offset,rows)
    var currentPlayer datastruct.PlayerScore
    arrData := make([]datastruct.PlayerRankScoreData,0,len(playersData))
    for index,v:=range playersData{
        var rankScoreData datastruct.PlayerRankScoreData
        var playerScore datastruct.PlayerScore
        playerScore.Rank = index + 1
        playerScore.Score = v.MaxScore
        playerScore.Name = v.NickName
        playerScore.Avatar = v.Avatar
        rankScoreData.LengthRank=&playerScore
        arrData = append(arrData,rankScoreData)
    }
    rank,score:=getRankWithDesc(desc,table,uid)
    currentPlayer.Rank = rank
    currentPlayer.Score = score
    return &currentPlayer,arrData
}

func hanleKillNumRank(uid int,mode datastruct.GameModeType,rankStart int,rankEnd int)(*datastruct.PlayerKillNum,[]datastruct.PlayerRankKillNumData){
    desc:="max_kill_num"
    return queryKillNumRank(uid,mode,rankStart,rankEnd,desc)
}

func queryKillNumRank(uid int,mode datastruct.GameModeType,rankStart int,rankEnd int,desc string)(*datastruct.PlayerKillNum,[]datastruct.PlayerRankKillNumData){
    table,offset,rows:=getTableOffsetRows(mode,rankStart,rankEnd)
    playersData:=getPlayersMaxData(table,desc,offset,rows)
    var currentPlayer datastruct.PlayerKillNum
    arrData := make([]datastruct.PlayerRankKillNumData,0,len(playersData))
    for index,v:=range playersData{
        var killNumRankData datastruct.PlayerRankKillNumData
        var playerKillNum datastruct.PlayerKillNum
        playerKillNum.Rank = index + 1
        playerKillNum.KillNum = v.MaxKillNum
        playerKillNum.Name = v.NickName
        playerKillNum.Avatar = v.Avatar
        killNumRankData.KillNumRank=&playerKillNum
        arrData = append(arrData,killNumRankData)
    }
    rank,max:=getRankWithDesc(desc,table,uid)
    currentPlayer.Rank = rank
    currentPlayer.KillNum = max
    return &currentPlayer,arrData
}

func getRankWithDesc(columnName string,table string,uid int)(int,int){
     rank:=0
     max:=0
     sql_str:=fmt.Sprintf("SELECT b.* FROM ( SELECT t.*, @rownum := @rownum + 1 AS rownum FROM (SELECT @rownum := 0) r, (SELECT * FROM %s ORDER BY %s DESC) AS t ) AS b WHERE b.uid = %d",table,columnName,uid)
     results,err:= dbEngine.Query(sql_str)
     if err == nil && len(results)>0{
        tmp_max,err:=strconv.Atoi(string(results[0][columnName]))
        if err==nil{
            max = tmp_max
        }
        tmp_rank,err:=strconv.Atoi(string(results[0]["rownum"]))
        if err==nil{
            rank = tmp_rank
        }
    }
    return rank,max
}

type playerMaxData struct {
    NickName string
    Avatar string
    MaxScore int
    MaxKillNum int
}
func getPlayersMaxData(table string,desc string,offset int ,rows int)[]playerMaxData{
     var playersData []playerMaxData
     sql_str:=fmt.Sprintf("select max_score,nick_name,avatar,max_kill_num from %s m join user u on m.uid = u.id order by %s desc,update_time asc limit %d,%d",table,desc,offset,rows)
     dbEngine.Sql(sql_str).Find(&playersData)
     return playersData
}

func getTableOffsetRows(mode datastruct.GameModeType,rankStart int,rankEnd int)(string,int,int){
    rankStart=rankStart-1
    if rankStart < 0{
       rankStart = 0
    }
    if rankEnd <=0{
       rankEnd = 1 
    }
    offset:=rankStart
    rows:= rankEnd - offset
    table:=""
    switch mode{
    case datastruct.SinglePersonMode:
         table="max_score_in_single_person_mode"
    case datastruct.EndlessMode:
         table="max_score_in_endless_mode"
    default:
    }
    return table,offset,rows
}

func insertTestData(engine *xorm.Engine){
     const avatar="https://wx.qlogo.cn/mmopen/vi_32/XIcVyEkoFxEmrl7z0Iz1NzGgiaLC4pMMb9KY1WO34aYctK2r63tEPulfRfUictw82C3ibjG5PuYOuAU3Qb8wyyeKQ/132"
     for i:=0;i<200;i++{
        user:=new(datastruct.User)
        user.LoginName = fmt.Sprintf("LoginName%d",i+1)
        user.Avatar=avatar
        user.NickName=fmt.Sprintf("test%d",i+1)
        _,err:=engine.Insert(user)
        if err ==nil {
            uid:=user.Id
            if i%2==0{
                var scoreInfo datastruct.MaxScoreInSinglePersonMode
                scoreInfo.Uid = uid
                scoreInfo.MaxScore = tools.RandInt(1,2000+1)
                scoreInfo.UpdateTime = time.Now().Unix()+int64(i)
                scoreInfo.MaxKillNum = tools.RandInt(1,100+1)
                engine.Insert(&scoreInfo)   
            }else{
                var scoreInfo datastruct.MaxScoreInEndlessMode
                scoreInfo.Uid = uid
                scoreInfo.MaxScore = tools.RandInt(1,1000+1)
                scoreInfo.UpdateTime = time.Now().Unix()+int64(i)
                scoreInfo.MaxKillNum = tools.RandInt(1,50+1)
                engine.Insert(&scoreInfo)
            }
        }
     }
}
