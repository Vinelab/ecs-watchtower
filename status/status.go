package status

import (
	"fmt"
	"strings"

	"github.com/sourcegraph/checkup"
	"github.com/vinelab/watchtower/infrastructure"
)

// PrintNotifier to print the results of health checks
type PrintNotifier struct {
}

// IPAccess determines whether to access the private or public IP address of a service
type IPAccess string

const (
	// IPAccessPUBLIC is for accessing the public IP address
	IPAccessPUBLIC = "public"
	// IPAccessPRIVATE is for accessing the private IP adress
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

func (pn PrintNotifier) Notify(result []checkup.Result) error {
	fmt.Println(result)

	return nil
}

func checkAndStore(checkers []checkup.Checker) {
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

// Check performs the health check on the corresponding services in the given clusters
func Check(i *infrastructure.Infrastructure, clusters []*infrastructure.Cluster) {
	var checks []*HealthCheckConfig
	for _, cluster := range clusters {
		for _, service := range cluster.Services {
			// get info about the service and its tasks
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
					// set the data for the health check
					hcconf.SetServiceName(*sn)
					hcconf.SetCheckProtocol(Protocol(hcinfo[0]))
					hcconf.SetCheckIPAccess(IPAccess(hcinfo[1]))

					// set service info
					if len(hcinfo) < 3 {
						hcconf.SetCheckEndpoint("/")
					} else {
						hcconf.SetCheckEndpoint(hcinfo[2])
					}
					// describe container instance
					ci := i.GetContainerInstance(cluster, task.ContainerInstanceArn)
					if ci != nil {
						// get ec2 instance id
						ec2i := i.GetEc2Instance(ci.Ec2InstanceId)
						hcconf.SetCheckPublicIPAddress(*ec2i.PublicIpAddress)
						hcconf.SetCheckPrivateIPAddress(*ec2i.PrivateIpAddress)
					}

					// determine the open port(s) for a task definition
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

	checkAndStore(checkers)
}
