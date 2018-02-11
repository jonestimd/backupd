package backend

import (
	"testing"
	"reflect"
)

func newMessage(local string, remote string, action Action) *message {
	return &message{&local, &remote, action}
}

func TestQueue_IsFifo(t *testing.T) {
	messages := []*message{
		newMessage("local path 1", "remote path 1", StoreAction),
		newMessage("local path 2", "remote path 2", StoreAction),
	}
	q := NewQueue()

	for _, m := range messages {
		q.Add(m)
	}

	for _, expected := range messages {
		actual := q.Get()
		if ! reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected %v but got %v", expected, actual)
		}
	}
}

func TestQueue_GetWaitsForMessage(t *testing.T) {
	messages := []*message{
		newMessage("local path 1", "remote path 1", StoreAction),
		newMessage("local path 2", "remote path 2", StoreAction),
	}
	q := NewQueue()
	ch := make(chan *message)
	go func() {
		for range messages {
			ch <- q.Get()
		}
		close(ch)
	}()

	for _, m := range messages {
		q.Add(m)
	}

	for _, expected := range messages {
		actual := <-ch
		if ! reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected %v but got %v", expected, actual)
		}
	}
}