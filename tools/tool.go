package tools

import (
	"fmt"
	"crypto/md5"  
    crypto_rand "crypto/rand"
    "math/rand" 
    "encoding/base64"  
    "encoding/hex" 
    "io"
    "github.com/name5566/leaf/log"
    "server/datastruct"
    "github.com/name5566/leaf/gate"
    "server/msg"
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
    offset:=300
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

func GetRandomPoint(num1 int,num2 int,quad []msg.Quadrant)[]msg.EnergyPoint{
    //  num1,num2:=GetEnergyNum(msg.TypeA,msg.TypeB,num,maxpower)
     slice_point:=make([]msg.EnergyPoint,0,num1+num2)
     slice_point=append(slice_point,getQuadrantPoints(num1,msg.TypeA,quad)...)
     slice_point=append(slice_point,getQuadrantPoints(num2,msg.TypeB,quad)...)
     return slice_point
}

func getQuadrantPoints(num int,e_type msg.EnergyPointType,quad []msg.Quadrant)[]msg.EnergyPoint{
    slice_point:=make([]msg.EnergyPoint,0,num)
    for i:=0;i<num;i++{
        index:=GetRandomQuadrantIndex()
        quad:=quad[index]
        random_x:=randInt(quad.X_Min,quad.X_Max)
        random_y:=randInt(quad.Y_Min,quad.Y_Max)
        slice_point=append(slice_point,msg.EnergyPoint{
            Type:int(e_type),
            X:random_x,
            Y:random_y,
        })
    }
    return slice_point
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

func ReSetAgentUserData(a gate.Agent,uid int){
    str:=UniqueId()
    a.SetUserData(datastruct.AgentUserData{
        ConnUUID:str,
        Uid:uid,
    })
}

func UpdateAgentUserData(a gate.Agent,connUUID string,uid int,r_id string){
    a.SetUserData(datastruct.AgentUserData{
        ConnUUID:connUUID,
        Uid:uid,
        RoomID:r_id,
    })
}

const startIndex = 30000

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



func CreateRobot(index int,isRelive bool,quad []msg.Quadrant) *datastruct.Robot{
     robot:=new(datastruct.Robot)
     robot.Id = startIndex+index
     robot.IsRelive = isRelive
     robot.Avatar = fmt.Sprintf("Avatar%d",index)
     robot.NickName = fmt.Sprintf("Robot%d",index)
     robot.Action = GetCreateRobotAction(robot.Id,quad)
     robot.SpeedInterval = randInt(minSpeedInterval,maxSpeedInterval+1)
     robot.DirectionInterval = randInt(minDirectionInterval,maxDirectionInterval+1)
     return robot
}

func GetCreateRobotAction(p_id int,quad []msg.Quadrant)msg.CreatePlayer{
    randomIndex:=GetRandomQuadrantIndex()
    point:=GetCreatePlayerPoint(quad[randomIndex],randomIndex) 
    action:=msg.GetCreatePlayerAction(p_id,point.X,point.Y)
    return action
}


func GetRandomDirection()msg.Point{
     return msg.Point{
         X:randInt(minDirection,maxDirection+1),
         Y:randInt(minDirection,maxDirection+1),
     }
}



func GetRandomSpeed()int{
    return randInt(minSpeed,maxSpeed+1)
}

func GetRandomSpeedDuration()int{
    return randInt(minSpeedDuration,maxSpeedDuration+1)
}
