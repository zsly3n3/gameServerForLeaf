package datastruct

import (
	"sync"
)



/*只统计执行过匹配的在线玩家们*/
type OnlinePlayers struct {
	Lock *sync.RWMutex //读写互斥量
	Bm   map[string]Player //map[int]*Player 根据Id保存
}


// NewOnlinePlayers return new OnlinePlayers
func NewOnlinePlayers() *OnlinePlayers {
	return &OnlinePlayers{
		Lock: new(sync.RWMutex),
		Bm:   make(map[string]Player),
	}
}

// Get from maps return the k's value
func (m *OnlinePlayers) Get(k string) (Player,bool) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	val, ok := m.Bm[k]
	if  ok {
		return val,ok
	}
	return Player{},ok
}


// func (m *OnlinePlayers) GetAndUpdateState(key []string,state PlayerEnterType,room_id string) []Player {
// 	m.lock.RLock()
// 	defer m.lock.RUnlock()
// 	if val, ok := m.bm[k]; ok {
// 		return val
// 	}
// 	return Player{}
// }

func (m *OnlinePlayers) GetsAndUpdateState(keys []string,state PlayerEnterType,room_id string) []Player {
	players := make([]Player, 0)
	m.Lock.Lock()
	defer m.Lock.Unlock()
	for _,k := range keys{
		if val, ok := m.Bm[k]; ok {
		   val.GameData.EnterType = state
		   val.GameData.RoomId = room_id
		   players = append(players,val)
		   m.Bm[k] = val
		}
	}
	return players
}

func (m *OnlinePlayers) GetAndUpdateState(k string,state PlayerEnterType,room_id string) (Player,bool)  {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	val, ok := m.Bm[k];
	if ok {
		   val.GameData.EnterType = state
		   val.GameData.RoomId = room_id
		   m.Bm[k] = val
	}
    return val,ok
}

func (m *OnlinePlayers) CheckAndCleanState(k string,state PlayerEnterType,room_id string) (Player,bool)  {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	val, ok := m.Bm[k];
	if ok {
		   tmp0:=val.GameData.EnterType
		   tmp1:=val.GameData.RoomId
		   val.GameData.EnterType = state
		   val.GameData.RoomId = room_id
		   m.Bm[k] = val
		   val.GameData.EnterType = tmp0
		   val.GameData.RoomId = tmp1
	}
    return val,ok
}



// func (m *OnlinePlayers) GetWithAddr(addr string) (string,*Player) {
// 	key:=NULLKEY
// 	var pl *Player = nil
// 	m.lock.RLock()
// 	for k, v := range m.bm {
// 		str:=v.Agent.RemoteAddr().String()
// 		if str == addr{
// 			key = k
// 			pl = v
// 			break
// 		}
// 	}
// 	m.lock.RUnlock()
// 	return key,pl
// }

// Set Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
// func (m *OnlinePlayers) Set(k string, v Player) bool {
// 	m.Lock.Lock()
// 	defer m.Lock.Unlock()
// 	if val, ok := m.Bm[k]; !ok {
// 		m.Bm[k] = v
// 	} else if val != v {
// 		m.Bm[k] = v
// 	} else {
// 		return false
// 	}
// 	return true
// }

// Check Returns true if k is exist in the map.
func (m *OnlinePlayers) Check(k string) (Player,bool){
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	v, ok := m.Bm[k]
	return v,ok
}

func (m *OnlinePlayers) IsExist(k string) bool{
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	_, ok := m.Bm[k]
	return ok
}




// Delete the given key and value.
func (m *OnlinePlayers) Delete(k string) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	delete(m.Bm, k)
}

// Items returns all items in safemap.
func (m *OnlinePlayers) Items() map[string]Player {
	m.Lock.RLock()
	r := make(map[string]Player)
	for k, v := range m.Bm {
		r[k] = v
	}
	m.Lock.RUnlock()
	return r
}







// Count returns the number of items within the map.
func (m *OnlinePlayers) Count() int {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return len(m.Bm)
}
