package main

import (
	"errors"
	"math"
)

const (
	DEFAULT_STEP int   = 1
	INIT_SIZE    int64 = 8
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
*/
func (dict *Dict) expandIfNeeded() error {

}

/*
 */
func (dict *Dict) keyIndex(key *Gobj) int64 {

}

/*
 */
func (dict *Dict) AddRaw(key *Gobj) *Entry {

}

func (dict *Dict) Add(key, val *Gobj) error {

}

func (dict *Dict) Set(key, val *Gobj) {

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
