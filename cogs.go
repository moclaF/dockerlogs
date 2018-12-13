package main

import (
	"github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types"
					"io"
			"bytes"
	"fmt"
	"strconv"
	"time"
)

func main() {
	ctx := context.Background()
	cli, _ := client.NewClientWithOpts(client.WithHost("tcp://master3g7.cs.cn-shenzhen.aliyuncs.com:20119"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/test/ca.pem", "/Users/moclaf/.acs/certs/test/cert.pem", "/Users/moclaf/.acs/certs/test/key.pem"))
	reader, _ := cli.ContainerLogs(ctx, "985beeb97d65", types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since: strconv.Itoa(int(time.Now().Unix())),
	})

	var b bytes.Buffer
	io.Copy(&b, reader)
	fmt.Println(b.String())
}
