package main

import (
	"blockchain-oracle/rollup"
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/sethvargo/go-envconfig"
	log "github.com/sirupsen/logrus"
)

func main() {

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{})

	var cfg rollup.Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		log.Fatal(err)
	}
	log.Debugf("Read config from env: %+v\n", cfg)
	cfg.SeqPrivate = "00fd4d6af5ac34d29d63a04ecf7da1ccfcbcdf7f7ed4042b8975e1c54e96d685"

	ethBlockData := make(chan rollup.EthBlockData)
	shutdownSignal := make(chan bool)
	cl := rollup.NewChainListeners("https://beacon-nd-942-489-268.p2pify.com/c450ba1e6c5025d33dd14dc4c54f5cf6", ethBlockData, shutdownSignal)

	app := rollup.NewApp(cfg, ethBlockData)

	fmt.Println("Running chain listeners in background!!")
	go cl.Run()

	fmt.Println("Running app!!")
	app.Run()

	// wait for ctrl + c stuff. this blocks the parent process too which is nice
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// tell all apps to shutdown gracefully
	shutdownSignal <- true
}
