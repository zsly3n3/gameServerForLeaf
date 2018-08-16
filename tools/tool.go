package tools

import (
	"fmt"
    "crypto/md5"
    "strconv"
    crypto_rand "crypto/rand"
    "math/rand" 
    "encoding/base64"  
    "encoding/hex" 
    "io"
    "github.com/name5566/leaf/log"
    "server/datastruct"
    "github.com/name5566/leaf/gate"
    "server/msg"
    "server/conf"
    "github.com/360EntSecGroup-Skylar/excelize"
    "server/tools/snowFlakeByGo"
)

var gateUserData datastruct.GateUserData

func randInt(min int,max int) int {
    return min + rand.Intn(max-min)
}

func GetRandomQuadrantIndex() int {
    return randInt(0,4)
}

func GetCreatePlayerPoint(quad msg.Quadrant,index int) msg.Point {
    var point msg.Point
    x_min:=0
    x_max:=0
    y_min:=0
    y_max:=0
    offset:=200
    switch index{
    case 0:
         x_min=quad.X_Min
         x_max=quad.X_Max-offset
         y_min=quad.Y_Min
         y_max=quad.Y_Max-offset
    case 1:
        x_min=quad.X_Min+offset
        x_max=quad.X_Max
        y_min=quad.Y_Min
        y_max=quad.Y_Max-offset
    case 2:
        x_min=quad.X_Min+offset
        x_max=quad.X_Max
        y_min=quad.Y_Min+offset
        y_max=quad.Y_Max
    case 3:
        x_min=quad.X_Min
        x_max=quad.X_Max-offset
        y_min=quad.Y_Min+offset
        y_max=quad.Y_Max
    }
    random_x:=randInt(x_min,x_max)
    random_y:=randInt(y_min,y_max)
    point.X = random_x
    point.Y = random_y
    // //测试
    // point.X = -200
    // point.Y = -100
    return point
}

func GetRandomPoint(num1 int,num2 int,quad []msg.Quadrant)[]datastruct.EnergyPoint{
    //  num1,num2:=GetEnergyNum(msg.TypeA,msg.TypeB,num,maxpower) 
     slice_point:=make([]datastruct.EnergyPoint,0,num1+num2)
     slice_point=append(slice_point,getQuadrantPoints(num1,msg.TypeA,quad)...)
     slice_point=append(slice_point,getQuadrantPoints(num2,msg.TypeB,quad)...)
     return slice_point
}

//测试
// func GetTestPoint()[]msg.EnergyPoint{
//      slice_point:=make([]msg.EnergyPoint,0,1)
//      point:=msg.EnergyPoint{
//         Type:1,
//         X:400,
//         Y:350,
//      }
//      slice_point=append(slice_point,point)
//      return slice_point
// }


func getQuadrantPoints(num int,e_type msg.EnergyPointType,quad []msg.Quadrant)[]datastruct.EnergyPoint{
    slice_point:=make([]datastruct.EnergyPoint,0,num)
    for i:=0;i<num;i++{
        index:=GetRandomQuadrantIndex()
        quad:=quad[index]
        random_x:=randInt(quad.X_Min,quad.X_Max)
        random_y:=randInt(quad.Y_Min,quad.Y_Max)
        slice_point=append(slice_point,datastruct.EnergyPoint{
            Type:int(e_type),
            X:random_x,
            Y:random_y,
            Scale:1.0,
        })
    }
    return slice_point
}

func CheckScalePoints(points []datastruct.EnergyPoint)([]datastruct.EnergyPoint,int){
     expend:=0
     for index,v := range points{
         if v.Scale < 1.0{
            v.Scale = 1.0
            points[index]=v 
         }
         expend+= msg.GetPower(msg.EnergyPointType(v.Type))
     }
     return points,expend
}

func CreateQuadrant(width int,height int,index int) msg.Quadrant{
    var min_x int
    var max_x int
    var min_y int
    var max_y int
    switch index{
    case 1:
        min_x =0
        max_x =width/2.0
        min_y =0
        max_y =height/2.0
    case 2:
        min_x =-width/2.0 
        max_x =0
        min_y =0
        max_y =height/2.0 
    case 3:
        min_x =-width/2.0 
        max_x =0
        min_y =-height/2.0
        max_y =0
    case 4:
        min_x =0
        max_x =width/2.0 
        min_y =-height/2.0
        max_y =0
    }
   
    return msg.Quadrant{
        X_Min:min_x,
        X_Max:max_x,
        Y_Min:min_y,
        Y_Max:max_y,
    }
}


func GetEnergyNum(t1 msg.EnergyPointType,t2 msg.EnergyPointType,num int,power int)(int,int){
    t1Power:=msg.GetPower(t1)
    t2Power:=msg.GetPower(t2)
    y:=(power-num*t1Power)/(t2Power-t1Power)
    x:=num-y
    return x,y
}

//生成32位md5字串  
func getMd5String(s string) string {  
    h := md5.New()  
	h.Write([]byte(s))
    return hex.EncodeToString(h.Sum(nil))  
}  
  
//生成Guid字串  
func UniqueId() string {
    // 生成节点实例
    b := make([]byte, 48)  
    if _, err := io.ReadFull(crypto_rand.Reader, b); err != nil {  
        return ""
    }  
    return getMd5String(base64.URLEncoding.EncodeToString(b))  
}
func UniqueIdFromInt()string{
    worker, _ := snowFlakeByGo.NewWorker(1)
    return fmt.Sprintf("%d",worker.GetId())
}

func IsValid(a gate.Agent) bool{//判断此连接是否有效
    gateUserData.Mutex.RLock()
    defer gateUserData.Mutex.RUnlock()
    _,tf:=gateUserData.UserData[a]
    if !tf{
      log.Error("Conn isValid")
    }
    // tf:=true
    // if data == nil{
    //    log.Error("Conn isValid")
    //    tf=false
    // }
    return tf
}

func ReSetAgentUserData(uid int,mode datastruct.GameModeType,PlayId int,a gate.Agent,connUUID string,extra datastruct.ExtraUserData){
    // a.SetUserData(datastruct.AgentUserData{
    //     ConnUUID:connUUID,
    //     Uid:uid,
    //     GameMode:mode,
    //     PlayId:PlayId,
    //     Extra:extra,
    // })
    gateUserData.Mutex.Lock()
    defer gateUserData.Mutex.Unlock()
    if gateUserData.UserData == nil && len(gateUserData.UserData)<=0{ 
       gateUserData.UserData = make(map[gate.Agent]datastruct.AgentUserData)
    }
    userData:=datastruct.AgentUserData{
            ConnUUID:connUUID,
            Uid:uid,
            GameMode:mode,
            PlayId:PlayId,
            Extra:extra,
    }
    gateUserData.UserData[a]=userData
}

func GetUserData(a gate.Agent)*datastruct.AgentUserData{
    gateUserData.Mutex.RLock()
    defer gateUserData.Mutex.RUnlock()
    data:=gateUserData.UserData[a]
    return &data
}
func RemoveUserData(a gate.Agent){
    gateUserData.Mutex.Lock()
    defer gateUserData.Mutex.Unlock()
    delete(gateUserData.UserData,a)
}


func ReSetExtraRoomID(extra datastruct.ExtraUserData) datastruct.ExtraUserData{
     extra.RoomID = datastruct.NULLSTRING
     extra.WaitRoomID = datastruct.NULLSTRING
     extra.IsSettle = false
     return extra
}


const minDirectionInterval = 5
const maxDirectionInterval = 10

const minDirection = -1000
const maxDirection = 1000

const minSpeedInterval = 5
const maxSpeedInterval = 10

const minSpeedDuration = 2
const maxSpeedDuration = 4

const minSpeed = 2
const maxSpeed = 2


func CreateOfflinePlayerMoved(action *msg.PlayerMoved,moveStep int) *msg.OfflinePlayerMoved{
    offlineMoved:=new(msg.OfflinePlayerMoved)
    offlineMoved.Action = *action
    offlineMoved.MoveStep = moveStep
    // offlineMoved.DirectionInterval=randInt(minDirectionInterval,maxDirectionInterval-2)
    // offlineMoved.SpeedInterval = randInt(minSpeedInterval,maxSpeedInterval+1)
    // offlineMoved.StopSpeedFrameIndex = 0
    return offlineMoved
}

func CreateRobot(name string,index int,isRelive bool,quad []msg.Quadrant,reliveFrameIndex int,pt msg.Point) *datastruct.Robot{
     robot:=new(datastruct.Robot)
     robot.Id = index
     robot.IsRelive = isRelive
     robot.Avatar = fmt.Sprintf("Avatar%d",index)
     robot.NickName = name
     robot.Action = GetCreateRobotAction(pt,robot.Id,quad,reliveFrameIndex,name,0,GetRobotAvatar())
    //  robot.SpeedInterval = randInt(minSpeedInterval,maxSpeedInterval+1)
    //  robot.DirectionInterval = randInt(minDirectionInterval,maxDirectionInterval+1)
    //  robot.StopSpeedFrameIndex = 0
     return robot
}

func GetCreateRobotAction(point msg.Point,p_id int,quad []msg.Quadrant,reliveFrameIndex int,name string,addEnergy int,avatar string)*msg.PlayerRelive{
    //randomIndex:=GetRandomQuadrantIndex()
    //point:=GetCreatePlayerPoint(quad[randomIndex],randomIndex)//测试
    
    action:=msg.GetCreatePlayerAction(p_id,point.X,point.Y,reliveFrameIndex,name,addEnergy,avatar)
    return action
}


func GetRandomDirection()msg.Point{
     return msg.Point{
         X:randInt(minDirection,maxDirection+1),
         Y:randInt(minDirection,maxDirection+1),
     }
}

func GetRandomSpeed()int{
    //return randInt(minSpeed,maxSpeed+1)
    return minSpeed
}

func GetRandomSpeedDuration()int{
    return randInt(minSpeedDuration,maxSpeedDuration+1)
}

func GetRobotNames()[]datastruct.RobotName{
    xlsx, err := excelize.OpenFile("conf/robotNames.xlsx")
    if err != nil {
        fmt.Println(err)
        log.Fatal("Excel error is %v", err.Error())
    }
    index:=1
    names:=make([]datastruct.RobotName, 0)
    for {
        cell_index:= fmt.Sprintf("A%d",index)
        cell := xlsx.GetCellValue("Sheet1", cell_index)
        if cell == "" {
            break
        }
        var robotName datastruct.RobotName
        robotName.Name = cell
        robotName.State = 0
        names = append(names,robotName)
        index++
    }
    return names
}

func GetRandID(names []datastruct.RobotName,count int) int {
    rand:=randInt(1,count+1)
    isExist:=false
    for _,v := range names{
        if rand == v.Id{
           isExist = true
           break
        }
    }
    if !isExist{
       return rand
    }else{
       return GetRandID(names,count)
    }
}
func GetOnceRandID(count int) int {
    rand:=randInt(1,count+1)
    return rand
}

func GetRobotPath()[]map[int]msg.Point{
    xlsx, err := excelize.OpenFile("conf/robot_path.xlsx")
    if err != nil {
        fmt.Println(err)
        log.Fatal("Excel error is %v", err.Error())
    }
    sheets:=9
    paths:=make([]map[int]msg.Point, 0)
    for i:=0;i<sheets;i++{
        map_path:=make(map[int]msg.Point)
        sheet_index:=fmt.Sprintf("Sheet%d",i)
        row:=1
        for {
            frameIndex_cell:= fmt.Sprintf("A%d",row)
            v_a_cell:= fmt.Sprintf("B%d",row)
            v_b_cell:= fmt.Sprintf("C%d",row)
            frameIndex := xlsx.GetCellValue(sheet_index, frameIndex_cell)
            v_a := xlsx.GetCellValue(sheet_index, v_a_cell)
            v_b := xlsx.GetCellValue(sheet_index, v_b_cell)
            if row == 1{
                pt_x_cell:= fmt.Sprintf("M%d",row)
                pt_y_cell:= fmt.Sprintf("N%d",row)
                pt_x_str := xlsx.GetCellValue(sheet_index, pt_x_cell)
                pt_y_str := xlsx.GetCellValue(sheet_index, pt_y_cell)
                pt_x,_ := strconv.Atoi(pt_x_str)
                pt_y,_ := strconv.Atoi(pt_y_str)
                map_path[row-1]=msg.Point{
                    X:pt_x,
                    Y:pt_y,
                }
            }
            if frameIndex == "" {
                break
            }
            var index int
            var a int
            var b int
            var err error
            index,err = strconv.Atoi(frameIndex)
            if err!=nil {
               panic("string to int error")
            }
            a,err = strconv.Atoi(v_a)
            if err!=nil {
               panic("string to int error")
            }
            b,err = strconv.Atoi(v_b)
            if err!=nil {
               panic("string to int error")
            }
            map_path[index]=msg.Point{
                X:a,
                Y:b,
            }
            row++
        }
        paths = append(paths,map_path)
    }
    return paths
}

func GetRandomFromSlice(slice []int)int{
     min:=0
     max:=len(slice)
     rs_index:=randInt(min,max)
     return slice[rs_index]
}

func GetRobotAvatar()string{
     return conf.Server.RemoteHttpServer+"/robotAvatar/robot.jpeg"
}

func EnableSettle(rid string,a gate.Agent) bool{
    tf:=false
    agentUserData := GetUserData(a)
    if rid == agentUserData.Extra.RoomID && !agentUserData.Extra.IsSettle{
       tf = true
       agentUserData.Extra.IsSettle = true
       log.Debug("EnableSettle")
       ReSetAgentUserData(agentUserData.Uid,agentUserData.GameMode,agentUserData.PlayId,a,agentUserData.ConnUUID,agentUserData.Extra)
    }
    return tf
}

func GetFragmentNum(rank int)int{
     rs:=0
     switch rank{
      case 1:
        rs=20
      case 2:
        rs=15
      case 3:
        rs=10
      case 4:
        rs=6
      case 5:
        rs=5
      case 6:
        rs=4
     }
     return rs
}
func GetGameIntegral(rank int) int{
    rs:=0
    switch rank{
      case 1:
        rs=3
      case 2:
        rs=2
      case 3:
        rs=1
    }
    return rs
}
func GetDefaultAvatar() string{
    return conf.Server.RemoteHttpServer+"/robotAvatar/DefaultAvatar.jpg"
}


func RandInt(min int,max int) int {
     return min + rand.Intn(max-min)
}