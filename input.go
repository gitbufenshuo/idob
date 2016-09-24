package idob

import (
	"bytes"
	syslog "log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
)

// NsqInput is a base input of nsq
type nsqInput struct {
	name               string
	nums               int
	topic              string
	channel            string
	nsqLookupd         string
	concurrentHandlers int
	maxInFlight        int
	consumers          []*nsq.Consumer
	confs              []*nsq.Config
	handler            func(*bytes.Buffer) error
	msgOut             *chan *bytes.Buffer
	confPre            string
	sigStop            bool
	byteBufferPool     *sync.Pool
}

func newNsqInput(name string) *nsqInput {
	confPre := name + ".#input."
	nsqinput := nsqInput{}
	nsqinput.name = name
	nsqinput.nums = viper.GetInt(confPre + "#nums")
	nsqinput.topic = viper.GetString(confPre + "#topic")
	nsqinput.channel = viper.GetString(confPre + "#channel")
	nsqinput.maxInFlight = viper.GetInt(confPre + "#maxInFlight")
	nsqinput.concurrentHandlers = viper.GetInt(confPre + "#concurrentHandlers")
	nsqinput.nsqLookupd = viper.GetString(confPre + "#nsqLookupd")
	nsqinput.confs = make([]*nsq.Config, nsqinput.nums)
	for idx := range nsqinput.confs {
		conf := nsq.NewConfig()
		for k, v := range viper.GetStringMap(confPre + "#reader") {
			if err := conf.Set(k, v); err != nil {
				log.Errorln("set reader config fail", err)
				panic(err)
			}
		}
		nsqinput.confs[idx] = conf
	}
	nsqinput.sigStop = viper.GetBool(confPre + "#sigStop")
	return &nsqinput
}

func (nsqinput *nsqInput) HandleMessage(message *nsq.Message) error {
	buf := nsqinput.byteBufferPool.Get().(*bytes.Buffer)
	buf.Write(message.Body)
	if dropit := nsqinput.handler(buf); dropit != nil {
		buf.Reset()
		nsqinput.byteBufferPool.Put(buf)
		return nil
	}
	*nsqinput.msgOut <- buf
	return nil
}

func (nsqinput *nsqInput) setHandler(hfunc func(*bytes.Buffer) error) {
	nsqinput.handler = hfunc
}
func (nsqinput *nsqInput) start() chan struct{} {
	nsqinput.consumers = make([]*nsq.Consumer, nsqinput.nums)
	for idx := range nsqinput.consumers {
		consumer, err := nsq.NewConsumer(nsqinput.topic+"#ephemeral", nsqinput.channel+"#ephemeral", nsqinput.confs[idx])
		if err != nil {
			log.Errorln("create consumer fail", err)
			panic(err)
		}
		logger := syslog.New(os.Stdout, "nsq>", syslog.Flags())
		consumer.SetLogger(logger, nsq.LogLevelError)
		consumer.AddConcurrentHandlers(nsqinput, nsqinput.concurrentHandlers)
		consumer.ChangeMaxInFlight(nsqinput.maxInFlight)
		if err := consumer.ConnectToNSQLookupd(nsqinput.nsqLookupd); err != nil {
			log.Errorln("connect to NSQLookupd fail", err)
			panic(err)
		}
		nsqinput.consumers[idx] = consumer
	}
	if nsqinput.sigStop {
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan
			nsqinput.stop()
			log.Infoln("<i><i><i><i> input module stopped ok <i><i><i><i>")
		}()
	}
	return nil
}
func (nsqinput *nsqInput) stop() {
	for idx := range nsqinput.consumers {
		nsqinput.consumers[idx].Stop()
	}
	for idx, consumer := range nsqinput.consumers {
		_ = <-consumer.StopChan
		log.Infof("consumer %d stopped.", idx)
	}
	log.Infoln("all consumer stopped")
	close(*nsqinput.msgOut)
}

func (nsqinput *nsqInput) getMsgOut() (msgOut **chan *bytes.Buffer) {
	return &nsqinput.msgOut
}
func (nsqinput *nsqInput) getMsgIn() (msgOut **chan *bytes.Buffer) {
	return nil
}
func (nsqinput *nsqInput) getBufPool() **sync.Pool {
	return &nsqinput.byteBufferPool
}
