package main

import (
	"errors"
	"math"
)

const (
	DEFAULT_STEP int   = 1
	INIT_SIZE    int64 = 8
	FORCE_RATIO  int64 = 2 // 扩容时的负载因子下限
	GROW_RATIO   int64 = 2 // 扩容的增长率
)

var (
	EP_ERR = errors.New("expand error")
	EX_ERR = errors.New("key exists error")
	NK_ERR = errors.New("key doesnt exist error")
)

// 包含两个方法，求哈希和比较
type DictType struct {
	HashFunc  func(key *Gobj) int64
	EqualFunc func(k1, k2 *Gobj) bool
}

type Entry struct {
	Key  *Gobj
	Val  *Gobj
	next *Entry
}

type htable struct {
	table []*Entry
	size  int64
	mask  int64
	used  int64
}

type Dict struct {
	DictType
	hts       [2]*htable
	rehashidx int64
}

func DictCreate(dictType DictType) *Dict {
	var dict Dict
	dict.DictType = dictType
	dict.rehashidx = -1 // 先设为-1
	return &dict
}

// 是否在rehash
func (dict *Dict) isRehashing() bool {
	return dict.rehashidx != -1
}

func (dict *Dict) rehashStep() {
	dict.rehash(DEFAULT_STEP)
}

/*
所谓rehash
就是要将字典中的键值对重新分布到新的哈希表里面
1. 只要step大于0，就一直进行
2. 如果hts[0].used == 0 说明这个表没有被使用，这个表已经迁移完了，可以直接赋值了.
3. 找到还没迁移的槽
4. 遍历这个槽(链地址法)
5. 根据mask找到这个entry的再hts[1]中的槽
6. 一次rehash只处理一个槽
7. 通过stop的值来调整rehash的槽位数量
*/
func (dict *Dict) rehash(step int) {
	if step > 0 {
		if dict.hts[0].used == 0 {
			dict.hts[1] = dict.hts[0]
			dict.hts[1] = nil
			dict.rehashidx = -1
			return
		}
		for dict.hts[0].table[dict.rehashidx] == nil {
			dict.rehashidx++
		}
		entry := dict.hts[0].table[dict.rehashidx]
		for entry != nil {
			ne := entry
			idx := dict.HashFunc(entry.Key) & dict.hts[1].mask
			entry.next = dict.hts[1].table[idx] // 头插法
			dict.hts[1].table[idx] = entry
			dict.hts[0].used--
			dict.hts[1].used++
			entry = ne
		}
		dict.hts[0].table[dict.rehashidx] = nil
		dict.rehashidx++
		step--
	}
}

/*
找到大于等于size的最小的2的幂
*/
func nextPower(size int64) int64 {
	for i := INIT_SIZE; i < math.MaxInt64; i *= 2 {
		if i >= size {
			return i
		}
	}
	return -1
}

/*
扩容
1. 根据size拿到一个2的幂次的值,作为下一次的size
2.
*/
func (dict *Dict) expand(size int64) error {
	sz := nextPower(size)
	if dict.isRehashing() || (dict.hts[0] != nil && dict.hts[0].size >= sz) {
		return EP_ERR
	}
	var ht htable
	ht.size = sz
	ht.mask = sz - 1
	ht.used = 0
	ht.table = make([]*Entry, sz)
	// 检查是不是在初始状态
	if dict.hts[0] == nil {
		dict.hts[0] = &ht
		return nil
	}
	dict.hts[1] = &ht
	dict.rehashidx = 0
	return nil
}

/*
是否需要扩容
如果要扩容，就扩喽.
扩容不是一个很费事的操作
*/
func (dict *Dict) expandIfNeeded() error {
	if dict.isRehashing() { // 如果正在 rehash 直接返回
		return nil
	}
	if dict.hts[0] == nil { // 初始状态
		return dict.expand(INIT_SIZE)
	}
	// 负载因子到位，可以扩容
	if (dict.hts[0].used > dict.hts[0].size) && (dict.hts[0].used/dict.hts[0].size > FORCE_RATIO) {
		return dict.expand(dict.hts[0].size * GROW_RATIO)
	}
	return nil
}

/*
找一个空闲槽位
1. 如果要扩容，先扩容
2. 遍历两个表，去找一个合适的槽
3. 先拿到idx，然后拿到entry
4. 然后拿到遍历entry所在的槽，如果key已经在里面了，直接返回-1
5. 如果没有在rehash，那么找第一个table就够了。
*/
func (dict *Dict) keyIndex(key *Gobj) int64 {
	err := dict.expandIfNeeded()
	if err != nil {
		return -1
	}
	h := dict.HashFunc(key)
	var idx int64
	for i := 0; i <= 1; i++ {
		idx = h & dict.hts[i].mask
		e := dict.hts[i].table[idx]
		for e != nil {
			if dict.EqualFunc(e.Key, key) {
				return -1
			}
			e = e.next
		}
		if !dict.isRehashing() { // 如果没有在rehash,那么找第一个表就够了
			break
		}
	}
	return idx
}

/*
 */
func (dict *Dict) AddRaw(key *Gobj) *Entry {
	if dict.isRehashing() {
		dict.rehashStep()
	}
	index := dict.keyIndex(key)
	if index == -1 {
		return nil
	}

	var ht *htable
	if dict.isRehashing() { // 如果正在rehash，就会王第二个表里插
		ht = dict.hts[1]
	} else {
		ht = dict.hts[0]
	}
	var e Entry
	e.Key = key
	key.IncrRefCount()
	e.next = ht.table[index]
	ht.table[index] = &e
	ht.used++
	return &e

}

func (dict *Dict) Add(key, val *Gobj) error {
	entry := dict.AddRaw(key)
	if entry != nil {
		return EX_ERR
	}
	entry.Val = val
	val.IncrRefCount()
	return nil
}

func (dict *Dict) Set(key, val *Gobj) {
	err := dict.Add(key, val)
	if err == nil { // 如果key不存在，并且已经设置好了,直接返回
		return
	}
	entry := dict.Find(key) // 如果key存在，那就重新设置一下
	entry.Val.DecrRefCount()
	entry.Val = val
}

func (dict *Dict) Delete(key *Gobj) error {

}

/*
释放元素
*/
func freeEntry(e *Entry) {
	e.Key.DecrRefCount()
	e.Val.DecrRefCount()
}

func (dict *Dict) Find(key *Gobj) *Entry {

}

func (dict *Dict) Get(key *Gobj) *Gobj {

}

/*
随机拿一个
*/
func (dict *Dict) RandomGet() *Entry {

}
