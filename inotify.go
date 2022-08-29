//go:build linux
// +build linux

package fsnotify

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

type inotify struct {
	mtx      sync.Mutex
	f        *os.File
	watches  map[string]uint32
	rwatches map[uint32]string
	events   chan Event
	fd       int
}

func NewNotifier() (Notifier, error) {
	// Create inotify fd
	fd, errno := unix.InotifyInit1(unix.IN_CLOEXEC)
	if fd == -1 {
		return nil, errno
	}

	errno = unix.SetNonblock(fd, true)
	if errno != nil {
		return nil, errno
	}

	return &inotify{
		fd:       fd,
		f:        os.NewFile(uintptr(fd), "inotify"),
		watches:  make(map[string]uint32),
		rwatches: make(map[uint32]string),
	}, nil
}

// Add starts watching the named file or directory (non-recursively).
func (i *inotify) AddWatch(name string, flags uint32) error {
	name = filepath.Clean(name)
	// TODO: do we need to validate the path name
	// If flags is still 0, make it all events.
	if flags == 0 {
		flags = unix.IN_ALL_EVENTS
	}

	wd, err := unix.InotifyAddWatch(i.fd, name, flags)
	if err != nil {
		return err
	}

	i.mtx.Lock()
	i.watches[name] = uint32(wd)
	i.rwatches[uint32(wd)] = name
	i.mtx.Unlock()
	return nil
}

// AddRecursive watchs on an entire directory tree.
// if path is a directory, every subdirectory will also be watched for the events
// up to the maximum readable depth. If the path is a file, the file is watched
// exactly as if Add were used.
func (i *inotify) AddWatchRecursive(path string, flags uint32) error {
	if err := i.AddWatch(path, flags); err != nil {
		return err
	}

	// traverse existing children in the dir
	return filepath.WalkDir(path, func(p string, info fs.DirEntry, err error) error {
		if err != nil {
			if p == path {
				return fmt.Errorf("error accessing path: %s error: %v", path, err)
			}
			return nil
		}

		// do not call fsWatcher.Add twice on the root dir to avoid potential problems.
		if p == path {
			return nil
		}

		if info.IsDir() {
			// Watch each directory within this directory
			if err := i.AddWatchRecursive(p, flags); err != nil {
				return fmt.Errorf("failed to watch %s, err: %v", path, err)
			}
		}

		return nil
	})
}

// Add starts watching the named file or directory (non-recursively).
func (i *inotify) RemoveWatch(name string) error {
	name = filepath.Clean(name)

	i.mtx.Lock()
	defer i.mtx.Unlock()

	wd, ok := i.watches[name]
	if !ok {
		return nil
	}

	_, err := unix.InotifyRmWatch(i.fd, wd)
	if err != nil {
		return err
	}

	delete(i.watches, name)
	delete(i.rwatches, wd)
	return nil
}

func (i *inotify) Read() ([]Event, error) {
	var (
		events []Event
		buf    [unix.SizeofInotifyEvent * MaxEvents]byte
	)

	n, err := i.f.Read(buf[:])
	if err != nil {
		return nil, err
	}

	if n < unix.SizeofInotifyEvent {
		return nil, fmt.Errorf("inotify: short read")
	}

	var offset int

	for offset+unix.SizeofInotifyEvent <= n {
		// Point "raw" to the event in the buffer
		raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))

		mask := uint32(raw.Mask)
		nameLen := uint32(raw.Len)

		if mask&unix.IN_Q_OVERFLOW != 0 {
			return nil, ErrEventOverflow
		}

		// If the event happened to the watched directory or the watched file, the kernel
		// doesn't append the filename to the event, but we would like to always fill the
		// the "Name" field with a valid filename. We retrieve the path of the watch from
		// the "paths" map.
		i.mtx.Lock()
		name, ok := i.rwatches[uint32(raw.Wd)]

		// IN_DELETE_SELF occurs when the file/directory being watched is removed.
		// This is a sign to clean up the maps, otherwise we are no longer in sync
		// with the inotify kernel state which has already deleted the watch
		// automatically.
		if ok && mask&unix.IN_DELETE_SELF == unix.IN_DELETE_SELF {
			delete(i.rwatches, uint32(raw.Wd))
			delete(i.watches, name)
		}
		i.mtx.Unlock()

		if nameLen > 0 {
			// Point "bytes" at the first byte of the filename
			bytes := (*[unix.PathMax]byte)(unsafe.Pointer(&buf[offset+unix.SizeofInotifyEvent]))[:nameLen:nameLen]
			// The filename is padded with NULL bytes. TrimRight() gets rid of those.
			name += "/" + strings.TrimRight(string(bytes[0:nameLen]), "\000")
		}

		events = append(events, Event{Name: name, Op: Op(mask)})

		// Move to the next event in the buffer
		offset += unix.SizeofInotifyEvent + int(raw.Len)
	}

	return events, nil
}

// Close should be called when inotify is no longer needed in order to cleanup used resources.
func (i *inotify) Close() error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	for _, w := range i.watches {
		_, err := unix.InotifyRmWatch(i.fd, w)
		if err != nil {
			return err
		}
	}
	return i.f.Close()
}
