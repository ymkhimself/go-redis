package main

import (
	"golang.org/x/sys/unix"
	"log"
)

type FeType int
type TeType int

const (
	KE_READABLE FeType = 1
	KE_WRITABLE FeType = 2
)

const (
	KE_NORMAL TeType = 1
	KE_ONCE   TeType = 2
)

type FileProc func(loop *KeLoop, fd int, extra interface{}) // FileEvent的回调函数
type TimeProc func(loop *KeLoop, id int, extra interface{}) // TimeEvent的回调函数

type KeFileEvent struct {
	fd    int
	mask  FeType
	proc  FileProc
	extra interface{}
}

type KeTimeEvent struct {
	id       int
	mask     TeType
	when     int64
	internal int64
	proc     TimeProc
	extra    interface{}
	next     *KeTimeEvent
}

type KeLoop struct {
	FileEvents      map[int]*KeFileEvent // TODO 关于为什么从用map不用list，为什么不用dict，而用内置map
	TimeEvents      *KeTimeEvent
	fileEventFd     int
	timeEventNextId int
	stop            bool
}

// 根据类型，确定一个Fe的Key   fd+mask 确定唯一的一个FileEvent
func getFeKey(fd int, mask FeType) int {
	if mask == KE_READABLE {
		return fd
	} else {
		return fd * -1
	}
}

var fe2ep = [3]uint32{0, unix.EPOLLIN, unix.EPOLLOUT} // TODO EOPLLIN  EOPLLOUT

func (loop *KeLoop) getEpollMask(fd int) uint32 {
	var ev uint32
	// 如果已经注册过读事件了
	if loop.FileEvents[getFeKey(fd, KE_READABLE)] != nil {
		ev |= fe2ep[KE_READABLE]
	}
	if loop.FileEvents[getFeKey(fd, KE_WRITABLE)] != nil {
		ev |= fe2ep[KE_WRITABLE]
	}
	return ev
}

// 添加文件事件
func (loop *KeLoop) AddFileEvent(fd int, mask FeType, proc FileProc, extra interface{}) {
	// 拿到epoll ctl
	ev := loop.getEpollMask(fd)
	if ev&fe2ep[mask] != 0 {
		// 如果事件已经注册过
		return
	}
	op := unix.EPOLL_CTL_ADD
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	ev |= fe2ep[mask]
	err := unix.EpollCtl(loop.fileEventFd, op, fd, &unix.EpollEvent{Fd: int32(fd), Events: ev})
	if err != nil {
		log.Printf("epoll ctl error: %v\n", err)
		return
	}

	// 添加到事件循环中去。
	var fe KeFileEvent
	fe.fd = fd
	fe.mask = mask
	fe.proc = proc
	fe.extra = extra
	loop.FileEvents[getFeKey(fd, mask)] = &fe
	log.Printf("ke add file event fd:%v, mask:%v\n", fd, mask)
}

// 移除文件事件
func (loop *KeLoop) RemoveFileEvent(fd int, mask FeType) {
	// epoll ctl
	op := unix.EPOLL_CTL_DEL
	ev := loop.getEpollMask(fd)
	ev &= ^fe2ep[mask] // 取反再与，把操作摘出来
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	err := unix.EpollCtl(loop.fileEventFd, op, fd, &unix.EpollEvent{Fd: int32(fd), Events: ev})
	if err != nil {
		log.Printf("epoll del error:%v\n", ev)
	}
	loop.FileEvents[getFeKey(fd, mask)] = nil
	log.Printf("ae remove file event fd:%v,mask:%v\n", fd, mask)
}
