package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/pquerna/ffjson/ffjson"

	cache "github.com/khowarizmi/go-clru"
)

type Msg struct {
	Time int
	Pgid string
}
type Action struct {
	Time int
	Item *bytes.Buffer
}

type ByTime []Action

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Time < a[j].Time }

func hfuncDonothing(msgBody *bytes.Buffer) error {
	log.Infoln("<<< we get a msg in nsq>>>")
	return nil
}
func tocacheJoin(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool) {
	add2Cache := func(id string, k int, v *bytes.Buffer, c *cache.CLRU) {
		update := func(entry *cache.Entry) {
			item, ok := entry.Value.([]Action)
			if !ok {
				fmt.Println("convert item fail")
			} else {
				entry.Value = append(item, Action{k, v})
			}
		}
		c.AddOrUpdate(id, []Action{{k, v}}, update)
	}

	var m Msg
	err := ffjson.Unmarshal(msg.Bytes(), &m)
	if err != nil {
		fmt.Println("Parse json error", msg, string(msg.Bytes()), err)
	} else {
		add2Cache(m.Pgid, m.Time, msg, c)
	}
}
func onevicted(entry *cache.Entry, msgOut *chan *bytes.Buffer, pooll *sync.Pool) {
	pgid := entry.Key
	actions := entry.Value.([]Action)
	// length := len(actions)
	sort.Sort(ByTime(actions))

	// remove msg not contains "init"
	buf := actions[0].Item
	hasInit := strings.Contains(buf.String(), "type\":\"init")
	if !hasInit {
		buf.Reset()
		pooll.Put(buf)
		return
	}
	buf = pooll.Get().(*bytes.Buffer)
	buf.WriteString("BBB" + "{\"pgid\":\"")
	buf.WriteString(pgid)
	buf.WriteString("\",\"data\":[")

	for i, action := range actions {
		if i > 0 && bytes.Equal(action.Item.Bytes(), actions[i-1].Item.Bytes()) {
			continue
		}

		itemLen := len(action.Item.Bytes())
		if buf.Cap()-buf.Len() < 2*itemLen {
			buf.Grow(itemLen * 2)
		}
		if i != 0 {
			buf.WriteString(",")
		}
		buf.Write(action.Item.Bytes())
	}
	buf.WriteString("]}DDD")
	*msgOut <- buf
}
func TaskSync(c *cache.CLRU, wgTimerTask *sync.WaitGroup) {
	go func() {
		log.Infoln("syncing action series...")
		keys := c.Keys()
		for _, key := range keys {
			time.Sleep(50 * time.Millisecond)
			c.Get(key)
		}
		(*wgTimerTask).Done()
	}()
}
func TaskLog(c *cache.CLRU, wgTimerTask *sync.WaitGroup) {
	log.Infoln(c.Len())
	(*wgTimerTask).Done()
}
