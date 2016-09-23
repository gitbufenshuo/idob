package main

import (
	"os"

	log "github.com/Sirupsen/logrus"

	"github.com/gitbufenshuo/idob"
	"github.com/spf13/viper"
)

func main() {
	loadConfigFile() // do this yourself
	test2()
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

func test1() {
	b := idob.NewBundle("bufenshuo")
	b.ConfigInputHandler(hfunc)
	b.ConfigOutputHandler(hfunc)
	b.ConfigDealCachePolicy(0, tocache, onev)
	b.ConfigDealTimerTask(0, "logInterval", timetask)
	<-b.Start()
	os.Exit(0)
}
func test2() {
	b := idob.NewBundle("bufenshuo")
	b.ConfigInputHandler(hfuncDonothing)
	b.ConfigOutputHandler(hfuncDonothing)
	b.ConfigDealCachePolicy(0, tocacheJoin, onevicted)
	b.ConfigDealTimerTask(0, "logInterval", TaskSync)
	b.ConfigDealTimerTask(0, "logInterval", TaskLog)

	<-b.Start()
	os.Exit(0)
}
