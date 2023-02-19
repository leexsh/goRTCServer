package ws

import "sync"

// Rooms 房间管理对象
type Rooms struct {
	roomMap map[string]*Room
	sync.Mutex
}

// NewRooms 新建rooms对象
func NewRooms() *Rooms {
	return &Rooms{
		roomMap: make(map[string]*Room),
	}
}

// AddRoom 新增room
func (r *Rooms) AddRoom(rid string) *Room {
	room := r.GetRoom(rid)
	if room == nil {
		room = NewRoom(rid)
		r.Lock()
		r.roomMap[rid] = room
		r.Unlock()
	}
	return room
}

// DelRoom 删除Room
func (r *Rooms) DelRoom(rid string) {
	r.Lock()
	defer r.Unlock()
	if r.roomMap[rid] != nil {
		r.roomMap[rid].Close()
		delete(r.roomMap, rid)
	}
}

// GetRoom 获取Room
func (r *Rooms) GetRoom(rid string) *Room {
	r.Lock()
	defer r.Unlock()
	if room, ok := r.roomMap[rid]; ok {
		return room
	}
	return nil
}

// GetRoom 获取rooms
func (r *Rooms) GetRooms() map[string]*Room {
	return r.roomMap
}

// NotifyWithUid 通知房间指定人
func (r *Rooms) NotifyWithUid(rid, uid, method string, data map[string]interface{}) {
	room := r.roomMap[rid]
	if room != nil {
		room.NotifyWithUid(uid, method, data)
	}
	return
}

// NotifyWithoutUid 通知房间里面其他人
func (r *Rooms) NotifyWithoutUid(rid, fuid, method string, data map[string]interface{}) {
	room := r.GetRoom(rid)
	if room != nil {
		room.NotifyWithoutUid(fuid, method, data)
	}
}

// NotifyAll 通知房间里面所有人
func (r *Rooms) NotifyAll(rid, method string, data map[string]interface{}) {
	room := r.GetRoom(rid)
	if room != nil {
		room.NotifyAll(method, data)
	}
}
