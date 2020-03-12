package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/recruit-mp/kangol/internal"
	"os"
)

const version = "0.2.8"

func main() {
	app := internal.NewApp(version)
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err.Error())
	}
}
