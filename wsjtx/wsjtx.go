package wsjtx

import (
	"context"
	"log"
	"net"

	"github.com/ftl/dxmapd/dxmap"
	wsjtx "github.com/k0swe/wsjtx-go/v4"
)

func Run(ctx context.Context, udpAddr string) <-chan any {
	messages := make(chan any, 1)

	addr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		// TODO there must be a better way
		log.Fatal(err)
	}
	if addr.IP.IsMulticast() {
		log.Printf("using multicast UDP: %v", addr.IP)
	}

	server, err := wsjtx.MakeServerGiven(addr.IP, uint(addr.Port))
	if err != nil {
		// TODO there must be a better way
		log.Fatal(err)
	}

	wsjtxMessages := make(chan any, 1)
	errors := make(chan error, 1)
	go server.ListenToWsjtx(wsjtxMessages, errors)
	go handleWsjtx(ctx, messages, wsjtxMessages, errors)

	return messages
}

func handleWsjtx(ctx context.Context, messages chan any, wsjtxMessages <-chan any, errors <-chan error) {
	handler := &wsjtxMessageHandler{}
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case wsjtxMessage := <-wsjtxMessages:
			handler.handle(messages, wsjtxMessage)
		case err := <-errors:
			log.Printf("wsjt-x error: %v", err)
		}
	}

	close(messages)
}

type wsjtxMessageHandler struct {
	lastDXCall string
}

func (h *wsjtxMessageHandler) handle(messages chan<- any, wsjtxMessage any) {
	switch m := wsjtxMessage.(type) {
	case wsjtx.StatusMessage:
		if m.DxCall == "" {
			return
		}
		if m.DxCall == h.lastDXCall {
			return
		}
		h.lastDXCall = m.DxCall
		messages <- dxmap.PartialCall{Call: m.DxCall}
	case wsjtx.QsoLoggedMessage:
		if m.DxCall == "" {
			return
		}
		frequencyKHz := float64(m.TxFrequency) / 1000.0
		messages <- dxmap.LoggedCall{Call: m.DxCall, FrequencyKHz: frequencyKHz}
	default:
		// ignore
	}
}
