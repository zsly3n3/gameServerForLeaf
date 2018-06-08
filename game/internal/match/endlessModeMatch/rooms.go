package endlessModeMatch

import (
	"sync"
	"server/datastruct"
)


/*只统计执行过匹配的在线玩家们*/
type Rooms struct {
	lock *sync.RWMutex
	bm   map[string]*Room 
}

func NewRooms() *Rooms {
	return &Rooms{
		lock: new(sync.RWMutex),
		bm:   make(map[string]*Room),
	}
}

// Get from maps return the k's value
func (m *Rooms) Get(k string) (bool,*Room) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if val, ok := m.bm[k]; ok {
		return ok,val
	}
	return false,nil
}


// Set Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (m *Rooms) Set(k string, v *Room) bool {
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
func (m *Rooms) Check(k string) bool{
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.bm[k]
	return ok
}

// Delete the given key and value.
func (m *Rooms) Delete(k string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.bm, k)
}


func (m *Rooms) GetFreeRoomId() (string,bool){
	r_id:=datastruct.NULLSTRING
	tf:=false
	m.lock.RLock()
	defer m.lock.RUnlock()
	for k, v := range m.bm {
		v.Mutex.RLock()
		isOn:=v.IsOn
		v.Mutex.RUnlock()
		if isOn{
			r_id = k
			tf = true
			break
		}

	}
	return r_id,tf
}

