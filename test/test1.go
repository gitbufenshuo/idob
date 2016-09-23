package main

import (
	"bytes"
	"fmt"
	"sync"

	cache "github.com/khowarizmi/go-clru"
)

func hfunc(msgBody *bytes.Buffer) error {
	msgBody.WriteByte('~')
	return nil
}
func tocache(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool) {

}
func onev(entry *cache.Entry, msgOut *chan *bytes.Buffer, pooll *sync.Pool) {

}
func timetask(c *cache.CLRU, wgTimerTask *sync.WaitGroup) {
	fmt.Println("time task")
	(*wgTimerTask).Done() // Caution, do please call this
}
