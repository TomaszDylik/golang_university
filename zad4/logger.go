package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileDataLogger odczytuje LogEntry z kanalu i zapisuje je jako JSON lines do pliku.
// Na zamkniecie systemu oproznia kanal przed zamknieciem pliku.
type FileDataLogger struct {
	logIn    <-chan LogEntry
	filename string
}

// Log jest zachowane dla zgodnosci z interfejsem; wpisy i tak trafiaja przez kanal.
func (d *FileDataLogger) Log(entry LogEntry) {
	_ = entry
}

// Flush zostaje jako lekka metoda zgodna z interfejsem.
func (d *FileDataLogger) Flush() error {
	return nil
}

// Run uruchamia goroutine loggera.
func (d *FileDataLogger) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		f, err := os.Create(d.filename)
		if err != nil {
			fmt.Printf("[LOGGER] Blad otwarcia pliku %s: %v\n", d.filename, err)
			return
		}
		defer f.Close()

		writer := bufio.NewWriter(f)
		defer writer.Flush()

		enc := json.NewEncoder(writer)

		for {
			select {
			case entry := <-d.logIn:
				if err := enc.Encode(entry); err != nil {
					fmt.Printf("[LOGGER] Blad zapisu: %v\n", err)
				}
			case <-ctx.Done():
				// Drain: opróżnij kanal zanim zamkniesz plik.
				for {
					select {
					case entry := <-d.logIn:
						enc.Encode(entry)
					default:
						fmt.Printf("[LOGGER] Zamknieto. Dane zapisane do %s\n", d.filename)
						return
					}
				}
			}
		}
	}()
}
