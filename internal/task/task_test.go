package task

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func TestAddImageTag(t *testing.T) {
	expect := "image:123"
	actual := addImageTag("image:tag", "123")
	if actual != expect {
		t.Errorf("expect %s but actual %s", expect, actual)
	}
}

func TestReadConfigDoesNotExistFile(t *testing.T) {
	expect := ClusterService{}
	cs, td, err := ReadConfig("not_found.yml", map[string]string{"image": "revision"})
	if cs != expect {
		t.Errorf("expect %+v but actual %+v", expect, cs)
	}
	if td != nil {
		t.Error("RegisterTaskDefinitionInput is not nil")
	}
	if err == nil {
		t.Error("err does not occur")
	}
}

func TestReadConfig(t *testing.T) {
	expectClusterService := ClusterService{
		Cluster: "cluster-name",
		Service: "service-name",
		Count:   5,
	}
	envs := []*ecs.KeyValuePair{
		&ecs.KeyValuePair{
			Name:  aws.String("ENV1"),
			Value: aws.String("VALUE1"),
		},
		&ecs.KeyValuePair{
			Name:  aws.String("ENV2"),
			Value: aws.String("VALUE2"),
		},
	}
	expectTaskDefinition := &ecs.RegisterTaskDefinitionInput{
		Family:  aws.String("task-name"),
		Volumes: []*ecs.Volume{},
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Cpu:       aws.Int64(256),
				Essential: aws.Bool(true),
				Image:     aws.String("registry.example.com/image:revision"),
				Memory:    aws.Int64(512),
				Name:      aws.String("service-name"),
				PortMappings: []*ecs.PortMapping{
					&ecs.PortMapping{
						ContainerPort: aws.Int64(9000),
						HostPort:      aws.Int64(0),
						Protocol:      aws.String("tcp"),
					},
				},
				Command:     []*string{},
				EntryPoint:  []*string{},
				Environment: envs,
				Links:       []*string{},
				MountPoints: []*ecs.MountPoint{},
				VolumesFrom: []*ecs.VolumeFrom{
					&ecs.VolumeFrom{
						SourceContainer: aws.String("test"),
						ReadOnly:        aws.Bool(false),
					},
				},
				DockerLabels: map[string]*string{
					"key1":      aws.String("value1"),
					"key2":      aws.String("value2"),
					"my.dotkey": aws.String("my.dotvalue"),
				},
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String(ecs.LogDriverAwslogs),
					Options: map[string]*string{
						"awslogs-region": aws.String("ap-northeast-1"),
						"awslogs-group":  aws.String("log-group"),
					},
				},
			},
		},
	}
	cs, td, err := ReadConfig("task_test.yml", map[string]string{"service-name": "revision"})
	if cs != expectClusterService {
		t.Errorf("expect %+v but actual %+v", expectClusterService, cs)
	}
	if !reflect.DeepEqual(td, expectTaskDefinition) {
		t.Errorf("expect %+v but actual %+v", expectTaskDefinition, td)
	}
	if err != nil {
		t.Errorf("err %+v", err)
	}
}

func TestReadConfigWithNetworkMode(t *testing.T) {
	expectClusterService := ClusterService{
		Cluster: "cluster-name",
		Service: "service-name",
		Count:   5,
	}
	envs := []*ecs.KeyValuePair{
		&ecs.KeyValuePair{
			Name:  aws.String("ENV1"),
			Value: aws.String("VALUE1"),
		},
		&ecs.KeyValuePair{
			Name:  aws.String("ENV2"),
			Value: aws.String("VALUE2"),
		},
	}
	expectTaskDefinition := &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("task-name"),
		Volumes:     []*ecs.Volume{},
		NetworkMode: aws.String("awsvpc"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Cpu:       aws.Int64(256),
				Essential: aws.Bool(true),
				Image:     aws.String("registry.example.com/image:revision"),
				Memory:    aws.Int64(512),
				Name:      aws.String("service-name"),
				PortMappings: []*ecs.PortMapping{
					&ecs.PortMapping{
						ContainerPort: aws.Int64(9000),
						HostPort:      aws.Int64(0),
						Protocol:      aws.String("tcp"),
					},
				},
				HealthCheck: &ecs.HealthCheck{
					Command:     aws.StringSlice([]string{"bash", "-c", "test.sh"}),
					Interval:    aws.Int64(5),
					Retries:     aws.Int64(5),
					StartPeriod: aws.Int64(10),
					Timeout:     aws.Int64(10),
				},
				Command:     []*string{},
				EntryPoint:  []*string{},
				Environment: envs,
				Links:       []*string{},
				MountPoints: []*ecs.MountPoint{},
				VolumesFrom: []*ecs.VolumeFrom{
					&ecs.VolumeFrom{
						SourceContainer: aws.String("test"),
						ReadOnly:        aws.Bool(false),
					},
				},
				DockerLabels: map[string]*string{
					"key1":      aws.String("value1"),
					"key2":      aws.String("value2"),
					"my.dotkey": aws.String("my.dotvalue"),
				},
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String(ecs.LogDriverAwslogs),
					Options: map[string]*string{
						"awslogs-region": aws.String("ap-northeast-1"),
						"awslogs-group":  aws.String("log-group"),
					},
				},
			},
		},
	}
	cs, td, err := ReadConfig("task_network_mode_test.yml", map[string]string{"service-name": "revision"})
	if cs != expectClusterService {
		t.Errorf("expect %+v but actual %+v", expectClusterService, cs)
	}
	if !reflect.DeepEqual(td, expectTaskDefinition) {
		t.Errorf("expect %+v but actual %+v", expectTaskDefinition, td)
	}
	if err != nil {
		t.Errorf("err %+v", err)
	}
}
