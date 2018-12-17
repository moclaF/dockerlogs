package main

import (
	"github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types"
	"fmt"
	"time"
	"strconv"
	"io"
	"os"
)

func main() {
	m, _ := time.ParseDuration("-1h")
	now := time.Now()
	m1 := now.Add(m)

	fmt.Println(m1)
	ctx := context.Background()
	cli, _ := client.NewClientWithOpts(client.WithHost("tcp://master3g7.cs.cn-shenzhen.aliyuncs.com:20119"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/test/ca.pem", "/Users/moclaf/.acs/certs/test/cert.pem", "/Users/moclaf/.acs/certs/test/key.pem"))
	reader, _ := cli.ContainerLogs(ctx, "2dc73402269a", types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      strconv.FormatInt(m1.Unix(), 10),
		Follow:     true,
	})

	p := make([]byte, )

	defer cli.Close()
	for {
		n, err := reader.Read(p)
		if err != nil {
			if err == io.EOF {
				fmt.Print(string(p[:n])) //should handle any remainding bytes.
				break
			}
			fmt.Println(err)
			os.Exit(1)
		}
		//fmt.Printf(string(p[:n]))
		fmt.Println(n)
	}
}
