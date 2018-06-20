package main

import (
	"os"
	"bytes"
	"os/exec"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"net"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1alpha"
)

//const (
var socketName string   //= "sfcNIC"
var resourceName string //= "solarflare.com/sfc"

//)
// sfcNICManager manages Solarflare NIC devices
type sfcNICManager struct {
	devices     map[string]*pluginapi.Device
	deviceFiles []string
}

func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewSFCNICManager() (*sfcNICManager, error) {
	return &sfcNICManager{
		devices:     make(map[string]*pluginapi.Device),
		deviceFiles: []string{"/dev/onload", "/dev/onload_epoll", "/dev/sfc_char", "/dev/sfc_affinity"},
	}, nil
}

// v1.10 func (m *sfcNICManager) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
// v1.10	return &pluginapi.DevicePluginOptions{}, nil
// v1.10 }

// v1.10 func (m *sfcNICManager) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
// v1.10 	return &pluginapi.PreStartContainerResponse{}, nil
// v1.10 }


func ExecCommand(cmdName string, arg ...string) (bytes.Buffer, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(cmdName, arg...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("CMD--" + cmdName + ": " + fmt.Sprint(err) + ": " + stderr.String())
	}

	return out, err
}

func (sfc *sfcNICManager) discoverSolarflareResources() bool {
	found := false
	sfc.devices = make(map[string]*pluginapi.Device)
	glog.Info("discoverSolarflareResources")


	out, err := ExecCommand("/usr/bin/list_sfc_devices.sh")
	if err != nil {
		glog.Errorf("error while listing sfc devices : %v %s", err, out)
		return found
	}

	sfc_dev := strings.Split(out.String(), "\n")

	sfc_dev = sfc_dev[2:] // remove comment

	fmt.Printf("len=%d cap=%d %v\n", len(sfc_dev), cap(sfc_dev), sfc_dev)	

	for idx, element := range sfc_dev {
		if len(sfc_dev) - 1 == idx { continue }
		i := strings.Split(element, " ")
		dev := pluginapi.Device{ID: i[0], Health: pluginapi.Healthy}	
		sfc.devices[i[0]] = &dev
		found = true
	}

	fmt.Printf("Devices: %v \n", sfc.devices)

	return found
}

func (sfc *sfcNICManager) isOnloadInstallHealthy() bool {
	healthy := true
	// If the following command executes without ERROR,Failed we have 
	// a valid onload installation no need to check for libonload
/*
	out, _ := ExecCommand("onload", "/usr/bin/ping", "-c1", "localhost")
	fmt.Printf("Onload: %s", out.String());
	if strings.Contains(out.String(), "ERROR") {
		fmt.Errorf("onload error, looks like libonload is missing %s", out)
		return false
	}
        if strings.Contains(out.String(), "Failed") {
                fmt.Errorf("onload error, looks like devices are missing %s", out)
                return false
        } 
	if strings.Contains(out.String(), "1 received") {
		healthy = true
	} else {
		fmt.Errorf("onload error, did not receive a packet %s", out)
	}
	return healthy
*/
        return healthy
}


func (sfc *sfcNICManager) Init() int {
	glog.Info("Init\n")
	return 0
}

func Register(kubeletEndpoint string, pluginEndpoint, socketName string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("device-plugin: cannot connect to kubelet service: %v", err)
	}
	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}

// Implements DevicePlugin service functions
func (sfc *sfcNICManager) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	glog.Info("device-plugin: ListAndWatch start\n")
	for {
		sfc.discoverSolarflareResources()
		if !sfc.isOnloadInstallHealthy() {
			glog.Errorf("Error with onload installation. Marking devices unhealthy.")
			for _, device := range sfc.devices {
				device.Health = pluginapi.Unhealthy
			}
		}
		resp := new(pluginapi.ListAndWatchResponse)
		for _, dev := range sfc.devices {
			glog.Info("dev ", dev)
			resp.Devices = append(resp.Devices, dev)
		}
		glog.Info("resp.Devices ", resp.Devices)
		if err := stream.Send(resp); err != nil {
			glog.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (sfc *sfcNICManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
        glog.Info("Allocate")
        resp := new(pluginapi.AllocateResponse)
        //      containerName := strings.Join([]string{"k8s", "POD", rqt.PodName, rqt.Namespace}, "_")
        for _, id := range rqt.DevicesIDs {
                if _, ok := sfc.devices[id]; ok {
/*                        for _, d := range sfc.deviceFiles {
                                resp.Devices = append(resp.Devices, &pluginapi.DeviceSpec{
                                        HostPath:      d,
                                        ContainerPath: d,
                                        Permissions:   "mrw",
                                })
                        }
*/
                        glog.Info("Allocated interface ", id)
                        //glog.Info("Allocate interface ", id, " to ", containerName)
 //                       go MoveInterface(id)
                }
        }
        return resp, nil
}


func main()  {
	flag.Parse()
	fmt.Printf("Starting main \n")

	socketName   = "sfcNIC"
	resourceName = "solarflare.com/sfc"

	flag.Lookup("logtostderr").Value.Set("true")

	sfc, err := NewSFCNICManager()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	found := sfc.discoverSolarflareResources()
	if !found {
		glog.Errorf("No SolarFlare NICs are present\n")
		os.Exit(1)
	}

	if !sfc.isOnloadInstallHealthy() {
		glog.Errorf("Error with onload installation")
	}

	pluginEndpoint := fmt.Sprintf("%s-%d.sock", socketName, time.Now().Unix())
	//serverStarted := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	// Starts device plugin service.
	go func() {
		defer wg.Done()
		fmt.Printf("DevicePluginPath %s, pluginEndpoint %s\n", pluginapi.DevicePluginPath, pluginEndpoint)
		fmt.Printf("device-plugin start server at: %s\n", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		lis, err := net.Listen("unix", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		if err != nil {
			glog.Fatal(err)
			return
		}
		grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
		pluginapi.RegisterDevicePluginServer(grpcServer, sfc)
		grpcServer.Serve(lis)
	}()

	conn, err := dial(path.Join(pluginapi.DevicePluginPath, pluginEndpoint), 5 * time.Second)
	if err != nil {
		glog.Fatal("Error dialling grpcServer\n")
		return
	}
	conn.Close()

	err = Register(pluginapi.KubeletSocket, pluginEndpoint, resourceName)
	if err != nil {
		glog.Fatal(err)
	}
	fmt.Printf("device-plugin registered\n")

	wg.Wait()
}
