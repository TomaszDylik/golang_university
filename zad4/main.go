package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// main uruchamia tylko szkielet i pokazuje graceful shutdown.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		fmt.Println("\n[SYSTEM] Otrzymano sygnał przerwania. Zamykanie...")
		cancel()
	}()

	var wg sync.WaitGroup

	fmt.Println("[SYSTEM] Etap 1: plan implementacji i architektura kanałów.")
	startStarterSystem(ctx, &wg)

	wg.Wait()
	fmt.Println("[SYSTEM] Wszystkie komponenty planu zamknięte. Koniec.")
}
