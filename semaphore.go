package semaphore

import (
	"errors"
	"sync"
	"time"
)

// Semaphore is an implementation of semaphore.
type Semaphore struct {
	permits int
	avail   int
	channel chan struct{}
	aMutex  *sync.RWMutex
	rMutex  *sync.Mutex
}

// New creates a new Semaphore with specified number of permits.
func New(permits int) *Semaphore {
	if permits < 1 {
		panic("Invalid number of permits. Less than 1")
	}
	return &Semaphore{
		permits,
		permits,
		make(chan struct{}, permits),
		&sync.RWMutex{},
		&sync.Mutex{},
	}
}

// Acquire acquires one permit. If it is not available, the goroutine will block until it is available.
func (s *Semaphore) Acquire() {
	s.aMutex.Lock()
	defer s.aMutex.Unlock()

	s.channel <- struct{}{}
	s.avail--
}

// AcquireMany is similar to Acquire() but for many permits.
// An error is returned if n is greater number of permits in the semaphore.
func (s *Semaphore) AcquireMany(n int) error {
	if n > s.permits {
		return errors.New("Too many requested permits")
	}
	s.aMutex.Lock()
	defer s.aMutex.Unlock()

	s.avail -= n
	for ; n > 0; n-- {
		s.channel <- struct{}{}
	}
	s.avail += n
	return nil
}

// AcquireWithin is similar to AcquireMany() but cancels if duration elapses before getting the permits.
// Returns true if successful and false if timeout occurs.
func (s *Semaphore) AcquireWithin(n int, d time.Duration) bool {
	timeout := make(chan bool, 1)
	cancel := make(chan bool, 1)
	go func() {
		time.Sleep(d)
		timeout <- true
	}()
	go func() {
		s.AcquireMany(n)
		timeout <- false
		if <-cancel {
			s.ReleaseMany(n)
		}
	}()
	if <-timeout {
		cancel <- true
		return false
	}
	cancel <- false
	return true
}

// Release releases one permit.
func (s *Semaphore) Release() {
	s.rMutex.Lock()
	defer s.rMutex.Unlock()

	<-s.channel
	s.avail++
}

// ReleaseMany releases n permits.
func (s *Semaphore) ReleaseMany(n int) {
	if n > s.permits {
		panic("Too many requested releases")
	}
	for ; n > 0; n-- {
		s.Release()
	}
}

// AvailablePermits gives number of available unacquired permits.
func (s *Semaphore) AvailablePermits() int {
	s.aMutex.RLock()
	defer s.aMutex.RUnlock()

	if s.avail < 0 {
		return 0
	}
	return s.avail
}

// DrainPermits acquires all available permits and return the number of permits acquired.
func (s *Semaphore) DrainPermits() int {
	n := s.AvailablePermits()
	if n > 0 {
		s.AcquireMany(n)
	}
	return n
}
