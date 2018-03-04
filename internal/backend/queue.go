package backend

import (
	"container/list"
	"sync"
)

type Action int

const (
	StoreAction  Action = iota + 1
	UpdateAction
	TrashAction
)

// Pending action for a file.
type message struct {
	local  *string
	remote *string
	action  Action
}

type queue struct {
	items *list.List
	mutex *sync.Mutex
	ready *sync.Cond
}

func NewQueue() *queue {
	mutex := &sync.Mutex{}
	return &queue{list.New(), mutex, sync.NewCond(mutex)}
}

func (q *queue) Add(m *message) {
	q.mutex.Lock()
	q.items.PushBack(m)
	q.mutex.Unlock()
	q.ready.Signal()
}

func (q *queue) Get() (*message) {
	q.mutex.Lock()
	for q.items.Len() == 0 {
		q.ready.Wait()
	}
	e := q.items.Front()
	q.items.Remove(e)
	q.mutex.Unlock()
	return e.Value.(*message)
}
