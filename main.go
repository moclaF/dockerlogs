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
)

type SocketClient struct {
	Online     bool
	RemoteAddr net.Addr
	clientMes  chan []types.Container
}

var addr = flag.String("addr", "localhost:8000", "http service address")
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var clientConns = make(map[*websocket.Conn]*SocketClient)

func main() {
	flag.Parse()
	ctx := context.Background()
	cli, _ := client.NewClientWithOpts(client.WithHost("tcp://master3g7.cs.cn-shenzhen.aliyuncs.com:20119"), client.WithTLSClientConfig("/Users/moclaf/.acs/certs/test/ca.pem", "/Users/moclaf/.acs/certs/test/cert.pem", "/Users/moclaf/.acs/certs/test/key.pem"))

	time_interval, _ := strconv.Atoi("5")
	ticker := time.NewTicker(time.Second * time.Duration(time_interval))

	go func() {
		for _ = range ticker.C {
			message := getcontainerlist(cli, ctx)
			for k,v := range clientConns {
				log.Println("now is client", clientConns[k].RemoteAddr,clientConns[k].Online)
				if v.Online {
					v.clientMes <- message
				} else if !v.Online {
					delete(clientConns, k)
				}
			}
		}
	}()
	log.Println("serverstart")
	http.HandleFunc("/echo", echo)
	//http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func getcontainerlist(client *client.Client, ctx context.Context) []types.Container {
	res, _ := client.ContainerList(ctx, types.ContainerListOptions{
		Size: true,
		All:  true,
	})
	log.Println("get healthy check")
	return res
}

//func home(w http.ResponseWriter, r *http.Request) {
//	http.ServeFile(w, r, "./index.html")
//}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	sc := &SocketClient{Online: true, RemoteAddr: c.RemoteAddr(), clientMes: make(chan []types.Container, 5)}
	clientConns[c]=sc
	log.Println(clientConns[c].RemoteAddr,"has connected")
	go getClientOnlineStatus(*c, sc)
	for {
		mes := <- sc.clientMes
		err = c.WriteJSON(mes[0].Status)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func getClientOnlineStatus(c websocket.Conn, sc *SocketClient) {
	for {
		_, _,err := c.ReadMessage()
		if err != nil {
			sc.Online = false
			log.Println("read:", err)
			log.Println(sc.RemoteAddr,"has disconnected")
			c.Close()
			break
		}
	}
}
