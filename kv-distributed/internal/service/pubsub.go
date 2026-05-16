package service

import (
	"sync"
)

// PubSub Engine for real-time messaging
type PubSub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan string]bool
}

func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[string]map[chan string]bool),
	}
}

// Subscribe returns a channel that will receive messages for the given channel name.
func (ps *PubSub) Subscribe(channel string) chan string {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.subscribers[channel]; !ok {
		ps.subscribers[channel] = make(map[chan string]bool)
	}

	// Buffered channel so Publish doesn't block if client is slow
	ch := make(chan string, 100)
	ps.subscribers[channel][ch] = true
	return ch
}

// Unsubscribe removes a channel from the subscribers list.
func (ps *PubSub) Unsubscribe(channel string, ch chan string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if subs, ok := ps.subscribers[channel]; ok {
		if _, exists := subs[ch]; exists {
			delete(subs, ch)
			close(ch)
		}
		if len(subs) == 0 {
			delete(ps.subscribers, channel)
		}
	}
}

// Publish broadcasts a message to all subscribers of a channel.
func (ps *PubSub) Publish(channel string, message string) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if subs, ok := ps.subscribers[channel]; ok {
		for ch := range subs {
			// Non-blocking send to avoid holding up the loop if a channel is full
			select {
			case ch <- message:
			default:
				// If channel is full, drop message for this slow client
			}
		}
	}
}
