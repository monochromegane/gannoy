package gannoy

import (
	"io"
	"syscall"
)

type Locker interface {
	ReadLock(uintptr, int64, int64) error
	WriteLock(uintptr, int64, int64) error
	UnLock(uintptr, int64, int64) error
}

// Only Linux and kernel version 3.15 or later.
// This depends on open file description lock (F_OFD_SETLKW).
type Fcntl struct {
}

const F_OFD_SETLKW = 38

func (f Fcntl) ReadLock(fd uintptr, start, len int64) error {
	return f.fcntl(syscall.F_RDLCK, fd, start, len)
}

func (f Fcntl) WriteLock(fd uintptr, start, len int64) error {
	return f.fcntl(syscall.F_WRLCK, fd, start, len)
}

func (f Fcntl) UnLock(fd uintptr, start, len int64) error {
	return f.fcntl(syscall.F_UNLCK, fd, start, len)
}

func (f Fcntl) fcntl(typ int16, fd uintptr, start, len int64) error {
	return syscall.FcntlFlock(fd, F_OFD_SETLKW, &syscall.Flock_t{
		Start:  start,
		Len:    len,
		Type:   typ,
		Whence: io.SeekStart,
	})
}

type Flock struct {
}

func (f Flock) ReadLock(fd uintptr, start, len int64) error {
	return f.flock(fd, syscall.LOCK_SH)
}

func (f Flock) WriteLock(fd uintptr, start, len int64) error {
	return f.flock(fd, syscall.LOCK_EX)
}

func (f Flock) UnLock(fd uintptr, start, len int64) error {
	return f.flock(fd, syscall.LOCK_UN)
}

func (f Flock) flock(fd uintptr, how int) error {
	return syscall.Flock(int(fd), how)
}
