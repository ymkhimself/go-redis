package main

import (
	"golang.org/x/sys/unix"
	"log"
)

const BACKLOG int = 64

// 接收连接
func Accept(fd int) (int, error) {
	// 忽略掉了客户端的地址
	nfd, _, err := unix.Accept(fd)
	return nfd, err
}

func Connect(host [4]byte, port int) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	var addr unix.SockaddrInet4
	addr.Addr = host
	addr.Port = port
	if err != nil {
		log.Printf("connect err: %v\n", err)
		return -1, err
	}
	return s, nil
}

func Read(fd int, buf []byte) (int, error) {
	return unix.Read(fd, buf)
}

func Write(fd int, buf []byte) (int, error) {
	return unix.Write(fd, buf)
}

func Close(fd int) {
	unix.Close(fd)
}

func TcpServer(port int) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0) // SOCK_STREAM TCP连接
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	err = unix.SetsockoptInt(s, unix.SOL_SOCKET, unix.SO_REUSEPORT, port) // TODO 不懂这里的SO_REUSEPORT，和SO_REUSEADDR有什么区别
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
