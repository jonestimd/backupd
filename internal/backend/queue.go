package backend

import (
	"container/list"
	"sync"
)

// Action is an enum of actions to perform for a file.
type Action int

const (
	// StoreAction indicates that a new file needs to be backed up.
	StoreAction Action = iota + 1
	// UpdateAction indicates that a backed up file has been modified.
	UpdateAction
	// TrashAction indicates that a backed up file has been deleted.
	TrashAction
)

// Message contains a pending action for a file.
type Message struct {
	local  *string
	remote *string
	action Action
}

// Queue maintains a list of pending backup updates.
type Queue struct {
	items *list.List
	mutex *sync.Mutex
	ready *sync.Cond
}

// NewQueue creates an empty queue.
func NewQueue() *Queue {
	mutex := &sync.Mutex{}
	return &Queue{list.New(), mutex, sync.NewCond(mutex)}
}

// Add appends a message to the queue.
func (q *Queue) Add(m *Message) {
	q.mutex.Lock()
	q.items.PushBack(m)
	q.mutex.Unlock()
	q.ready.Signal()
}

// Get gets a message from the queue.
func (q *Queue) Get() *Message {
	q.mutex.Lock()
	for q.items.Len() == 0 {
		q.ready.Wait()
	}
	e := q.items.Front()
	q.items.Remove(e)
	q.mutex.Unlock()
	return e.Value.(*Message)
}
