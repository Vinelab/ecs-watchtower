package infrastructure

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// REGION defines the AWS AZ that we're working with
const REGION = "eu-central-1"

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

// Service details within a cluster
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

// Task details within a Service
type Task struct {
	*ecs.Task
}

// Infrastructure represents the infrastructure
type Infrastructure struct {
	ECSSvc *ecs.ECS
	EC2Svc *ec2.EC2
}

func (i *Infrastructure) checkError(err error) {
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

func (i *Infrastructure) listClusters() []*Cluster {
	// get clusters ARNs
	arnsInput := &ecs.ListClustersInput{}

	arnsResult, err := i.ECSSvc.ListClusters(arnsInput)
	i.checkError(err)

	var arns []*string
	for _, arn := range arnsResult.ClusterArns {
		arns = append(arns, arn)
	}

	describeInput := &ecs.DescribeClustersInput{
		Clusters: arns,
	}

	result, err := i.ECSSvc.DescribeClusters(describeInput)
	i.checkError(err)

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

func (i *Infrastructure) listServices(cluster *Cluster) []*ecs.Service {
	var services []*ecs.Service
	var nextToken *string

	for {
		input := &ecs.ListServicesInput{
			Cluster:   cluster.Arn,
			NextToken: nextToken,
		}

		result, err := i.ECSSvc.ListServices(input)
		i.checkError(err)

		arns := result.ServiceArns

		servicesInput := &ecs.DescribeServicesInput{
			Cluster:  cluster.Arn,
			Services: arns,
		}

		servicesResult, err := i.ECSSvc.DescribeServices(servicesInput)
		i.checkError(err)

		services = append(services, servicesResult.Services...)

		if result.NextToken == nil {
			return services
		}

		nextToken = result.NextToken
	}
}

func (i *Infrastructure) getTaskDefinitionForService(s *ecs.Service, c *Cluster) *ecs.TaskDefinition {
	input := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: s.TaskDefinition,
	}

	result, err := i.ECSSvc.DescribeTaskDefinition(input)
	i.checkError(err)
	return result.TaskDefinition
}

func (i *Infrastructure) listServiceTasks(service *ecs.Service, cluster *Cluster) []*Task {
	arnsInput := &ecs.ListTasksInput{
		Cluster:     cluster.Arn,
		ServiceName: service.ServiceArn,
	}

	arnsResult, err := i.ECSSvc.ListTasks(arnsInput)
	i.checkError(err)

	if len(arnsResult.TaskArns) > 0 {
		tasksInput := &ecs.DescribeTasksInput{
			Cluster: cluster.Arn,
			Tasks:   arnsResult.TaskArns,
		}

		result, err := i.ECSSvc.DescribeTasks(tasksInput)
		i.checkError(err)

		var tasks []*Task
		for _, ecsTask := range result.Tasks {
			tasks = append(tasks, &Task{ecsTask})
		}
		return tasks
	}

	return []*Task{}
}

// GetEc2Instance fetches EC2 metadata on an instance
func (i *Infrastructure) GetEc2Instance(id *string) *ec2.Instance {

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{id},
	}

	result, err := i.EC2Svc.DescribeInstances(input)
	i.checkError(err)

	if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
		return result.Reservations[0].Instances[0]
	}

	return nil
}

func (i *Infrastructure) makeService(ecsService *ecs.Service, cluster *Cluster, services chan *Service, wg *sync.WaitGroup) {
	defer wg.Done()
	// list service tasks
	tasks := i.listServiceTasks(ecsService, cluster)
	taskDefinition := i.getTaskDefinitionForService(ecsService, cluster)
	services <- &Service{
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
}

func (i *Infrastructure) getClusterInfo(cluster *Cluster, wg *sync.WaitGroup) {
	defer wg.Done()
	ecsServices := i.listServices(cluster)

	services := make(chan *Service, len(ecsServices))
	var servicesWG sync.WaitGroup
	servicesWG.Add(len(ecsServices))
	for _, ecsService := range ecsServices {
		go i.makeService(ecsService, cluster, services, &servicesWG)
	}
	servicesWG.Wait()
	cluster.SetServices(services, len(ecsServices))
}

// GetContainerInstance fetches information about the EC2 instance that hosts tasks
func (i *Infrastructure) GetContainerInstance(c *Cluster, ci *string) *ecs.ContainerInstance {
	input := &ecs.DescribeContainerInstancesInput{
		Cluster: c.Arn,
		ContainerInstances: []*string{
			ci,
		},
	}

	result, err := i.ECSSvc.DescribeContainerInstances(input)
	i.checkError(err)

	if len(result.ContainerInstances) > 0 {
		return result.ContainerInstances[0]
	}

	return nil
}

// SetServices is used to set the services on the corresponding cluster
func (c *Cluster) SetServices(s chan *Service, count int) {
	var services []*Service
	for i := 0; i < count; i++ {
		services = append(services, <-s)
	}
	c.Services = services
}

// New creates a new Infrastructure instance and returns it
func New() *Infrastructure {
	// create an ECS instance
	ecsSvc := ecs.New(session.New(&aws.Config{
		Region: aws.String(REGION),
	}))

	// create an EC2 instance
	ec2Svc := ec2.New(session.New(&aws.Config{
		Region: aws.String(REGION),
	}))

	return &Infrastructure{ECSSvc: ecsSvc, EC2Svc: ec2Svc}
}

// Clusters fetches the information about clusters, their services, tasks and definitions from AWS
func (i *Infrastructure) Clusters() []*Cluster {
	// list clusters
	clusters := i.listClusters()
	fmt.Println(len(clusters), " clusters found")

	// list cluster services and associate them accordingly
	var wg sync.WaitGroup
	wg.Add(len(clusters))
	for _, cluster := range clusters {
		go i.getClusterInfo(cluster, &wg)
	}
	wg.Wait()

	return clusters
}

// Recover a failing task
func (i Infrastructure) Recover(t *Task, reason string) error {
	input := &ecs.StopTaskInput{
		Cluster: t.ClusterArn,
		Task:    t.TaskArn,
		Reason:  aws.String(reason),
	}

	_, err := i.ECSSvc.StopTask(input)

	return err
}
