package sse

import "sync"

type Client struct {
	Channel chan interface{}
}

var (
	clients = make(map[string]*Client)
	mu      sync.RWMutex
)

func Register(id string) *Client {
	mu.Lock()
	defer mu.Unlock()

	client := &Client{
		Channel: make(chan interface{}, 20),
	}
	clients[id] = client
	return client
}

func Get(id string) *Client {
	mu.RLock()
	defer mu.RUnlock()
	return clients[id]
}

func Unregister(id string) {
	mu.Lock()
	defer mu.Unlock()

	if client, ok := clients[id]; ok {
		close(client.Channel)
		delete(clients, id)
	}
}

func Send(id string, data interface{}) {
	mu.RLock()
	client := clients[id]
	mu.RUnlock()

	if client != nil {
		select {
		case client.Channel <- data:
		default:
		}
	}
}
