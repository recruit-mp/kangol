package awsecs

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var svc = awsECSConfig()
var deploymentMessage = ""
var pollingCount = 0

// Deployments has deployment information at ECS
type Deployments struct {
	active  int64
	primary int64
	desire  int64
	message string
}

func awsECSConfig() *ecs.ECS {
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

	svc := ecs.New(
		session.New(),
		&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		})
	return svc
}

// GetOldRevision can get revision you specified
func GetOldRevision(service, cluster string) (revision string, err error) {

	params := &ecs.DescribeServicesInput{
		Services: []*string{
			aws.String(service),
		},
		Cluster: aws.String(cluster),
	}

	resp, err := svc.DescribeServices(params)
	if err != nil {
		return "", err
	}

	return strings.Split(*resp.Services[0].TaskDefinition, "/")[1], err
}

// RegisterTaskDefinition can get register task-definition using your yml file
func RegisterTaskDefinition(taskDefinition *ecs.RegisterTaskDefinitionInput) (revision string, err error) {

	resp, err := svc.RegisterTaskDefinition(taskDefinition)
	if err != nil {
		log.Fatal("RegisterTaskDefinition Error -> ", err.Error())
	}
	return strings.Split(*resp.TaskDefinition.TaskDefinitionArn, "/")[1], err
}

// UpdateService can update service using revision you specified
func UpdateService(service, cluster, revision string, desiredCount int64) error {

	params := &ecs.UpdateServiceInput{
		Service:        aws.String(service),
		Cluster:        aws.String(cluster),
		DesiredCount:   aws.Int64(desiredCount),
		TaskDefinition: aws.String(revision),
	}

	_, err := svc.UpdateService(params)
	return err
}

// DescribeDeployedService can get running count
func DescribeDeployedService(service, cluster string) (Deployments, error) {
	param := &ecs.DescribeServicesInput{
		Services: []*string{
			aws.String(service),
		},
		Cluster: aws.String(cluster),
	}

	res, err := svc.DescribeServices(param)

	deployment := Deployments{}

	if err != nil {
		return deployment, err
	}

	deployment.desire = *res.Services[0].DesiredCount

	for _, v := range res.Services[0].Deployments {
		switch *v.Status {
		case "ACTIVE":
			deployment.active = *v.RunningCount
		case "PRIMARY":
			deployment.primary = *v.RunningCount
		}
	}

	deployment.message = *res.Services[0].Events[0].Message
	return deployment, err
}

// PollingDeployment can check deployment message at update-service
func PollingDeployment(service, cluster string, pollingTime int64) (string, error) {
	log.Info(fmt.Sprintf("Wait %d seconds", pollingTime))
	time.Sleep(time.Duration(pollingTime) * time.Second)
	deployment, err := DescribeDeployedService(service, cluster)
	if err != nil {
		return deployment.message, err
	}
	log.Info(deployment.message)

	// TODO: タスクが連続で変わり続ける場合はdeploy失敗
	// (ex) service nginx has started 1 tasks: task 474be549-f9e0-4aee-bf1b-6fbac8e3b445.

	log.Info(int64(pollingCount) * pollingTime)
	if int64(pollingCount)*pollingTime > 600 {
		return deployment.message, errors.New("PollingDeployment time out")
	}

	if deploymentMessage == "" {
		deploymentMessage = deployment.message
	} else if strings.Contains(deployment.message, "insufficient memory") || strings.Contains(deployment.message, "insufficient CPU") {
		log.Info("Waiting for Node AutoScaler...")
	} else if deploymentMessage != deployment.message {
		pollingCount = 0
		_, err := checkResource(deployment.message)
		if err != nil {
			return deployment.message, err
		}
	}

	if (deployment.primary == deployment.desire) && deployment.active == 0 {
		return deployment.message, nil
	}

	deploymentMessage = deployment.message
	pollingCount++
	return PollingDeployment(service, cluster, pollingTime)

}

// RunOneShotTask can run an ECS task
func RunOneShotTask(cluster string, taskDefinition string, command []*string, cpu int64, memory int64) (string, error) {
	containerName := strings.Split(taskDefinition, ":")[0]
	param := &ecs.RunTaskInput{
		Cluster:        aws.String(cluster),
		Count:          aws.Int64(1),
		TaskDefinition: aws.String(taskDefinition),
		LaunchType:     aws.String(ecs.LaunchTypeEc2),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name:              aws.String(containerName),
					Command:           command,
					Cpu:               aws.Int64(cpu),
					MemoryReservation: aws.Int64(memory),
				},
			},
		},
	}

	log.Info("RunTask with parameter -> ", *param)
	res, err := svc.RunTask(param)
	if err != nil {
		return "", err
	}
	log.Info("RunTask result -> ", *res)
	if len(res.Tasks) == 0 {
		return "", errors.New("failure RunTask result count is zero")
	}
	taskArn := res.Tasks[0].TaskArn
	tasksInput := &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks: []*string{
			taskArn,
		},
	}
	log.Info("Task Created -> ", *taskArn)
	// デフォルトは6s * 100回で合計600s待つようになっている
	// 並列で実行することを考慮して間隔を10sにした
	maxAttempts := request.WithWaiterMaxAttempts(200)
	waiterDelay := request.WithWaiterDelay(request.ConstantWaiterDelay(10 * time.Second))
	waitTaskStoppedError := svc.WaitUntilTasksStoppedWithContext(
		aws.BackgroundContext(),
		tasksInput,
		maxAttempts,
		waiterDelay,
	)
	if waitTaskStoppedError != nil {
		return *taskArn, waitTaskStoppedError
	}
	log.Info("Task RUNNING and STOPPED -> ", *taskArn)

	result, runTaskError := svc.DescribeTasks(tasksInput)
	if runTaskError != nil {
		return *taskArn, runTaskError
	}
	log.Info("Describe Task result -> ", *result)
	if *result.Tasks[0].Containers[0].ExitCode != 0 {
		return *taskArn, errors.New("task exited abnormally")
	}

	return *taskArn, err
}

func checkResource(message string) (string, error) {
	// TODO: コンテナ配置時におけるメッセージにてエラーを検出
	if strings.Contains(message, "resources could not be found") {
		return message, errors.New("resources could not be found")
	} else if strings.Contains(message, "was unable to place a task") {
		return message, errors.New(message)
	}
	return message, nil
}
