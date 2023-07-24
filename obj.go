package main

import "strconv"

type Gtype uint8

const (
	GSTR  Gtype = 0x00
	GLIST Gtype = 0x01
	GSET  Gtype = 0x02
	GZSET Gtype = 0x03
)

type Gval interface{}

type Gobj struct {
	Type     Gtype
	Val      Gval
	refCount int // 用于引用计数
}

func (o *Gobj) InitVal() int64 {
	if o.Type != GSTR {
		return 0
	}
	val, _ := strconv.ParseInt(o.Val.(string), 10, 64)
	return val
}

func (o *Gobj) StrVal() string {
	if o.Type != GSTR {
		return ""
	}
	return o.Val.(string)
}
