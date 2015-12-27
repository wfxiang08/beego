// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package session

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

//
// 似乎为一个: LRU
//
var mempder = &MemProvider{list: list.New(), sessions: make(map[string]*list.Element)}

// memory session store.
// it saved sessions in a map in memory.
type MemSessionStore struct {
	sid          string                      //session id
	timeAccessed time.Time                   //last access time
	value        map[interface{}]interface{} //session store
	lock         sync.RWMutex
}

// set value to memory session
func (st *MemSessionStore) Set(key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value[key] = value
	return nil
}

// get value from memory session by key
func (st *MemSessionStore) Get(key interface{}) interface{} {
	st.lock.RLock()
	defer st.lock.RUnlock()
	if v, ok := st.value[key]; ok {
		return v
	} else {
		return nil
	}
}

// delete in memory session by key
func (st *MemSessionStore) Delete(key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.value, key)
	return nil
}

// clear all values in memory session
func (st *MemSessionStore) Flush() error {
	st.lock.Lock()
	defer st.lock.Unlock()

	// 只是清空和 sid 对应的数据
	st.value = make(map[interface{}]interface{})
	return nil
}

// get this id of memory session store
func (st *MemSessionStore) SessionID() string {
	return st.sid
}

// Implement method, no used.
func (st *MemSessionStore) SessionRelease(w http.ResponseWriter) {
}

//
// 实现不同的Session的管理
//
type MemProvider struct {
	lock        sync.RWMutex             // locker
	sessions    map[string]*list.Element // map in memory
	list        *list.List               // for gc
	maxlifetime int64
	savePath    string
}

// init memory session
func (pder *MemProvider) SessionInit(maxlifetime int64, savePath string) error {
	pder.maxlifetime = maxlifetime
	pder.savePath = savePath
	return nil
}

// get memory session store by sid
func (pder *MemProvider) SessionRead(sid string) (SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[sid]; ok {
		// 异步更新数据
		// 为什么不同步执行呢? 因为Lock的类型不一样
		go pder.SessionUpdate(sid)

		pder.lock.RUnlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()

		// 不存在对应的Session, 则新建一个Session
		newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[sid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
}

// check session store exist in memory session by sid
func (pder *MemProvider) SessionExist(sid string) bool {
	pder.lock.RLock()
	defer pder.lock.RUnlock()
	if _, ok := pder.sessions[sid]; ok {
		return true
	} else {
		return false
	}
}

// generate new sid for session store in memory session
func (pder *MemProvider) SessionRegenerate(oldsid, sid string) (SessionStore, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[oldsid]; ok {
		// 更新oldsid对应的Item
		go pder.SessionUpdate(oldsid)
		pder.lock.RUnlock()
		pder.lock.Lock() // 谁先获取lock呢? 如果是上面先获取，则OK; 否则逻辑上出现问题

		// 切换: sid, 删除 oldsid
		element.Value.(*MemSessionStore).sid = sid
		pder.sessions[sid] = element
		delete(pder.sessions, oldsid)
		pder.lock.Unlock()
		return element.Value.(*MemSessionStore), nil
	} else {
		pder.lock.RUnlock()
		pder.lock.Lock()
		//  不存在oldsid, 则直接添加新的sid
		newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
		element := pder.list.PushBack(newsess)
		pder.sessions[sid] = element
		pder.lock.Unlock()
		return newsess, nil
	}
}

// delete session store in memory session by id
func (pder *MemProvider) SessionDestroy(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
		return nil
	}
	return nil
}

// clean expired session stores in memory session
func (pder *MemProvider) SessionGC() {
	pder.lock.RLock()
	for {
		element := pder.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*MemSessionStore).timeAccessed.Unix() + pder.maxlifetime) < time.Now().Unix() {
			pder.lock.RUnlock()

			// 注意Lock的使用
			pder.lock.Lock()
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*MemSessionStore).sid)
			pder.lock.Unlock()

			pder.lock.RLock()
		} else {
			break
		}
	}
	pder.lock.RUnlock()
}

// get count number of memory session
func (pder *MemProvider) SessionAll() int {
	return pder.list.Len()
}

// expand time of session store by id in memory session
func (pder *MemProvider) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	// 异步更新: element的 timeAccessed以及调整它在LRU中的位置
	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*MemSessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
		return nil
	}
	return nil
}

// 在 init 中注册: provider
func init() {
	Register("memory", mempder)
}
