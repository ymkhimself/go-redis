package main

import (
	"golang.org/x/sys/unix"
	"log"
	"time"
)

/**
KE 时间循环是干啥呢？
KeWait：拿到就绪的FilEvent和TimeEvent
KeProcess：将就绪的两类事件拿去执行。
*/

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

func GetMsTime() int64 {
	return time.Now().UnixMilli()
}

func (loop *KeLoop) AddTimeEvent(mask TeType, interval int64, proc TimeProc, extra interface{}) int {
	id := loop.timeEventNextId
	loop.timeEventNextId++
	var te KeTimeEvent
	te.id = id
	te.mask = mask
	te.internal = interval
	te.when = GetMsTime() + interval
	te.proc = proc
	te.extra = extra
	te.next = loop.TimeEvents // 采用头插法
	loop.TimeEvents = &te
	return id
}

// 删除TimeEvent,一个简单的从链表中的删除操作。
func (loop *KeLoop) RemoveTimeEvent(id int) {
	p := loop.TimeEvents
	var pre *KeTimeEvent
	for p != nil {
		if p.id == id {
			if pre == nil {
				loop.TimeEvents = p.next
			} else {
				pre.next = p.next
			}
		}
		pre = p
		p = p.next
	}
}

func KeLoopCreate() (*KeLoop, error) {
	epollFd, err := unix.EpollCreate1(0) // 简单创建一个epoll实例
	if err != nil {
		return nil, err
	}
	return &KeLoop{
		FileEvents:      make(map[int]*KeFileEvent),
		fileEventFd:     epollFd,
		timeEventNextId: 1,
		stop:            false,
	}, nil
}

// 找到最近的时间
func (loop *KeLoop) nearestTime() int64 {
	nearest := GetMsTime() + 1000
	p := loop.TimeEvents
	for p != nil {
		if p.when < nearest {
			nearest = p.when
		}
		p = p.next
	}
	return nearest
}

func (loop *KeLoop) KeWait() (tes []*KeTimeEvent, fes []*KeFileEvent) {
	timeout := loop.nearestTime() - GetMsTime()
	if timeout <= 0 {
		timeout = 10
	}
	var events [128]unix.EpollEvent
	n, err := unix.EpollWait(loop.fileEventFd, events[:], int(timeout)) // 如果timeout时间内还没有就绪，就要返回了，不能耽误TimeEvent
	if err != nil {
		log.Printf("epoll wait warnning: %v\n", err)
	}
	if n > 0 {
		log.Printf("ae get %v epoll events\n", n)
	}

	// 手机FileEvent
	for i := 0; i < n; i++ {
		if events[i].Events&unix.EPOLLIN != 0 {
			fe := loop.FileEvents[getFeKey(int(events[i].Fd), KE_READABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
		if events[i].Events&unix.EPOLLOUT != 0 {
			fe := loop.FileEvents[getFeKey(int(events[i].Fd), KE_WRITABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
	}

	now := GetMsTime()
	p := loop.TimeEvents
	for p != nil {
		if p.when <= now {
			tes = append(tes, p)
		}
		p = p.next
	}
	return
}

func (loop *KeLoop) KeProcess(tes []*KeTimeEvent, fes []*KeFileEvent) {
	for _, te := range tes {
		te.proc(loop, te.id, te.extra)
		if te.mask == KE_ONCE {
			loop.RemoveTimeEvent(te.id)
		} else {
			te.when = GetMsTime() + te.internal
		}
	}
	if len(fes) > 0 {
		log.Println("ke is processiong file events")
		for _, fe := range fes {
			fe.proc(loop, fe.fd, fe.extra)
		}
	}
}

func (loop *KeLoop) KeMain() {
	for loop.stop != true {
		tes, fes := loop.KeWait()
		loop.KeProcess(tes, fes)
	}
}
