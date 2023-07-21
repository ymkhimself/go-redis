package main

import (
	"golang.org/x/sys/unix"
	"log"
)

const BACKLOG int = 64

func Accept(fd int) (int, error) {
	// 忽略掉了客户端的地址
	nfd, _, err := unix.Accept(fd)
	return nfd, err
}

func TcpServer(port int) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0) // SOCK_STREAM TCP连接
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	err = unix.SetsockoptInt(s, unix.SOL_SOCKET, unix.SO_REUSEPORT, port)
	if err != nil {
		log.Printf("set SO_REUSEPORT err: %v\n", err)
		unix.Close(s)
		return -1, nil
	}
	var addr unix.SockaddrInet4
	addr.Port = port
	unix.Bind(s, &addr)
	if err != nil {
		log.Printf("bind addr err: %v\n", err)
		unix.Close(s)
		return -1, nil
	}
	err = unix.Listen(s, BACKLOG)
	if err != nil {
		log.Printf("listen socket err: %v\n", err)
		unix.Close(s)
		return -1, err
	}
	return s, nil
}
