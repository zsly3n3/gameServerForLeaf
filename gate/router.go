package gate

import (  
    "server/game"  
    "server/login" 
    "server/msg"
)  

func init() {
     // 这里指定消息 Test 路由到 game 模块  
    // 模块间使用 ChanRPC 通讯，消息路由也不例外  
    //msg.Processor.SetRouter(&msg.Test{}, game.ChanRPC)
	//路由分发数据到login
    msg.Processor.SetRouter(&msg.CS_UserLogin{}, login.ChanRPC)
    msg.Processor.SetRouter(&msg.CS_PlayerMatching{}, game.ChanRPC)
    msg.Processor.SetRouter(&msg.CS_PlayerCancelMatching{}, game.ChanRPC)
    msg.Processor.SetRouter(&msg.CS_PlayerJoinRoom{}, game.ChanRPC)
    msg.Processor.SetRouter(&msg.CS_RoomFrameData{}, game.ChanRPC)
}
