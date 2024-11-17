package pubsub

import (
	"log"
	"sync"
)

type PubSub struct {
	mu     sync.Mutex
	events map[string][]chan interface{}
}

func NewPubSub() *PubSub {
	instance := PubSub{events: make(map[string][]chan interface{})}

	return &instance
}

func (ps *PubSub) Subscribe(triggerName string, ch chan interface{}) func() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.events[triggerName] = append(ps.events[triggerName], ch)

	return func() {
		ps.Unsubscribe(triggerName, ch)
	}
}

func (ps *PubSub) Unsubscribe(triggerName string, ch chan interface{}) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for index, listener := range ps.events[triggerName] {
		if listener == ch {
			ps.events[triggerName] = append(ps.events[triggerName][:index], ps.events[triggerName][index+1:]...)
			close(listener)
		}
	}
}

func (ps *PubSub) Publish(triggerName string, data interface{}) {
	for _, listener := range ps.events[triggerName] {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered from panic:", r)
				}
			}()

			listener <- data
		}()
	}
}
