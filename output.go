package idob

import (
	"bytes"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
)

type nsqOutput struct {
	name           string
	nums           int
	topic          string
	nsqd           string
	wgIn           sync.WaitGroup
	msgIn          *chan *bytes.Buffer
	handler        func(*bytes.Buffer)
	confPre        string
	sigOut         chan struct{}
	conf           *nsq.Config
	byteBufferPool *sync.Pool
}

func newNsqOutput(name string) *nsqOutput {
	confPre := name + ".#output."
	nsqoutput := nsqOutput{}
	nsqoutput.name = name
	nsqoutput.nsqd = viper.GetString(confPre + "#nsqd")
	nsqoutput.nums = viper.GetInt(confPre + "#nums")
	nsqoutput.topic = viper.GetString(confPre + "#topic")
	nsqoutput.conf = nsq.NewConfig()
	for k, v := range viper.GetStringMap(confPre + "#writer") {
		if err := nsqoutput.conf.Set(k, v); err != nil {
			log.Errorln("set writer config fail", err)
			panic(err)
		}
	}
	nsqoutput.sigOut = make(chan struct{})
	nsqoutput.confPre = confPre
	nsqoutput.wgIn = sync.WaitGroup{}
	return &nsqoutput
}

func (nsqoutput nsqOutput) setHandler(hfunc func(*bytes.Buffer)) {
	nsqoutput.handler = hfunc
}

func (nsqoutput *nsqOutput) start() chan struct{} {
	log.Infoln("----output msgin--->", *nsqoutput.msgIn)

	producer, err := nsq.NewProducer(nsqoutput.nsqd, nsqoutput.conf)
	if err != nil {
		log.Errorln("create producer fail", err)
		panic(err)
	}

	for i := 0; i < nsqoutput.nums; i++ {
		nsqoutput.wgIn.Add(1)
		go func() {
			ch := *nsqoutput.msgIn
			for {
				if buf, ok := <-ch; ok {
					nsqoutput.handler(buf)
					producer.Publish(nsqoutput.topic+"#ephemeral", buf.Bytes())
					nsqoutput.byteBufferPool.Put(buf)
				} else {
					break
				}
			}
			nsqoutput.wgIn.Done()
		}()
	}
	go func() {
		nsqoutput.wgIn.Wait()
		close(nsqoutput.sigOut)
		log.Infoln("<o><o><o><o> output module stopped <o><o><o><o>")
	}()
	return nsqoutput.sigOut
}

func (nsqoutput *nsqOutput) getMsgIn() (msgIn **chan *bytes.Buffer) {
	return &nsqoutput.msgIn
}
func (nsqoutput *nsqOutput) getMsgOut() (msgIn **chan *bytes.Buffer) {
	return nil
}
func (nsqoutput *nsqOutput) getBufPool() **sync.Pool {
	return &nsqoutput.byteBufferPool
}
