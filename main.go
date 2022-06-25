package main

import (
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	a := App{}
	a.Initialize()
	a.Run()
}
