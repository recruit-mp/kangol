package operation

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/recruit-mp/kangol/internal/awsecs"
	"github.com/recruit-mp/kangol/internal/task"
)

func Deploy(conf, tag string, debug bool, skipPolling bool, pollingTime int64) {

	deployment, taskDefinition, err := task.ReadConfig(conf, appendTags(tag))

	if err != nil {
		log.Fatal(err.Error())
	}

	service := deployment.Service
	cluster := deployment.Cluster
	count := deployment.Count

	oldRevision, err := awsecs.GetOldRevision(service, cluster)

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Info("Now Revision is ... ", oldRevision)
	revision := ""

	if debug {
		log.Info("Stop All Tasks at debug mode ....")
		stopTaskError := awsecs.UpdateService(service, cluster, oldRevision, 0)
		if stopTaskError != nil {
			log.Fatal("Stop All Tasks Error -> ", stopTaskError.Error())
		}
		_, err := awsecs.PollingDeployment(service, cluster, pollingTime)
		if err != nil {
			log.Fatal("Stop All Tasks Error -> ", err.Error())
		}
	}

	if conf != "" {
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

	if skipPolling {
		log.Info("Skip Polling Deploy SUCCESS -> ", service)
	} else {
		_, deployError := awsecs.PollingDeployment(service, cluster, pollingTime)
		if deployError != nil {
			rollback := awsecs.UpdateService(service, cluster, oldRevision, count)
			if rollback != nil {
				log.Fatal("Deploy Error & RollBack Revision Error -> ", getRevisionError.Error())
			} else {
				log.Info("RollBack Revision -> ", oldRevision)
				log.Fatal("Deploy Error -> ", deployError.Error())
			}
		} else {
			log.Info("Deploy SUCCESS -> ", service)
		}
	}

}

func appendTags(tag string) map[string]string {
	tags := make(map[string]string)
	if tag == "" {
		return tags
	}

	nameTags := strings.Split(tag, ",")
	for _, v := range nameTags {
		nameAndTag := strings.Split(v, ":")
		tags[nameAndTag[0]] = nameAndTag[1]
	}
	return tags
}

func Loading(finished chan bool) {
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
