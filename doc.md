# main



# ke

ke
kk event loop 事件循环

keLoop是最核心的数据结构。

KeLoop包括些什么呢？
- 两个类型的事件  文件事件和时间事件
- 当前文件事件的fd
- 下一个事件事件的id
- 是否停止

## 有两个类型的事件：
### fileEvent
- 一个fd
- 一个mask ，表示类型：读和写
- proc 事件的执行，一个回调函数
- extra  额外的东西


- getFeKey : 通过mask和fd，去fileEvent的map里找到为一个FileEvent
- AddFileEvent：添加文件事件
- RemoveFileEvent：移除文件事件




### timeEvent
- id
- mask,表示类型，普通还是单次的
- when   啥时候执行
- interval



# net

处理网络相关

# obj

redis object

# DS

## dict

## list

## zset


# resp
RESP（REdis Serialization Protocol）是Redis使用的一种序列化协议。它是一种简单且高效的文本协议，用于在Redis客户端和服务器之间进行通信。

RESP协议的设计目标是简单、可读性强，并且可以被多种编程语言轻松地解析和生成。它以行为单位进行通信，每个请求或响应都由一个或多个字节序列组成。

RESP协议的基本规则如下：
- 请求和响应都以行结束符（\r\n）作为分隔符。
- 请求的第一个字节是命令类型，后面是参数。
- 响应的第一个字节表示响应类型，后面是具体的数据。

RESP协议支持多种数据类型，包括简单字符串、错误信息、整数、大整数、浮点数、数组和空值。每种类型都有对应的前缀字符来标识。

通过使用RESP协议，Redis客户端可以向服务器发送命令请求，并接收服务器返回的响应。这种简单而高效的协议设计使得Redis在处理大量请求时表现出色，并且可以被广泛地应用于各种应用场景。


# 一些问题

## unix包在windows下报错

当使用unix包的时候，在windows下只有很少的函数可以用，这是为什么呢？
因为你的OS是windwos，默认认为你的代码是要构建到windows下跑。
调成linux之后，就可以了。  setting -> Go -> Build Tags



