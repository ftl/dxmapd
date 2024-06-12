package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/ftl/dxmapd/dxmap"
	"github.com/ftl/dxmapd/wsjtx"
)

var rootFlags = struct {
	websocket string
	wsjtx     string
	trace     bool
}{}

var rootCmd = &cobra.Command{
	Use:   "dxmapd",
	Short: "A proxy for the HamDXMap wtSock protocol.",
	Run:   runWithContext(run),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootFlags.websocket, "websocket", "localhost:8073", "open the websocket for HamDXMap on this address")
	rootCmd.PersistentFlags().StringVar(&rootFlags.wsjtx, "wsjtx", "", "listen for wsjt-x UDP packets on this address")
	rootCmd.PersistentFlags().BoolVar(&rootFlags.trace, "trace", false, "write trace output to the console")
}

func run(ctx context.Context, cmd *cobra.Command, args []string) {
	messages := dxmap.Run(ctx, rootFlags.websocket)

	var wsjtxMessages <-chan any
	if rootFlags.wsjtx != "" {
		log.Printf("listening for wsjt-x messages on %v", rootFlags.wsjtx)
		wsjtxMessages = wsjtx.Run(ctx, rootFlags.wsjtx)
	} else {
		wsjtxMessages = make(chan any)
	}

	ticker := time.NewTicker(5 * time.Second)

loop:
	for {
		select {
		case now := <-ticker.C:
			_ = now
			// messages <- dxmap.Gab{From: "dxmapd", To: "HamDXMap", Message: fmt.Sprintf("Its %v", now)}
		case m := <-wsjtxMessages:
			messages <- m
		case <-ctx.Done():
			break loop
		}
	}

	err := ctx.Err()
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func runWithContext(f func(context.Context, *cobra.Command, []string)) func(*cobra.Command, []string) {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go handleCancelation(signals, cancel)

	return func(cmd *cobra.Command, args []string) {
		f(ctx, cmd, args)
	}
}

func handleCancelation(signals <-chan os.Signal, cancel context.CancelFunc) {
	count := 0
	for {
		select {
		case <-signals:
			count++
			if count == 1 {
				cancel()
			} else {
				log.Fatal("hard shutdown")
			}
		}
	}
}
