package fsnotify

import (
	"errors"
)

const (
	MaxEvents = 4096
)

type Notifier interface {
	AddWatch(name string, flags uint32) error
	AddWatchRecursive(name string, flags uint32) error
	RemoveWatch(name string) error
	Read() ([]Event, error)
	Close() error
}

// Common errors that can be reported by a watcher
var (
	ErrEventOverflow = errors.New("fsnotify queue overflow")

	// ErrWatcherClosed is returned when the poller is closed
	ErrWatcherClosed = errors.New("watcher is closed")

	// ErrNoSuchWatch is returned when trying to remove a watch that doesn't exist
	ErrNoSuchWatch = errors.New("watch does not exist")
)

type Watcher struct {
	notifier Notifier
	events   chan Event
	errors   chan error
	done     chan struct{}
	more     chan struct{}
}

func NewWatcher() (*Watcher, error) {

	n, err := NewNotifier()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		notifier: n,
		events:   make(chan Event),
		errors:   make(chan error),
		done:     make(chan struct{}),
		more:     make(chan struct{}, 1),
	}

	go watcher.loop()

	return watcher, nil
}

func (w *Watcher) loop() {
	var pending []Event

	for {
		select {
		case <-w.done:
			close(w.events) // tells receiver we're done
			return
		case <-w.more:
			for _, e := range pending {
				w.events <- e
			}
		default:
			if len(pending) > MaxEvents {
				continue
			}
			events, err := w.notifier.Read()
			if err != nil {
				w.errors <- err
				continue
			}
			pending = append(pending, events...)
			w.setMore()
		}
	}
}

func (w *Watcher) setMore() {
	// If we cannot send on the channel, it means the signal already exists
	// and has not been consumed yet.
	select {
	case w.more <- struct{}{}:
	default:
	}
}

func (w *Watcher) Events() <-chan Event {
	return w.events
}

func (w *Watcher) Errors() chan error {
	return w.errors
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.notifier.Close()
}

func (w *Watcher) AddWatch(name string, flags uint32) error {
	return w.notifier.AddWatch(name, flags)
}
