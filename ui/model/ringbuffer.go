package model

import (
	"container/ring"
)

// RingBuffer is a circular buffer for efficient message storage.
type RingBuffer struct {
	r     *ring.Ring
	size  int
	count int
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		r:    ring.New(capacity),
		size: capacity,
	}
}

// Append adds a message to the buffer.
func (rb *RingBuffer) Append(msg Message) {
	rb.r.Value = msg
	rb.r = rb.r.Next()
	if rb.count < rb.size {
		rb.count++
	}
}

// Get returns all messages in order.
func (rb *RingBuffer) Get() []Message {
	if rb.count == 0 {
		return nil
	}
	
	result := make([]Message, 0, rb.count)
	
	// Start from the oldest message
	start := rb.r
	if rb.count == rb.size {
		// Buffer is full, start at current position
		start = rb.r
	}
	
	for i := 0; i < rb.count; i++ {
		if start.Value != nil {
			result = append(result, start.Value.(Message))
		}
		start = start.Next()
	}
	
	return result
}

// Len returns the number of messages in the buffer.
func (rb *RingBuffer) Len() int {
	return rb.count
}

// Cap returns the capacity of the buffer.
func (rb *RingBuffer) Cap() int {
	return rb.size
}

// Clear empties the buffer.
func (rb *RingBuffer) Clear() {
	rb.r = ring.New(rb.size)
	rb.count = 0
}
