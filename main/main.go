package main

import (
	"context"
	"fmt"
	"github.com/GeneralElectric/TrueConnect-Link/link"
	"os"
	"os/signal"
)

func main() {
	currentContext, cancelMethod := context.WithCancel(context.Background())
	killableContext := watchForKill(currentContext)
	client, err := link.NewClient(killableContext, os.Args)
	if err != nil {
		os.Exit(1)
	}

	fmt.Println(client.Start())
	cancelMethod()
	client.Dispose()
	os.Exit(client.GetExitCode())
}

func watchForKill(currentContext context.Context) context.Context {
	myContext, cancelMethod := context.WithCancel(context.Background())
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc)
	go func() {
		select {
		case sig, isOk := <-sigc:
			if isOk {
				fmt.Println("got signal " + sig.String())
				close(sigc)
			}
			break
		case <-currentContext.Done():
			close(sigc)
			break
		}
		cancelMethod()
	}()
	return myContext
}
