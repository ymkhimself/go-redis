package main

const (
	DEFAULT_STEP int   = 1
	INT_SIZE     int64 = 8
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

func (dict *Dict) rehash(step int) {

}

func (dict *Dict) nextPower(size int64) int64 {

}

func (dict *Dict) expand(size int64) error {

}

func (dict *Dict) expandIfNeeded() error {

}

func (dict *Dict) keyIndex(key *Gobj) int64 {

}
