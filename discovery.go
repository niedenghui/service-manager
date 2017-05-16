package service_manager

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type Discovery struct {
	sync.RWMutex
	kapi   client.KeysAPI
	serviceName    string
	nodes  map[string]string
}

func NewDiscovery(serviceName string, endpoints []string) (*Discovery, error) {
	cfg := client.Config{
		Endpoints:               endpoints,
		HeaderTimeoutPerRequest: time.Second * 2,
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}


	Discovery := &Discovery{
		kapi:   client.NewKeysAPI(c),
		serviceName:    serviceName,
		nodes:  make(map[string]string),
	}
	Discovery.fetch()

	/// `fetch` Timer may work well too?
	go Discovery.watch()

	return Discovery, err
}

func (dis *Discovery) GetNodes() map[string]string {
	dis.RLock()
	defer dis.RUnlock()
	return dis.nodes
}


//private
func (dis *Discovery)getKey() string{
	return fmt.Sprintf("/%s/%s/", kRoot, dis.serviceName)
}

func (dis *Discovery) fetch() error {
	resp, err := dis.kapi.Get(context.Background(), dis.getKey(), nil)
	if err != nil {
		return err
	}
	if resp.Node.Dir {
		for _, v := range resp.Node.Nodes {
			dis.addNode(v.Key, v.Value)
		}
	}
	return err
}

func (dis *Discovery) watch() {
	watcher := dis.kapi.Watcher(dis.getKey(), &client.WatcherOptions{
		Recursive: true,
	})
	for {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}

		switch resp.Action {
			case "set", "update":
				dis.addNode(resp.Node.Key, resp.Node.Value)
				break
			case "expire", "delete":
				dis.delNode(resp.Node.Key)
				break
			default:
				log.Println("watchme!!!", "resp ->", resp)
		}
	}
}



func (dis *Discovery) addNode(key, isActive string) {
	dis.Lock()
	defer dis.Unlock()
	log.Println(key)
	log.Println(dis.getKey())
	service := strings.TrimPrefix(key, dis.getKey())
	log.Println(service)
	log.Println(isActive)
	dis.nodes[service] = isActive
}

func (dis *Discovery) delNode(key string) {
	dis.Lock()
	defer dis.Unlock()
	service := strings.TrimPrefix(key, dis.getKey())
	delete(dis.nodes, service)
}
