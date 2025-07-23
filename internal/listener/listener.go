package listener

import (
	"context"
	"fmt"
)

// notification listener

type Handler func(ctx context.Context, topic string) error

type TopicHandler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	callback Handler
}

type Listener struct {
	handlers map[string]TopicHandler
}

func (l *Listener) subscribe(topic string, callback Handler) {
	ctx, cancel := context.WithCancel(context.Background())
	th := TopicHandler{ctx, cancel, callback}
	l.handlers[topic] = th
	fmt.Printf("subscribe listener %v\n", topic)
}

func (l *Listener) unsubcribe(topic string) {
	if th, ok := l.handlers[topic]; ok {
		th.cancel()
		fmt.Printf("unsubscribe listener %v\n", topic)
	}
}

var l *Listener

func Listen() {
	l = &Listener{}
	l.handlers = make(map[string]TopicHandler)
	assignTopics()
	for key, th := range l.handlers {
		go th.callback(th.ctx, key)
	}
	fmt.Println("listener started")
}

func Stop() {
	for key := range l.handlers {
		l.unsubcribe(key)
	}
	l.handlers = make(map[string]TopicHandler)
	fmt.Println("listener stopped")
}
