package fsnotify

import (
	"fmt"
)

// Op describes a set of file operations.
type Op uint32

// Event represents a single file system notification.
type Event struct {
	Name string // Path to the file or directory.
	Op   Op     // File operation that triggered the event.
}

// String returns a string representation of the event in the form
// "file: REMOVE|WRITE|..."
func (e Event) String() string {
	return fmt.Sprintf("%q: %s", e.Name, e.Op.String())
}
