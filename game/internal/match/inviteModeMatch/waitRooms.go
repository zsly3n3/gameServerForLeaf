package inviteModeMatch

import (
	"sync"
)

/*只统计执行过匹配的在线玩家们*/
type WaitRooms struct {
	lock *sync.RWMutex
	bm   map[string]*WaitRoom
}

func NewWaitRooms() *WaitRooms {
	return &WaitRooms{
		lock: new(sync.RWMutex),
		bm:   make(map[string]*WaitRoom),
	}
}
// Set Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (m *WaitRooms) Set(k string, v *WaitRoom) bool {
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

// Get from maps return the k's value
func (m *WaitRooms) Get(k string) (bool,*WaitRoom) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if val, ok := m.bm[k]; ok {
		return ok,val
	}
	return false,nil
}

// Delete the given key and value.
func (m *WaitRooms) Delete(k string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.bm, k)
}