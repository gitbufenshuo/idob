package main

import (
	"bytes"
	"fmt"

	"os"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/gitbufenshuo/idob"
	cache "github.com/khowarizmi/go-clru"
	"github.com/spf13/viper"
)

func main() {
	loadConfigFile()
	b := idob.NewBundle("bufenshuo")
	b.ConfigInputHandler(hfunc)
	b.ConfigOutputHandler(hfunc)
	b.ConfigDealCachePolicy(0, tocache, onev)
	b.ConfigDealTimerTask(0, "logInterval", timetask)
	<-b.Start()
	os.Exit(0)
}
func loadConfigFile() {
	file, err := os.Open("./config.json")
	defer file.Close()
	if err != nil {
		log.Fatalln(err)
	}
	viper.SetConfigType("json")
	viper.ReadConfig(file)
	// print(viper.Get("bufenshuo.#input"))
}
func hfunc(msgBody *bytes.Buffer) {
	msgBody.WriteByte('~')
}
func tocache(msg *bytes.Buffer, c *cache.CLRU, bufpool *sync.Pool) {

}
func onev(entry *cache.Entry, msgOut *chan *bytes.Buffer, pooll *sync.Pool) {

}
func timetask(c *cache.CLRU, wgTimerTask *sync.WaitGroup) {
	fmt.Println("time task")
	(*wgTimerTask).Done() // Caution, do please call this
}
