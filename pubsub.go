package pubsub

import (
	"log"
	"sync"
)

// PubSub provides an in-memory publish-subscribe mechanism.
// It allows multiple listeners to subscribe to specific triggers and receive published data.
type PubSub struct {
	mu     sync.Mutex                    // Mutex to ensure thread-safe access to events map
	events map[string][]chan interface{} // Map of trigger names to their subscriber channels
}

// NewPubSub initializes a new PubSub instance.
func NewPubSub() *PubSub {
	// Create an instance with an empty map for managing events
	instance := PubSub{events: make(map[string][]chan interface{})}
	return &instance
}

// Subscribe adds a listener (channel) for a specific trigger.
// Returns a function to unsubscribe the listener.
func (ps *PubSub) Subscribe(triggerName string, ch chan interface{}) func() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Add the channel to the list of listeners for the given trigger
	ps.events[triggerName] = append(ps.events[triggerName], ch)

	// Return a function to allow the caller to unsubscribe
	return func() {
		ps.Unsubscribe(triggerName, ch)
	}
}

// Unsubscribe removes a listener (channel) from a specific trigger.
func (ps *PubSub) Unsubscribe(triggerName string, ch chan interface{}) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Find and remove the channel from the list of listeners
	for index, listener := range ps.events[triggerName] {
		if listener == ch {
			ps.events[triggerName] = append(ps.events[triggerName][:index], ps.events[triggerName][index+1:]...)
			close(listener) // Close the channel to signal the end of the subscription
		}
	}
}

// Publish sends data to all listeners of a specific trigger.
func (ps *PubSub) Publish(triggerName string, data interface{}) {
	// Iterate over all listeners for the trigger
	for _, listener := range ps.events[triggerName] {
		go func() {
			// Recover in case a channel is closed or encounters a panic
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered from panic:", r)
				}
			}()

			// Send the data to the listener
			listener <- data
		}()
	}
}
