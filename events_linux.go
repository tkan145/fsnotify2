//go:build linux
// +build linux

package fsnotify

import "golang.org/x/sys/unix"

func (o Op) String() (str string) {
	name := func(event Op, name string) {
		if o&event == 0 {
			return
		}
		if str != "" {
			str += "|"
		}
		str += name
	}

	name(unix.IN_ACCESS, "ACCESS")
	name(unix.IN_MODIFY, "MODIFY")
	name(unix.IN_ATTRIB, "ATTRIB")
	name(unix.IN_CLOSE_WRITE, "CLOSE_WRITE")
	name(unix.IN_CLOSE_NOWRITE, "CLOSE_NOWRITE")
	name(unix.IN_OPEN, "OPEN")
	name(unix.IN_MOVED_FROM, "MOVED_FROM")
	name(unix.IN_MOVED_TO, "MOVED_TO")
	name(unix.IN_CREATE, "CREATE")
	name(unix.IN_DELETE, "DELETE")
	name(unix.IN_DELETE_SELF, "DELETE_SELF")
	name(unix.IN_UNMOUNT, "UNMOUNT")
	name(unix.IN_Q_OVERFLOW, "Q_OVERFLOW")
	name(unix.IN_IGNORED, "IGNORE")
	name(unix.IN_CLOSE, "CLOSE")
	name(unix.IN_MOVE_SELF, "MOVE_SELF")
	name(unix.IN_ISDIR, "ISDIR")
	name(unix.IN_ONESHOT, "ONESHOT")
	return
}
