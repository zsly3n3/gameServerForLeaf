package datastruct

import (
	"sync"
)

/*只统计执行过匹配的在线玩家们*/
type OnlinePlayers struct {
	lock *sync.RWMutex //读写互斥量
	bm   map[int]*Player //map[int]*Player 根据Id保存
}



// NewOnlinePlayers return new OnlinePlayers
func NewOnlinePlayers() *OnlinePlayers {
	return &OnlinePlayers{
		lock: new(sync.RWMutex),
		bm:   make(map[int]*Player),
	}
}

// Get from maps return the k's value
func (m *OnlinePlayers) Get(k int) *Player {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if val, ok := m.bm[k]; ok {
		return val
	}
	return nil
}


func (m *OnlinePlayers) GetWithAddr(addr string) (int,*Player) {
	index:=-1
	var pl *Player = nil
	m.lock.RLock()
	for k, v := range m.bm {
		str:=(*v.Agent).RemoteAddr().String()
		if str == addr{
			index = k
			pl = v
			break
		}
	}
	m.lock.RUnlock()
	return index,pl
}

// Set Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (m *OnlinePlayers) Set(k int, v *Player) bool {
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
func (m *OnlinePlayers) Check(k int) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.bm[k]
	return ok
}

// Delete the given key and value.
func (m *OnlinePlayers) Delete(k int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.bm, k)
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





/*
// Items returns all items in OnlinePlayers.
func (m *OnlinePlayers) Items() map[interface{}]interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	r := make(map[interface{}]interface{})
	for k, v := range m.bm {
		r[k] = v
	}
	return r
}

// Count returns the number of items within the map.
func (m *OnlinePlayers) Count() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.bm)
}
*/