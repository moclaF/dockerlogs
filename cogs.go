package main

import (
	"github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types"
	"fmt"
	"time"
	"strconv"
			"bufio"
)

func main() {
	m, _ := time.ParseDuration("-1h")
	now := time.Now()
	m1 := now.Add(m)

	fmt.Println(m1)
	ctx := context.Background()
	cli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20033"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/api/ca.pem", "/Users/moclaf/.acs/certs/api/cert.pem", "/Users/moclaf/.acs/certs/api/key.pem"))
	reader, _ := cli.ContainerLogs(ctx, "d8261e9f8a377848e42118fcf5873a2039fc7d1e7a06ee76050778868e064931", types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      strconv.FormatInt(now.Unix(), 10),
		Follow:     true,
	})
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		fmt.Print(scanner.Text()[8:])
	}
}
