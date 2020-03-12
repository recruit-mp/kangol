package task

import (
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"

	"gopkg.in/yaml.v2"
)

type ClusterService struct {
	Cluster string
	Service string
	Count   int64
}

// Deployment has deployment config setting
type Deployment struct {
	Cluster     string                         `yaml:"cluster"`
	Service     string                         `yaml:"service"`
	Count       int64                          `yaml:"desiredCount"`
	Name        string                         `yaml:"name"`
	NetworkMode string                         `yaml:"networkMode"`
	Task        map[string]ContainerDefinition `yaml:"task"`
}

// ContainerDefinition is struct for ECS TaskDefinition
type ContainerDefinition struct {
	CPU              int64              `yaml:"cpu"`
	Essential        bool               `yaml:"essential"`
	Image            string             `yaml:"image"`
	Memory           int64              `yaml:"memory"`
	PortMappings     []PortMapping      `yaml:"portMappings"`
	HealthCheck      HealthCheck        `yaml:"healthCheck"`
	Command          []string           `yaml:"command"`
	EntryPoint       []string           `yaml:"entrypoint"`
	Environment      []Environment      `yaml:"environment"`
	Link             []string           `yaml:"links"`
	MountPoints      []MountPoint       `yaml:"mountPoint"`
	VolumesFrom      []VolumesFrom      `yaml:"volumesFrom"`
	Volumes          []Volume           `yaml:"volumes"`
	LogConfiguration LogConfiguration   `yaml:"logConfiguration"`
	DockerLabels     map[string]*string `yaml:"dockerLabels"`
}

// PortMapping is struct for ECS TaskDefinition's PortMapping
type PortMapping struct {
	ContainerPort int64  `yaml:"containerPort"`
	HostPort      int64  `yaml:"hostPort"`
	Protocol      string `yaml:"protocol"`
}

// HealthCheck is struct for ECS TaskDefinition's PortMapping
type HealthCheck struct {
	Command     []string `yaml:"command"`
	Interval    int64    `yaml:"interval"`
	Retries     int64    `yaml:"retries"`
	StartPeriod int64    `yaml:"startPeriod"`
	Timeout     int64    `yaml:"timeout"`
}

// Environment is struct for TaskDefinition's Environment
type Environment struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// MountPoint is struct for TaskDefinition's MoutPoint
type MountPoint struct {
	ContainerPath string `yaml:"containerPath"`
	ReadOnly      bool   `yaml:"readOnly"`
	SourceVolume  string `yaml:"souceVolume"`
}

// VolumesFrom is struct for TaskDefinition's VolumesFrom
type VolumesFrom struct {
	ReadOnly        bool   `yaml:"readOnly"`
	SourceContainer string `yaml:"sourceContainer"`
}

// Volume is struct for TaskDefinition's Volume
type Volume struct {
	Host TaskVolumeHost `yaml:"host"`
	Name string         `yaml:"name"`
}

// TaskVolumeHost is struct for TaskDefinition's Volumes's volumeHost
type TaskVolumeHost struct {
	SourcePath string `yaml:"sourcePath"`
}

// LogConfiguration is struct for TaskDefinition's LogConfiguration
type LogConfiguration struct {
	LogDriver string             `yaml:"logDriver"`
	Options   map[string]*string `yaml:"options"`
}

// ReadConfig can read config yml
func ReadConfig(conf string, tags map[string]string) (ClusterService, *ecs.RegisterTaskDefinitionInput, error) {
	data, readErr := ioutil.ReadFile(conf)

	if readErr != nil {
		return ClusterService{}, nil, readErr
	}

	containers := Deployment{}
	err := yaml.Unmarshal(data, &containers)

	clusterService := ClusterService{containers.Cluster, containers.Service, containers.Count}

	definitions := []*ecs.ContainerDefinition{}
	volumes := []*ecs.Volume{}

	for name, con := range containers.Task {
		if tags[name] != "" {
			con.Image = addImageTag(con.Image, tags[name])
		}

		def := &ecs.ContainerDefinition{
			Cpu:          aws.Int64(con.CPU),
			Essential:    aws.Bool(con.Essential),
			Image:        aws.String(con.Image),
			Memory:       aws.Int64(con.Memory),
			Name:         aws.String(name),
			PortMappings: getPortMapping(con),
			Command:      getCommands(con),
			EntryPoint:   getEntryPoints(con),
			Environment:  getEnvironments(con),
			Links:        getLinks(con),
			MountPoints:  getMountPoints(con),
			VolumesFrom:  getVolumesFrom(con),
			DockerLabels: con.DockerLabels,
		}

		if len(con.HealthCheck.Command) != 0 {
			def.HealthCheck = getHealthCheck(con)
		}

		if con.LogConfiguration.LogDriver != "" {
			def.LogConfiguration = getLogConfiguration(con)
		}
		definitions = append(definitions, def)

		volumes = getVolumes(con)
	}

	taskDefinitions := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: definitions,
		Family:               aws.String(containers.Name),
		Volumes:              volumes,
	}

	if containers.NetworkMode != "" {
		taskDefinitions.NetworkMode = aws.String(containers.NetworkMode)
	}

	return clusterService, taskDefinitions, err
}

func addImageTag(image, tag string) string {
	imagesArray := strings.Split(image, ":")
	imagesArray[len(imagesArray)-1] = tag
	return strings.Join(imagesArray, ":")
}

func getPortMapping(con ContainerDefinition) []*ecs.PortMapping {
	ports := []*ecs.PortMapping{}
	for _, v := range con.PortMappings {
		port := &ecs.PortMapping{
			ContainerPort: aws.Int64(v.ContainerPort),
			HostPort:      aws.Int64(v.HostPort),
			Protocol:      aws.String(v.Protocol),
		}
		ports = append(ports, port)
	}
	return ports
}

func getHealthCheck(con ContainerDefinition) *ecs.HealthCheck {
	healthCheck := &ecs.HealthCheck{
		Command:     aws.StringSlice(con.HealthCheck.Command),
		Interval:    aws.Int64(con.HealthCheck.Interval),
		Retries:     aws.Int64(con.HealthCheck.Retries),
		StartPeriod: aws.Int64(con.HealthCheck.StartPeriod),
		Timeout:     aws.Int64(con.HealthCheck.Timeout),
	}
	return healthCheck
}

func getCommands(con ContainerDefinition) []*string {
	commands := []*string{}
	for _, v := range con.Command {
		commands = append(commands, aws.String(v))
	}
	return commands
}

func getEntryPoints(con ContainerDefinition) []*string {
	entrypoints := []*string{}
	for _, v := range con.EntryPoint {
		entrypoints = append(entrypoints, aws.String(v))
	}
	return entrypoints
}

func getEnvironments(con ContainerDefinition) []*ecs.KeyValuePair {
	environments := []*ecs.KeyValuePair{}
	for _, v := range con.Environment {
		env := &ecs.KeyValuePair{
			Name:  aws.String(v.Name),
			Value: aws.String(v.Value),
		}
		environments = append(environments, env)
	}
	return environments
}

func getLinks(con ContainerDefinition) []*string {
	links := []*string{}
	for _, v := range con.Link {
		links = append(links, aws.String(v))
	}
	return links
}

func getMountPoints(con ContainerDefinition) []*ecs.MountPoint {
	mountPoints := []*ecs.MountPoint{}
	for _, v := range con.MountPoints {
		mountPoint := &ecs.MountPoint{
			ContainerPath: aws.String(v.ContainerPath),
			ReadOnly:      aws.Bool(v.ReadOnly),
			SourceVolume:  aws.String(v.SourceVolume),
		}
		mountPoints = append(mountPoints, mountPoint)
	}
	return mountPoints
}

func getVolumesFrom(con ContainerDefinition) []*ecs.VolumeFrom {
	volumesFroms := []*ecs.VolumeFrom{}
	for _, v := range con.VolumesFrom {
		volumesFrom := &ecs.VolumeFrom{
			ReadOnly:        aws.Bool(v.ReadOnly),
			SourceContainer: aws.String(v.SourceContainer),
		}
		volumesFroms = append(volumesFroms, volumesFrom)
	}
	return volumesFroms
}

func getVolumes(con ContainerDefinition) []*ecs.Volume {
	volumes := []*ecs.Volume{}
	for _, v := range con.Volumes {
		vol := &ecs.Volume{
			Host: &ecs.HostVolumeProperties{
				SourcePath: aws.String(v.Host.SourcePath),
			},
			Name: aws.String(v.Name),
		}
		volumes = append(volumes, vol)
	}
	return volumes
}

func getLogConfiguration(con ContainerDefinition) *ecs.LogConfiguration {
	logConf := con.LogConfiguration
	conf := &ecs.LogConfiguration{
		LogDriver: aws.String(con.LogConfiguration.LogDriver),
		Options:   logConf.Options,
	}
	return conf
}
