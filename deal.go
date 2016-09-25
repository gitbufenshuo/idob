package idob

import (
	"bytes"
	"reflect"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	cache "github.com/khowarizmi/go-clru"
	"github.com/spf13/viper"
)

type timerTask struct {
	dtime time.Duration
	task  func(c *cache.CLRU, wgTimerTask *sync.WaitGroup)
}

type deal struct {
	name           string
	nums           int
	c              *cache.CLRU
	msgIn          *chan *bytes.Buffer
	msgOut         *chan *bytes.Buffer
	wgIn           sync.WaitGroup
	msg2Cache      func(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool)
	onEvicted      func(entry *cache.Entry, msgOut *chan *bytes.Buffer, bufpool *sync.Pool)
	timertask      []timerTask
	timerStop      chan struct{}
	wgTimerTask    sync.WaitGroup
	confPre        string
	byteBufferPool *sync.Pool
}

func newdeal(name string) *deal {
	confPre := name + ".#deal."
	c := cache.New(viper.GetInt(confPre+"#cacheSize"), viper.GetDuration(confPre+"#cacheTTL"))
	var wgIn sync.WaitGroup
	var wgTimerTask sync.WaitGroup
	d := &deal{
		c:           c,
		wgIn:        wgIn,
		wgTimerTask: wgTimerTask,
	}
	d.name = name
	d.nums = viper.GetInt(confPre + "#nums")
	d.timertask = make([]timerTask, 0)
	d.confPre = confPre
	d.timerStop = make(chan struct{})
	return d
}

func (d *deal) setCachePolicy(msg2Cache func(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool), onEvicted func(entry *cache.Entry, msgOut *chan *bytes.Buffer, bufpool *sync.Pool)) {
	d.msg2Cache = msg2Cache
	d.onEvicted = onEvicted
	d.c.OnEvicted = func(entry *cache.Entry) {
		d.onEvicted(entry, d.msgOut, d.byteBufferPool)
	}
}

func (d *deal) addTimerTask(confField string, task func(c *cache.CLRU, wgTimerTask *sync.WaitGroup)) {
	timertask := timerTask{}
	if viper.Get(d.confPre+confField) == nil {
		panic("no config field " + d.confPre + confField)
	}
	timertask.dtime = viper.GetDuration(d.confPre + confField)
	timertask.task = task
	d.timertask = append(d.timertask, timertask)
}
func (d *deal) runTimerTask() {
	numOfTasks := len(d.timertask)
	selectCase := make([]reflect.SelectCase, numOfTasks)
	for idx := range selectCase {
		selectCase[idx].Dir = reflect.SelectRecv
		selectCase[idx].Chan = reflect.ValueOf(time.Tick(d.timertask[idx].dtime))
	}
	stopCase := reflect.SelectCase{}
	stopCase.Dir = reflect.SelectRecv
	stopCase.Chan = reflect.ValueOf(d.timerStop)
	selectCase = append(selectCase, stopCase)
	d.wgTimerTask.Add(1)
	go func() {
		for {
			chosen, _, _ := reflect.Select(selectCase)
			if chosen == numOfTasks {
				break
			}
			d.wgTimerTask.Add(1)

			d.timertask[chosen].task(d.c, &d.wgTimerTask)
		}
		d.wgTimerTask.Done()

	}()
}

func (d *deal) getMsgIn() (msgIn **chan *bytes.Buffer) {
	return &d.msgIn
}

func (d *deal) getMsgOut() (msgOut **chan *bytes.Buffer) {
	return &d.msgOut
}
func (d *deal) getBufPool() **sync.Pool {
	return &d.byteBufferPool
}

func (d *deal) start() chan struct{} {

	for i := 0; i < d.nums; i++ {
		d.wgIn.Add(1)
		go func() {
			ch := *d.msgIn
			for {
				if msg, ok := <-ch; ok {
					d.msg2Cache(msg, d.c, d.byteBufferPool)
				} else {
					break
				}
			}
			d.wgIn.Done()
		}()
	}

	d.runTimerTask()

	go func() {
		d.wgIn.Wait()
		d.c.Flush()
		close(d.timerStop)
		d.wgTimerTask.Wait()
		close(*d.msgOut)
		log.Infoln("<d><d><d><d> deal module stopped ok <d><d><d><d>")
	}()
	return nil
}
