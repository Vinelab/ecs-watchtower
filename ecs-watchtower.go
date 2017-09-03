package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/sourcegraph/checkup"
)

// print notifier to print the results of health checks
type PrintNotifier struct {
}

// IPAccess determines whether to access the private or public IP address of a service
type IPAccess string

const (
	// IPAccessPUBLIC is for accessing the public IP address
	IPAccessPUBLIC = "public"
	// IPAccessPrivate is for accessing the private IP adress
	IPAccessPRIVATE = "private"
)

// Protocol defines the supported protocol types
type Protocol string

const (
	ProtocolHTTP = "http"
	ProtocolTCP  = "tcp"
)

// HealthCheckConfig is intuitively named to specify the configuration of the health check to perform
type HealthCheckConfig struct {
	ServiceName           string
	CheckProtocol         Protocol
	CheckIPAccess         IPAccess
	CheckPublicIPAddress  string
	CheckPrivateIPAddress string
	CheckPort             int64
	CheckEndpoint         string
}

// SetServiceName is used to set the service name for the healthcheck
func (hcc *HealthCheckConfig) SetServiceName(name string) bool {
	hcc.ServiceName = strings.Title(strings.Replace(name, "-", " ", -1))

	return true
}

// SetCheckProtocol is used to validate and set the service name for the healthcheck
func (hcc *HealthCheckConfig) SetCheckProtocol(p Protocol) bool {
	switch p {
	case ProtocolHTTP, ProtocolTCP:
		hcc.CheckProtocol = p

		return true
	}

	return false
}

// SetCheckIPAccess is to validate and set the ip access of the check (public, private)
func (hcc *HealthCheckConfig) SetCheckIPAccess(ipa IPAccess) bool {
	switch ipa {
	case IPAccessPUBLIC, IPAccessPRIVATE:
		hcc.CheckIPAccess = ipa

		return true
	}

	return false
}

// SetCheckPublicIPAddress is for setting the health check's public ip address
func (hcc *HealthCheckConfig) SetCheckPublicIPAddress(ip string) {
	hcc.CheckPublicIPAddress = ip
}

// SetCheckPrivateIPAddress is for setting the health check's private ip address
func (hcc *HealthCheckConfig) SetCheckPrivateIPAddress(ip string) {
	hcc.CheckPrivateIPAddress = ip
}

// SetCheckPort is to set the health check's port
func (hcc *HealthCheckConfig) SetCheckPort(port int64) {
	hcc.CheckPort = port
}

// SetCheckEndpoint is for setting the health check's endpoint
func (hcc *HealthCheckConfig) SetCheckEndpoint(e string) {
	hcc.CheckEndpoint = e
}

// Cluster details
// https://docs.aws.amazon.com/goto/WebAPI/ecs-2014-11-13/Cluster
type Cluster struct {

	// The Amazon Resource Name (ARN) that identifies the service. The ARN contains
	// the arn:aws:ecs namespace, followed by the region of the service, the AWS
	// account ID of the service owner, the service namespace, and then the service
	// name. For example, arn:aws:ecs:region:012345678910:service/my-service.
	Arn *string `locationName:"serviceArn" type:"string"`

	// The name of your service. Up to 255 letters (uppercase and lowercase), numbers,
	// hyphens, and underscores are allowed. Service names must be unique within
	// a cluster, but you can have similarly named services in multiple clusters
	// within a region or across multiple regions.
	Name *string `locationName:"serviceName" type:"string"`

	// The status of the service. The valid values are ACTIVE, DRAINING, or INACTIVE.
	Status *string `locationName:"status" type:"string"`

	// The number of tasks in the cluster that are in the RUNNING state.
	RunningTasksCount *int64

	// The number of tasks in the cluster that are in the PENDING state.
	PendingTasksCount *int64

	ActiveServicesCount               *int64
	RegisteredContainerInstancesCount *int64
	Services                          []*Service
}

// Details on a service within a cluster
// Please also see https://docs.aws.amazon.com/goto/WebAPI/ecs-2014-11-13/Service
type Service struct {

	// The Amazon Resource Name (ARN) that identifies the service. The ARN contains
	// the arn:aws:ecs namespace, followed by the region of the service, the AWS
	// account ID of the service owner, the service namespace, and then the service
	// name. For example, arn:aws:ecs:region:012345678910:service/my-service.
	Arn *string `locationName:"serviceArn" type:"string"`

	// The Amazon Resource Name (ARN) of the cluster that hosts the service.
	ClusterArn *string `locationName:"clusterArn" type:"string"`

	// The Unix timestamp for when the service was created.
	CreatedAt *time.Time `locationName:"createdAt" type:"timestamp" timestampFormat:"unix"`

	// The desired number of instantiations of the task definition to keep running
	// on the service. This value is specified when the service is created with
	// CreateService, and it can be modified with UpdateService.
	DesiredCount *int64 `locationName:"desiredCount" type:"integer"`

	// The number of tasks in the cluster that are in the PENDING state.
	PendingCount *int64 `locationName:"pendingCount" type:"integer"`

	// The Amazon Resource Name (ARN) of the IAM role associated with the service
	// that allows the Amazon ECS container agent to register container instances
	// with an Elastic Load Balancing load balancer.
	RoleArn *string `locationName:"roleArn" type:"string"`

	// The number of tasks in the cluster that are in the RUNNING state.
	RunningCount *int64 `locationName:"runningCount" type:"integer"`

	// The name of your service. Up to 255 letters (uppercase and lowercase), numbers,
	// hyphens, and underscores are allowed. Service names must be unique within
	// a cluster, but you can have similarly named services in multiple clusters
	// within a region or across multiple regions.
	Name *string `locationName:"serviceName" type:"string"`

	// The status of the service. The valid values are ACTIVE, DRAINING, or INACTIVE.
	Status *string `locationName:"status" type:"string"`

	// The task definition to use for tasks in the service. This value is specified
	// when the service is created with CreateService, and it can be modified with
	// UpdateService.
	TaskDefinitionArn *string `locationName:"taskDefinition" type:"string"`

	TaskDefinition *ecs.TaskDefinition

	Tasks []*Task
}

type Task struct {
	*ecs.Task
}

func (pn PrintNotifier) Notify(result []checkup.Result) error {
	fmt.Println(result)

	return nil
}

func Watch(checkers []checkup.Checker) {
	n := PrintNotifier{}

	c := checkup.Checkup{
		Checkers: checkers,
		Storage: checkup.FS{
			Dir: "/var/cache/ecs-watchtower/checks",
		},
		Notifier: n,
	}

	c.CheckAndStoreEvery(1 * time.Minute)
	select {}
}

func Check(checkers []checkup.Checker) {
	n := PrintNotifier{}

	c := checkup.Checkup{
		Checkers: checkers,
		Storage: checkup.FS{
			Dir: "/var/cache/ecs-watchtower/checks",
		},
		Notifier: n,
	}

	c.CheckAndStore()
}

func checkError(err error) {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			case ecs.ErrCodeServiceNotFoundException:
				fmt.Println(ecs.ErrCodeServiceNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
}

func (c *Cluster) SetServices(s []*Service) {
	c.Services = s
}

func listClusters(svc *ecs.ECS) []*Cluster {
	// get clusters ARNs
	arnsInput := &ecs.ListClustersInput{}

	arnsResult, err := svc.ListClusters(arnsInput)
	checkError(err)

	var arns []*string
	for _, arn := range arnsResult.ClusterArns {
		arns = append(arns, arn)
	}

	describeInput := &ecs.DescribeClustersInput{
		Clusters: arns,
	}

	result, err := svc.DescribeClusters(describeInput)
	checkError(err)

	var clusters []*Cluster
	for _, cluster := range result.Clusters {
		clusters = append(clusters, &Cluster{
			Arn:                               cluster.ClusterArn,
			Name:                              cluster.ClusterName,
			Status:                            cluster.Status,
			RunningTasksCount:                 cluster.RunningTasksCount,
			PendingTasksCount:                 cluster.PendingTasksCount,
			ActiveServicesCount:               cluster.ActiveServicesCount,
			RegisteredContainerInstancesCount: cluster.RegisteredContainerInstancesCount,
		})
	}

	return clusters
}

func listServices(svc *ecs.ECS, cluster *Cluster) []*ecs.Service {
	input := &ecs.ListServicesInput{
		Cluster: cluster.Arn,
	}

	result, err := svc.ListServices(input)
	checkError(err)

	arns := result.ServiceArns
	// fmt.Println("Services arns:")

	servicesInput := &ecs.DescribeServicesInput{
		Cluster:  cluster.Arn,
		Services: arns,
	}

	servicesResult, err := svc.DescribeServices(servicesInput)
	checkError(err)

	return servicesResult.Services
}

func getTaskDefinitionForService(svc *ecs.ECS, s *ecs.Service, c *Cluster) *ecs.TaskDefinition {
	input := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: s.TaskDefinition,
	}

	result, err := svc.DescribeTaskDefinition(input)
	checkError(err)

	return result.TaskDefinition
}

func listServiceTasks(svc *ecs.ECS, service *ecs.Service, cluster *Cluster) []*Task {
	// fmt.Println("listing tasks for ", *cluster.Arn, *service.ServiceArn)
	arnsInput := &ecs.ListTasksInput{
		Cluster:     cluster.Arn,
		ServiceName: service.ServiceArn,
	}

	arnsResult, err := svc.ListTasks(arnsInput)
	checkError(err)

	tasksInput := &ecs.DescribeTasksInput{
		Cluster: cluster.Arn,
		Tasks:   arnsResult.TaskArns,
	}

	result, err := svc.DescribeTasks(tasksInput)
	checkError(err)

	var tasks []*Task
	for _, ecsTask := range result.Tasks {
		task := &Task{ecsTask}
		tasks = append(tasks, task)
	}
	return tasks
}

func getContainerInstance(svc *ecs.ECS, c *Cluster, i *string) *ecs.ContainerInstance {
	input := &ecs.DescribeContainerInstancesInput{
		Cluster: c.Arn,
		ContainerInstances: []*string{
			i,
		},
	}

	result, err := svc.DescribeContainerInstances(input)
	checkError(err)

	if len(result.ContainerInstances) > 0 {
		return result.ContainerInstances[0]
	}

	return nil
}

func getEc2Instance(svc *ec2.EC2, id *string) *ec2.Instance {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{id},
	}

	result, err := svc.DescribeInstances(input)
	checkError(err)

	if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
		return result.Reservations[0].Instances[0]
	}

	return nil
}

func FetchAndCheck() {
	const REGION = "eu-central-1"

	// list running tasks
	svc := ecs.New(session.New(&aws.Config{
		Region: aws.String(REGION),
	}))

	// list clusters
	clusters := listClusters(svc)
	fmt.Println(len(clusters), " clusters found")

	// list cluster services and associate them accordingly
	for _, cluster := range clusters {
		ecsServices := listServices(svc, cluster)

		var services []*Service
		for _, ecsService := range ecsServices {
			// list service tasks
			tasks := listServiceTasks(svc, ecsService, cluster)
			taskDefinition := getTaskDefinitionForService(svc, ecsService, cluster)
			service := &Service{
				Arn:               ecsService.ServiceArn,
				Name:              ecsService.ServiceName,
				Status:            ecsService.Status,
				ClusterArn:        ecsService.ClusterArn,
				CreatedAt:         ecsService.CreatedAt,
				DesiredCount:      ecsService.DesiredCount,
				PendingCount:      ecsService.PendingCount,
				RoleArn:           ecsService.RoleArn,
				RunningCount:      ecsService.RunningCount,
				TaskDefinitionArn: ecsService.TaskDefinition,
				TaskDefinition:    taskDefinition,
				Tasks:             tasks,
			}

			services = append(services, service)
		}
		cluster.SetServices(services)
	}

	var checks []*HealthCheckConfig
	for _, cluster := range clusters {
		// fmt.Println(*cluster.Name + ": ")
		for _, service := range cluster.Services {
			// get info about the service and its tasks
			// fmt.Println(*service.Name)
			var sn *string
			var hcinfo []string
			// determines whether this services need to be checked or not.
			// this is determined by finding a SERVICE_HEALTH_CHECK env variable
			// in the service's container environment through the task definition.
			// it defaults to false cause if the env variable wasn't found we won't
			// include it in the check, while when it's found, it'll be set to true.
			shouldCheck := false
			for _, cdef := range service.TaskDefinition.ContainerDefinitions {
				// get service name from TDF env vars
				for _, env := range cdef.Environment {
					// set health check info
					if *env.Name == "SERVICE_HEALTH_CHECK" {
						shouldCheck = true
						hcinfo = strings.Split(*env.Value, ":")
						// config should at least specify the protocol and the ip access (public, private)
						if len(hcinfo) < 2 {
							shouldCheck = false
							fmt.Println("Invalid health check configuration", *env.Value)
						}
					}

					// set service name
					if *env.Name == "SERVICE_NAME" {
						sn = env.Value
					}
				}
			}

			if shouldCheck {
				// get tasks IPs
				for _, task := range service.Tasks {
					// we create a health check per task
					hcconf := HealthCheckConfig{}

					hcconf.SetServiceName(*sn)
					hcconf.SetCheckProtocol(Protocol(hcinfo[0]))
					hcconf.SetCheckIPAccess(IPAccess(hcinfo[1]))

					// set service info
					if len(hcinfo) < 3 {
						hcconf.SetCheckEndpoint("/")
					} else {
						hcconf.SetCheckEndpoint(hcinfo[2])
					}
					// fmt.Println("Task has ", len(task.Containers), " containers")
					// describe container instance
					ci := getContainerInstance(svc, cluster, task.ContainerInstanceArn)
					if ci != nil {
						// get ec2 instance id
						ec2c := ec2.New(session.New(&aws.Config{
							Region: aws.String(REGION),
						}))
						ec2i := getEc2Instance(ec2c, ci.Ec2InstanceId)
						hcconf.SetCheckPublicIPAddress(*ec2i.PublicIpAddress)
						hcconf.SetCheckPrivateIPAddress(*ec2i.PrivateIpAddress)
					}
					for _, c := range task.Containers {
						if len(c.NetworkBindings) > 0 {
							for _, nb := range c.NetworkBindings {
								var port int64
								if *nb.HostPort > 0 {
									port = *nb.HostPort
								} else {
									port = 80
								}
								hcconf.SetCheckPort(port)
							}
						}
					}

					checks = append(checks, &hcconf)
				}
			}
		}
	}

	fmt.Println("we have", len(checks), "checks to perform")

	var checkers []checkup.Checker
	for _, chk := range checks {
		var ip string
		if chk.CheckIPAccess == IPAccessPUBLIC {
			ip = chk.CheckPublicIPAddress
		} else {
			ip = chk.CheckPrivateIPAddress
		}

		// set the checker according to the configured protocol
		switch chk.CheckProtocol {
		case ProtocolHTTP:
			checkers = append(checkers, &checkup.HTTPChecker{Name: chk.ServiceName, URL: fmt.Sprintf("%s://%s:%d", "http", ip, chk.CheckPort), Attempts: 5})
		case ProtocolTCP:
			checkers = append(checkers, &checkup.TCPChecker{Name: chk.ServiceName, URL: fmt.Sprintf("%s:%d", ip, chk.CheckPort), Attempts: 5})
		}
	}

	Check(checkers)
}

func main() {

	// start working
	FetchAndCheck()

	// wait a minute and then work again
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			FetchAndCheck()
		}
	}()

	select {}
	// Watch(checkers)
}
