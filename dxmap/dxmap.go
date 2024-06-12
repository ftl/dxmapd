package dxmap

import (
	"context"
	"log"
	"net/http"

	"github.com/ftl/godxmap"
)

type LoggedCall struct {
	Call         string
	FrequencyKHz float64
}

type PartialCall struct {
	Call string
}

type DXSpot struct {
	Spot         string
	Spotter      string
	FrequencyKHz float64
	Comments     string
}

type Gab struct {
	From    string
	To      string
	Message string
}

func Run(ctx context.Context, address string) chan<- any {
	messages := make(chan any)

	server := godxmap.NewServer(address)
	go serveMessages(ctx, server, messages)
	go runServer(ctx, server, messages)

	return messages
}

func runServer(ctx context.Context, server *godxmap.Server, messages chan any) {
	defer close(messages)
	_, cancel := context.WithCancelCause(ctx)

	err := server.Serve()
	if err != nil && err != http.ErrServerClosed {
		cancel(err)
	} else {
		cancel(nil)
	}
}

func serveMessages(ctx context.Context, server *godxmap.Server, messages <-chan any) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case m, ok := <-messages:
			if !ok {
				break loop
			}
			serveMessage(m, server)
		}
	}

	err := server.Close()
	if err != nil {
		log.Printf("cannot close websocket server: %v", err)
	}
}

func serveMessage(m any, server *godxmap.Server) {
	switch message := m.(type) {
	case LoggedCall:
		server.ShowLoggedCall(message.Call, message.FrequencyKHz)
	case PartialCall:
		server.ShowPartialCall(message.Call)
	case DXSpot:
		server.ShowDXSpot(message.Spot, message.Spotter, message.FrequencyKHz, message.Comments)
	case Gab:
		server.ShowGab(message.From, message.To, message.Message)
	default:
		log.Printf("unknown message type: %T", message)
	}
}
