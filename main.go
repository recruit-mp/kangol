package main

import (
	"flag"
	"fmt"
	"time"

	"./awsecs"
	"./task"
	log "github.com/Sirupsen/logrus"
)

var conf = flag.String("conf", "", "ECS service family at task definition")

func main() {
	finished := make(chan bool)
	go loading(finished)

	flag.Parse()
	deployment, taskDefinition, err := task.ReadConfig(*conf)

	if err != nil {
		log.Fatal(err.Error())
	}

	service := deployment.Service
	cluster := deployment.Cluster
	count := deployment.Count

	oldRevision, _ := awsecs.GetOldRevision(service, cluster)
	log.Info("Now Revision is ... ", oldRevision)
	revision := ""

	if *conf != "" {
		newRevision, err := awsecs.RegisterTaskDefinition(taskDefinition)
		if err != nil {
			log.Fatal(err.Error())
		}
		revision = newRevision
	} else {
		revision = oldRevision
	}

	log.Info("Deploying Revision is ... ", revision)
	log.Info("Deploy Start ....")

	getRevisionError := awsecs.UpdateService(service, cluster, revision, count)
	if getRevisionError != nil {
		log.Fatal("UpdateService Error -> ", getRevisionError.Error())
	}

	_, deployError := awsecs.PollingDeployment(service, cluster)
	if deployError != nil {
		log.Fatal("Deploy Error -> ", deployError.Error())
		rollback := awsecs.UpdateService(service, cluster, oldRevision, count)
		if rollback != nil {
			log.Fatal("RollBack Revision Error -> ", getRevisionError.Error())
		} else {
			log.Info("RollBack Revision -> ", oldRevision)
		}
	} else {
		log.Info("Deploy SUCCESS -> ", service)
	}
	finished <- true

}

func loading(finished chan bool) {
loop:
	for {
		select {
		case <-finished:
			break loop
		default:
			array := []string{"|", "/", "-", "\\"}
			for _, v := range array {
				fmt.Printf("%s\033[1D", v)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}
}
