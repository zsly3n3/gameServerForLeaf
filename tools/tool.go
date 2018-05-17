package tools

import (
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

func GetRandomPoint(quad msg.Quadrant,num int,maxRangeType int)[]msg.Point{
     slice_point:=make([]msg.Point,0,num)
     for i:=0;i<num;i++{
         random_x:=randInt(quad.X_Min,quad.X_Max)
         random_y:=randInt(quad.Y_Min,quad.Y_Max)
         slice_point=append(slice_point,msg.Point{
             Type:randInt(msg.TypeA,maxRangeType+1),
             X:random_x,
             Y:random_y,
         })
     }
     return slice_point
}

func CreateQuadrant(length int,width int,index int) msg.Quadrant{
    var min_x int
    var max_x int
    var min_y int
    var max_y int

    switch index{
    case 1:
        min_x =0
        max_x =length/2.0
        min_y =0
        max_y =width/2.0 
    case 2:
        min_x =-length/2.0 
        max_x =0
        min_y =0
        max_y =width/2.0 
    case 3:
        min_x =-length/2.0 
        max_x =0
        min_y =-width/2.0
        max_y =0
    case 4:
        min_x =0
        max_x =length/2.0 
        min_y =-width/2.0
        max_y =0
    }
   
    return msg.Quadrant{
        X_Min:min_x,
        X_Max:max_x,
        Y_Min:min_y,
        Y_Max:max_y,
    }
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

