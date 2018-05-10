package tools

import (
	"crypto/md5"  
    "crypto/rand"  
    "encoding/base64"  
    "encoding/hex" 
    "io"
    "github.com/name5566/leaf/log"
    "server/datastruct"
    "github.com/name5566/leaf/gate" 
)

//生成32位md5字串  
func getMd5String(s string) string {  
    h := md5.New()  
	h.Write([]byte(s))
    return hex.EncodeToString(h.Sum(nil))  
}  
  
//生成Guid字串  
func UniqueId() string {  
    b := make([]byte, 48)  
  
    if _, err := io.ReadFull(rand.Reader, b); err != nil {  
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
    a.SetUserData(datastruct.AgentUserData{
        ConnUUID:UniqueId(),
        Uid:uid,
    })
}
  
