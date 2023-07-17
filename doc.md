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

# 一些问题

## unix包在windows下报错

当使用unix包的时候，在windows下只有很少的函数可以用，这是为什么呢？
因为你的OS是windwos，默认认为你的代码是要构建到windows下跑。
调成linux之后，就可以了。  setting -> Go -> Build Tags

