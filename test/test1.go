package main

import (
	"bytes"
	"fmt"
	"sync"

	"errors"

	cache "github.com/gitbufenshuo/go-clru"
)

func hfuncInput(msgBody *bytes.Buffer) error {
	if (msgBody.Bytes())[0] == 'a' {
		return errors.New("")
	}
	return nil
}
func hfuncOutput(msgBody *bytes.Buffer) error {
	if (msgBody.Bytes())[0] == 'b' {
		return errors.New("")
	}
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
