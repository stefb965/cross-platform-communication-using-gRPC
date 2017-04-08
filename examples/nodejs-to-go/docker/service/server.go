package service

import (
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"encoding/json"
	"time"
	"io"
	"net"
	"log"
	"google.golang.org/grpc"
	"golang.org/x/net/context"
)

type containerService struct {}


func StartServer() error {

	config := getConfig()

	lis, err := net.Listen("tcp", config.serverHostPost)
	if err != nil {
		return err
	}
	log.Printf("Listening on [%s]....\n", config.serverHostPost)

	s := grpc.NewServer()
	cs := &containerService{}

	RegisterDockerServiceServer(s, cs)

	return s.Serve(lis)

}

func (cs *containerService) GetAllContainers(context.Context, *GetAllContainersRequest) (*ContainerCatalog, error) {

	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	clo := types.ContainerListOptions{
		All:true,
	}
	allContainers, err := cli.ContainerList(context.Background(), clo)
	if err != nil {
		return nil, err
	}

	containers := make([]*Container, 0)

	for _, c := range allContainers {

		container := &Container{
			Id:      c.ID,
			Name:    c.Names[0],
			State:   c.State,
			Status:  c.Status,
			Created: c.Created,
		}
		if "running" == container.State {
			container.Running = true
		}

		containers = append(containers, container)
	}

	containerCatalog := &ContainerCatalog{
		Containers: containers,
	}

	return containerCatalog, nil
}

func (cs *containerService) GetContainerStats(csr *ContainerStatsRequest, stream DockerService_GetContainerStatsServer) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()
	response, err := cli.ContainerStats(ctx, csr.Container, true)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)
	for {
		var containerStats *types.StatsJSON
		if err := dec.Decode(&containerStats); err != nil {
			dec = json.NewDecoder(io.MultiReader(dec.Buffered(), response.Body))
			if err == io.EOF {
				break
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if err = stream.Send(convert(containerStats)); err != nil {
			return err
		}
	}

	return nil

}

