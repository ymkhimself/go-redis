package main

type Node struct {
	Val  *Gobj
	pre  *Node
	next *Node
}

// 链表类型
type ListType struct {
	EqualFunc func(a, b *Gobj) bool
}

// List数据结构 双向链表
type List struct {
	ListType
	head   *Node
	tail   *Node
	length int
}

// 链表创建
func ListCreate(listType ListType) *List {
	var list List
	list.ListType = listType
	return &list
}

func (list *List) Length() int {
	return list.length
}

func (list *List) First() *Node {
	return list.head
}

func (list *List) Last() *Node {
	return list.tail
}

func (list *List) Find(val *Gobj) *Node {
	t := list.head
	for t != nil {
		if list.EqualFunc(t.Val, val) {
			break
		}
		t = t.next
	}
	return t
}

// 在尾部加
func (list *List) Append(val *Gobj) {
	var n Node
	n.Val = val
	if list.head == nil {
		list.head = &n
		list.tail = &n
	} else {
		n.pre = list.tail
		list.tail.next = &n
		list.tail = list.tail.next
	}
	list.length++
}

// 在头部加
func (list *List) Lpush(val *Gobj) {
	var n Node
	n.Val = val
	if list.head == nil {
		list.head = &n
		list.tail = &n
	} else {
		list.head.pre = &n
		n.next = list.head
		list.head = &n
	}
	list.length++
}

func (list *List) DelNode(n *Node) {
	if n == nil {
		return
	}
	if n == list.head { // 删除的是头节点
		if n.next != nil {
			n.next.pre = nil
		}
		list.head = n.next
		n.next = nil
	} else if n == list.tail { // 删除的是尾节点。
		if n.pre != nil {
			n.pre.next = nil
		}
		list.tail = n.pre
		n.pre = nil
	} else { // 删除的是中间节点
		if n.pre != nil {
			n.pre.next = n.next
		}
		if n.next != nil {
			n.next.pre = n.pre
		}
		n.pre = nil
		n.next = nil
	}
	list.length--
}

func (list *List) Delete(val *Gobj) {
	list.DelNode(list.Find(val))
}
