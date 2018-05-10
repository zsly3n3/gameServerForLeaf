package datastruct

import (
	"sync"
)

const NULLKEY = ""

/*只统计执行过匹配的在线玩家们*/
type OnlinePlayers struct {
	lock *sync.RWMutex //读写互斥量
	bm   map[string]Player //map[int]*Player 根据Id保存
}



// NewOnlinePlayers return new OnlinePlayers
func NewOnlinePlayers() *OnlinePlayers {
	return &OnlinePlayers{
		lock: new(sync.RWMutex),
		bm:   make(map[string]Player),
	}
}

// Get from maps return the k's value
func (m *OnlinePlayers) Get(k string) Player {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if val, ok := m.bm[k]; ok {
		return val
	}
	return Player{}
}

// Get from maps return the k's value
func (m *OnlinePlayers) Gets(keys []string) []Player {
	players := make([]Player, 0)
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _,k := range keys{
		if val, ok := m.bm[k]; ok {
		   players = append(players,val)
		}
	}
	return players
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
func (m *OnlinePlayers) Set(k string, v Player) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if val, ok := m.bm[k]; !ok {
		m.bm[k] = v
	} else if val != v {
		m.bm[k] = v
	} else {
		return false
	}
	return true
}

// Check Returns true if k is exist in the map.
func (m *OnlinePlayers) Check(k string) (Player,bool){
	m.lock.RLock()
	defer m.lock.RUnlock()
	v, ok := m.bm[k]
	return v,ok
}


// Delete the given key and value.
func (m *OnlinePlayers) Delete(k string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.bm, k)
}

// Items returns all items in safemap.
func (m *OnlinePlayers) Items() map[string]Player {
	m.lock.RLock()
	r := make(map[string]Player)
	for k, v := range m.bm {
		r[k] = v
	}
	m.lock.RUnlock()
	return r
}


// func (m *OnlinePlayers) RemovePlayer(addr string) {
// 	m.lock.RLock()
// 	index:=-1
// 	for k, v := range m.bm {
// 		str:=(*v.Agent).RemoteAddr().String()
// 		if str == addr{
// 			index = k
// 			break
// 		}
// 	}
// 	m.lock.RUnlock()
//     if index > 0{
//         m.lock.Lock()
// 		defer m.lock.Unlock()
// 		delete(m.bm, index)
// 	}
// }





// Count returns the number of items within the map.
func (m *OnlinePlayers) Count() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.bm)
}
