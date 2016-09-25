package main

import (
	"os"

	log "github.com/Sirupsen/logrus"

	"github.com/gitbufenshuo/idob"
	"github.com/spf13/viper"
)

func main() {
	loadConfigFile() // do this yourself
	stoptest1 := test1()
	stoptest2 := test2()
	<-stoptest1
	<-stoptest2
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
}

func test1() chan struct{} {
	b := idob.NewBundle("bufenshuo")
	b.ConfigInputHandler(hfuncInput)
	b.ConfigOutputHandler(hfuncOutput)
	b.ConfigDealCachePolicy(0, tocache, onev)
	b.ConfigDealTimerTask(0, "logInterval", timetask)
	return b.Start()
}
func test2() chan struct{} {
	b := idob.NewBundle("bufenshuo")
	b.ConfigInputHandler(hfuncDonothing)
	b.ConfigOutputHandler(hfuncDonothing)
	b.ConfigDealCachePolicy(0, tocacheJoin, onevicted)
	b.ConfigDealTimerTask(0, "syncInterval", TaskSync)
	b.ConfigDealTimerTask(0, "logInterval", TaskLog)

	return b.Start()
}
