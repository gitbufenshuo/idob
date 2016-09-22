package idob

import (
	"bytes"
	"strconv"
	"sync"

	log "github.com/Sirupsen/logrus"
	cache "github.com/khowarizmi/go-clru"
	"github.com/spf13/viper"
)

type upstream interface {
	getMsgOut() (msgOut **chan *bytes.Buffer)
}
type downstream interface {
	getMsgIn() (msgIn **chan *bytes.Buffer)
}
type flow interface {
	start() chan struct{}
	getMsgOut() (msgOut **chan *bytes.Buffer)
	getMsgIn() (msgIn **chan *bytes.Buffer)
}
type pool interface {
	getBufPool() **sync.Pool
}

// ConfigTerminal terminal: input and output are terminals
type configTerminal interface {
	setHandler(hfunc func(msgBody *bytes.Buffer))
}

// ConfigDeal two things have to be done
// 1. how do msgs go into the cache
// 2. how do msgs get out of the cache (aka. onEvicted)
// 3. (optional) timertask.
type configDeal interface {
	setCachePolicy(msg2Cache func(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool), onEvicted func(entry *cache.Entry, msgOut *chan *bytes.Buffer, bufpool *sync.Pool))
	addTimerTask(interval string, task func(c *cache.CLRU, wgTimerTask *sync.WaitGroup))
}

// Bundle is the struct you will use
type Bundle struct {
	name    string
	confPre string
	flows   []flow
	flowmap map[string]interface{}
}

// NewBundle is the only correct way to new a Bundle
func NewBundle(name string) *Bundle {
	confPre := name + ".#bundle."
	var flows []flow
	b := &Bundle{
		name:    name,
		confPre: confPre,
		flows:   flows,
	}
	b.flowmap = make(map[string]interface{})
	b.initInput()
	b.initDeal()
	b.initOutput()

	b.bind()

	return b
}
func (b *Bundle) encubatePool() {
	var bufPool = &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

	for k := range b.flowmap {
		*(b.flowmap[k].(pool).getBufPool()) = bufPool
	}
}
func (b *Bundle) appendFlows(aflow flow) {
	b.flows = append(b.flows, aflow)
}
func (b *Bundle) bind() {
	chanbuflen := viper.Get(b.name + ".#bundle.#bind")
	if chanbuflen == nil {
		panic(`config field "` + b.name + `.#bundle.#bind" is gone`)
	}
	lenofmodules := len(b.flows)
	buflens := chanbuflen.([]interface{})
	numofbuf := len(buflens)
	if numofbuf != lenofmodules-1 {
		panic(`config field "` + b.name + `.#bundle.#bind" is not enough`)
	}

	for i := 0; i < numofbuf; i++ {
		b.bindFlows(int(buflens[i].(float64)), b.flows[i], b.flows[i+1])
	}
}
func (b *Bundle) bindFlows(msgLen int, up upstream, down downstream) {
	msgChan := make(chan *bytes.Buffer, msgLen)
	upmsg := up.getMsgOut()
	downmsg := down.getMsgIn()
	*upmsg = &msgChan
	*downmsg = &msgChan
}

// Start fires the whole thing
func (b *Bundle) Start() chan struct{} {
	sigStop := make(chan struct{})
	for idx := range b.flows {
		sigStop = b.flows[idx].start()
	}
	return sigStop
}

func (b *Bundle) initInput() {
	if viper.Get(b.name+".#input") == nil {
		panic(`config field "` + b.name + `.#input" is gone`)
	}
	in := newNsqInput(b.name)
	b.flowmap["#input"] = in
	b.appendFlows(in)
	log.Infoln("<i><i><i><i> input module init ok <i><i><i><i>")
}
func (b *Bundle) initDeal() {
	index := ""
	inum := 0
	for {
		if viper.Get(b.name+".#deal"+index) == nil {
			if index == "" {
				panic(`config field "` + b.name + `.#deal" is gone`)
			}
			break
		}
		d := newdeal(b.name)
		b.flowmap["#deal"+index] = d
		b.appendFlows(d)
		inum++
		index = strconv.Itoa(inum)
	}
	log.Infoln("<d><d><d><d> deal module init ok <d><d><d><d>")
}
func (b *Bundle) initOutput() {
	if viper.Get(b.name+".#output") == nil {
		panic(`config field "` + b.name + `.#output" is gone`)
	}
	o := newNsqOutput(b.name)
	b.flowmap["#output"] = o
	b.appendFlows(o)
	log.Infoln("<o><o><o><o> output module init ok <o><o><o><o>")
}

// ConfigInputHandler is
func (b *Bundle) ConfigInputHandler(hfunc func(msgBody *bytes.Buffer)) {
	b.flowmap["#input"].(configTerminal).setHandler(hfunc)
}

//ConfigOutputHandler is
func (b *Bundle) ConfigOutputHandler(hfunc func(msgBody *bytes.Buffer)) {
	b.flowmap["#output"].(configTerminal).setHandler(hfunc)
}

//ConfigDealCachePolicy is
func (b *Bundle) ConfigDealCachePolicy(idx int, msg2Cache func(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool), onEvicted func(entry *cache.Entry, msgOut *chan *bytes.Buffer, bufpool *sync.Pool)) {
	index := ""
	if idx != 0 {
		index = strconv.Itoa(idx)
	}
	b.flowmap["#deal"+index].(configDeal).setCachePolicy(msg2Cache, onEvicted)
}

//ConfigDealTimerTask is
func (b *Bundle) ConfigDealTimerTask(idx int, interval string, task func(c *cache.CLRU, wgTimerTask *sync.WaitGroup)) {
	index := ""
	if idx != 0 {
		index = strconv.Itoa(idx)
	}
	b.flowmap["#deal"+index].(configDeal).addTimerTask(interval, task)
}
