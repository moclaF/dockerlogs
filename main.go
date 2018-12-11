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
)

type ContainerStatus struct {
	ID      string `json:"Id"`
	Name    string
	Created string
	State   string
	Status  string
}

type SocketClient struct {
	Online     bool
	RemoteAddr net.Addr
	ClusterID  string
	clientMes  chan []ContainerStatus
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
	ctx := context.Background()
	apiCli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20033"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/api/ca.pem", "/Users/moclaf/.acs/certs/api/cert.pem", "/Users/moclaf/.acs/certs/api/key.pem"))
	asyncCli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20004"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/async/ca.pem", "/Users/moclaf/.acs/certs/async/cert.pem", "/Users/moclaf/.acs/certs/async/key.pem"))
	indiaCli, _ := client.NewClientWithOpts(client.WithHost("tcp://master1g7.cs.cn-shenzhen.aliyuncs.com:20025"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/prod/ca.pem", "/Users/moclaf/.acs/certs/prod/cert.pem", "/Users/moclaf/.acs/certs/prod/key.pem"))

	clis := map[string]*client.Client{
		"api":   apiCli,
		"async": asyncCli,
		"india": indiaCli,
	}

	time_interval, _ := strconv.Atoi("5")
	ticker := time.NewTicker(time.Second * time.Duration(time_interval))

	go func() {
		for _ = range ticker.C {
			for k, v := range clientConns {
				log.Println("now is client", clientConns[k].RemoteAddr, clientConns[k].Online)
				if v.Online {
					var message []ContainerStatus
					for _, item := range getcontainerlist(clis[v.ClusterID], ctx) {
						message = append(message, ContainerStatus{item.ID, strings.Split(item.Names[0], "/")[len(strings.Split(item.Names[0], "/"))-1], time.Unix(item.Created, 0).Format("2006-01-02 15:04:05"), item.State, item.Status})
					}
					v.clientMes <- message
				} else if !v.Online {
					delete(clientConns, k)
				}
			}
		}
	}()
	log.Println("serverstart")
	http.HandleFunc("/echo", echo)
	//http.HandleFunc("/dockerlog", dockerlog)
	//http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func getcontainerlist(client *client.Client, ctx context.Context) []types.Container {
	res, _ := client.ContainerList(ctx, types.ContainerListOptions{
		Size: true,
		All:  true,
	})
	return res
}

//func home(w http.ResponseWriter, r *http.Request) {
//	http.ServeFile(w, r, "./index.html")
//}

func dockerlog() {

}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	sc := &SocketClient{Online: true, RemoteAddr: c.RemoteAddr(), clientMes: make(chan []ContainerStatus, 5), ClusterID: "api"}
	clientConns[c] = sc
	log.Println(clientConns[c].RemoteAddr, "has connected")
	go getClientOnlineStatus(*c, sc)
	for {
		mes := <-sc.clientMes
		err = c.WriteJSON(mes)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func getClientOnlineStatus(c websocket.Conn, sc *SocketClient) {
	for {
		_, req, err := c.ReadMessage()
		if err != nil {
			sc.Online = false
			log.Println("read:", err)
			log.Println(sc.RemoteAddr, "has disconnected")
			c.Close()
			break
		} else {
			if len(req) == 64 {
			} else {
				sc.ClusterID = string(req)
			}

		}
	}
}
