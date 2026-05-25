package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// main uruchamia obecny etap i pokazuje graceful shutdown.
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

	fmt.Println("[SYSTEM] Etap 9: kompletna symulacja sieci energetycznej.")
	startStarterSystem(ctx, &wg)

	wg.Wait()
	fmt.Println("[SYSTEM] Wszystkie uruchomione komponenty zamknięte. Koniec.")
}
