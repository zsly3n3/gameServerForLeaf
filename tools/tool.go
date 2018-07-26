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
    "github.com/360EntSecGroup-Skylar/excelize"
)

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
    b := make([]byte, 48)  
    if _, err := io.ReadFull(crypto_rand.Reader, b); err != nil {  
        return ""
    }  
    return getMd5String(base64.URLEncoding.EncodeToString(b))  
}

func IsValid(data interface{}) bool{//判断此连接是否有效
    tf:=true
    if data == nil{
       log.Error("Conn isValid")
       tf=false
    }
    return tf
}

func ReSetAgentUserData(uid int,mode datastruct.GameModeType,PlayId int,a gate.Agent,connUUID string,extra datastruct.ExtraUserData){
    a.SetUserData(datastruct.AgentUserData{
        ConnUUID:connUUID,
        Uid:uid,
        GameMode:mode,
        PlayId:PlayId,
        Extra:extra,
    })
}
func ReSetExtraRoomID(extra datastruct.ExtraUserData) datastruct.ExtraUserData{
     extra.RoomID = datastruct.NULLSTRING
     extra.WaitRoomID = datastruct.NULLSTRING
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


func CreateOfflinePlayerMoved(currentFrameIndex int,action *msg.PlayerMoved) *msg.OfflinePlayerMoved{
    offlineMoved:=new(msg.OfflinePlayerMoved)
    offlineMoved.Action = *action
    offlineMoved.StartFrameIndex = currentFrameIndex
    offlineMoved.DirectionInterval=randInt(minDirectionInterval,maxDirectionInterval-2)
    offlineMoved.SpeedInterval = randInt(minSpeedInterval,maxSpeedInterval+1)
    offlineMoved.StopSpeedFrameIndex = 0
    return offlineMoved
}

func CreateRobot(name string,index int,isRelive bool,quad []msg.Quadrant,reliveFrameIndex int) *datastruct.Robot{
     robot:=new(datastruct.Robot)
     robot.Id = index
     robot.IsRelive = isRelive
     robot.Avatar = fmt.Sprintf("Avatar%d",index)
     robot.NickName = name
     robot.Action = GetCreateRobotAction(robot,robot.Id,quad,reliveFrameIndex,name,0)
    //  robot.SpeedInterval = randInt(minSpeedInterval,maxSpeedInterval+1)
    //  robot.DirectionInterval = randInt(minDirectionInterval,maxDirectionInterval+1)
    //  robot.StopSpeedFrameIndex = 0
     return robot
}

func GetCreateRobotAction(robot *datastruct.Robot,p_id int,quad []msg.Quadrant,reliveFrameIndex int,name string,addEnergy int)*msg.PlayerRelive{
    //randomIndex:=GetRandomQuadrantIndex()
    //point:=GetCreatePlayerPoint(quad[randomIndex],randomIndex)//测试
    point := msg.Point{
        X:-660,
        Y:-1600,
    }
    robot.MoveStep = 1
    action:=msg.GetCreatePlayerAction(p_id,point.X,point.Y,reliveFrameIndex,name,addEnergy)
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
    row:=1
    paths:=make([]map[int]msg.Point, 0)
    map_path:=make(map[int]msg.Point)
    for {
        frameIndex_cell:= fmt.Sprintf("A%d",row)
        v_a_cell:= fmt.Sprintf("B%d",row)
        v_b_cell:= fmt.Sprintf("C%d",row)
        frameIndex := xlsx.GetCellValue("Sheet1", frameIndex_cell)
        v_a := xlsx.GetCellValue("Sheet1", v_a_cell)
        v_b := xlsx.GetCellValue("Sheet1", v_b_cell)
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
    return paths
}


