package main

import (
	"github.com/docker/docker/client"
	"context"
	"github.com/gorilla/websocket"
	"github.com/docker/docker/api/types"
	"strconv"
	"time"
	"log"
	"flag"
	"net/http"
	"net"
	"strings"
	"io"
)

type ContainerStatus struct {
	ID      string `json:"Id"`
	Name    string
	Created string
	State   string
	Status  string
}

type ctnLogOpt struct {
	containerID string
	logSwitch   bool
}

type SocketClient struct {
	Online     bool
	RemoteAddr net.Addr
	ClusterID  string
	clientMes  chan []ContainerStatus
	logStream  chan []byte
	ctnLogOpt  *ctnLogOpt
}

var addr = flag.String("addr", "localhost:8001", "http service address")
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var clientConns = make(map[*websocket.Conn]*SocketClient)

func main() {
	flag.Parse()
	apiCli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20033"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/api/ca.pem", "/Users/moclaf/.acs/certs/api/cert.pem", "/Users/moclaf/.acs/certs/api/key.pem"))
	asyncCli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20004"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/async/ca.pem", "/Users/moclaf/.acs/certs/async/cert.pem", "/Users/moclaf/.acs/certs/async/key.pem"))
	indiaCli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20025"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/prod/ca.pem", "/Users/moclaf/.acs/certs/prod/cert.pem", "/Users/moclaf/.acs/certs/prod/key.pem"))

	clis := map[string]*client.Client{
		"api":   apiCli,
		"async": asyncCli,
		"india": indiaCli,
	}

	var ctx = context.Background()
	logDelay, _ := time.ParseDuration("-10m")
	time_interval, _ := strconv.Atoi("10")
	ticker := time.NewTicker(time.Second * time.Duration(time_interval))

	// push msg of cluster status to all client
	go func() {
		for _ = range ticker.C {
			for k, v := range clientConns {
				go func() {
					if !v.pushStatusMes(clis, ctx) {
						delete(clientConns, k)
					}
					log.Println(v)
					judge, err := v.pushCtnLogs(clis, ctx, logDelay)
					if judge {
						if err != nil {
							v.ctnLogOpt = nil
						}
					}
				}()
			}
		}
	}()

	//push msg of containers logs to client

	log.Println("serverstart")
	http.HandleFunc("/echo", echo)
	//http.HandleFunc("/dockerlog", dockerlog)
	//http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func (sc *SocketClient) pushStatusMes(clis map[string]*client.Client, ctx context.Context) bool {
	log.Println("now is client", sc.RemoteAddr, sc.Online)
	if sc.Online {
		var message []ContainerStatus
		for _, item := range getcontainerlist(clis[sc.ClusterID], ctx) {
			message = append(message, ContainerStatus{item.ID, strings.Split(item.Names[0], "/")[len(strings.Split(item.Names[0], "/"))-1], time.Unix(item.Created, 0).Format("2006-01-02 15:04:05"), item.State, item.Status})
		}
		sc.clientMes <- message
	}
	return sc.Online
}

func getcontainerlist(client *client.Client, ctx context.Context) []types.Container {
	res, _ := client.ContainerList(ctx, types.ContainerListOptions{
		Size: true,
		All:  true,
	})
	return res
}

func (sc *SocketClient) pushCtnLogs(clis map[string]*client.Client, ctx context.Context, logDelay time.Duration) (bool, error) {
	if sc.ctnLogOpt == nil {
		return false, nil
	}
	reader, err := clis[sc.ClusterID].ContainerLogs(ctx, sc.ctnLogOpt.containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      strconv.FormatInt(time.Now().Add(logDelay).Unix(), 10),
		Follow:     true,
	})
	if err != nil {
		return true, err
	}
	defer reader.Close()
	m := make([]byte, 256)
	for {
		n, err := reader.Read(m)
		if err != nil {
			if err == io.EOF {
				sc.logStream <- m[:n]
			}
			return true, err
		}
		log.Println(m[:n])
		sc.logStream <- m[:n]
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	sc := &SocketClient{Online: true, RemoteAddr: c.RemoteAddr(), clientMes: make(chan []ContainerStatus, 5), ClusterID: "api", logStream: make(chan []byte), ctnLogOpt:nil }
	clientConns[c] = sc
	log.Println(clientConns[c].RemoteAddr, "has connected")
	go sc.getClientOnlineStatus(*c)
	go func() {
		for {
			ctnLog := <-sc.logStream
			log.Println(string(ctnLog))
			err = c.WriteJSON(string(ctnLog))
			if err != nil {
				log.Println("write log error:", err)
				sc.ctnLogOpt = nil
			}
		}
	}()
	for {
		mes := <-sc.clientMes
		err = c.WriteJSON(mes)
		if err != nil {
			log.Println("write status error:", err)
			break
		}
	}
}

func (sc *SocketClient) getClientOnlineStatus(c websocket.Conn) {
	for {
		_, req, err := c.ReadMessage()
		if err != nil {
			sc.Online = false
			log.Println("read:", err)
			log.Println(sc.RemoteAddr, "has disconnected")
			c.Close()
			break
		} else {
			if len(req) >= 64 {
				sc.ctnLogOpt = &ctnLogOpt{
					containerID: string(req),
					logSwitch:   true,
				}
			} else {
				sc.ClusterID = string(req)
			}

		}
	}
}
