package awscloudwatchlogs

import (
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

var svc = cloudWatchLogsClient()

func cloudWatchLogsClient() *cloudwatchlogs.CloudWatchLogs {
	accessKeyID := strings.Trim(os.Getenv("AWS_ACCESS_KEY_ID"), " ")
	secretAccessKey := strings.Trim(os.Getenv("AWS_SECRET_ACCESS_KEY"), " ")
	region := strings.Trim(os.Getenv("AWS_DEFAULT_REGION"), " ")

	if accessKeyID == "" || secretAccessKey == "" || region == "" {
		log.Fatal(
			"AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY or AWS_DEFAULT_REGION is NULL \n",
			"[MUST] \n",
			"export AWS_ACCESS_KEY_ID=<YOUR AWS_ACCESS_KEY_ID> \n",
			"export AWS_SECRET_ACCESS_KEY=<YOUR AWS_SECRET_ACCESS_KEY> \n",
			"export AWS_DEFAULT_REGION=<ECS AWS_DEFAULT_REGION> \n",
		)
	}

	svc := cloudwatchlogs.New(
		session.New(),
		&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		})
	return svc
}

// GetLogEvents can get log events
func GetLogEvents(logGroupName string, logStreamName string) (events []string, err error) {

	params := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
	}

	res, err := svc.GetLogEvents(params)
	if err != nil {
		events = []string{}
		return events, err
	}

	events = []string{}
	for _, v := range res.Events {
		events = append(events, *v.Message)
	}
	return events, err
}
