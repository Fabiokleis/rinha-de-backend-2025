package listener

import (
	"context"
	"fmt"
	"os"
)

// notification listener

type Topic string

type PaymentServices struct {
	defaultUrl  *string
	fallbackUrl *string
}

type Handler func(ctx context.Context, id uint64, topic string) error

type TopicHandler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	callback Handler
	poolSize uint64
}

type Listener struct {
	ctx      context.Context
	handlers map[string]TopicHandler
}

func (l *Listener) subscribe(poolSize uint64, topic string, callback Handler) {
	ctx, cancel := context.WithCancel(l.ctx)
	th := TopicHandler{ctx, cancel, callback, poolSize}
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

	df := os.Getenv("PROCESSOR_DEFAULT_URL")
	fb := os.Getenv("PROCESSOR_FALLBACK_URL")
	if df == "" || fb == "" {
		fmt.Fprintln(os.Stderr, "failed to find default/fallback URL env.")
		os.Exit(1)
	}
	fmt.Println(df)
	fmt.Println(fb)

	ctxValue := &PaymentServices{
		defaultUrl:  &df,
		fallbackUrl: &fb,
	}

	l.ctx = context.WithValue(context.Background(), "services", ctxValue)
	l.handlers = make(map[string]TopicHandler)

	assignTopics()

	for key, th := range l.handlers {
		fmt.Printf("listener initialized handler with pool size: %v\n", th.poolSize)
		for i := range th.poolSize { // create pool for each handler subscription
			go th.callback(th.ctx, i, key)
		}
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
