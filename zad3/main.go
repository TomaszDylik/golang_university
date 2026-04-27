package main

import (
	"errors"
	"fmt"
)

// DEKLARACJE STRUKTUR

type Pasazer struct {
	ID       int
	Imie     string
	Nazwisko string
}

type Samolot struct {
	ID                     string
	MaksymalnaLiczbaMiejsc int
}

type Rezerwacja struct {
	NumerRezerwacji string
	DanePasazera    *Pasazer
	PrzypisanyLot   *Lot
}

type Lot struct {
	NumerLotu  string
	Skad       string
	Dokad      string
	Maszyna    *Samolot
	Rezerwacje []Rezerwacja
}

type WszystkieLoty struct {
	DostepneLoty []*Lot
}

// interfejs

type Wyszukiwarka interface {
	ZnajdzLotyZLubDo(port string) []*Lot
	ZnajdzRezerwacjePasazera(pasazerID int) []Rezerwacja
}

// serwis

func (l *Lot) String() string {
	return fmt.Sprintf("Lot %s: %s -> %s | Samolot: %s | Wolne miejsca: %d/%d",
		l.NumerLotu, l.Skad, l.Dokad, l.Maszyna.ID, l.SprawdzWolneMiejsca(), l.Maszyna.MaksymalnaLiczbaMiejsc)
}

func (l *Lot) SprawdzWolneMiejsca() int {
	return l.Maszyna.MaksymalnaLiczbaMiejsc - len(l.Rezerwacje)
}

func (l *Lot) RezerwujMiejsce(p *Pasazer, numerRezerwacji string) error {
	if l.SprawdzWolneMiejsca() <= 0 {
		return errors.New("brak wolnych miejsc w samolocie")
	}

	// sprawdzenie podwójnej rezerwacji
	for _, rez := range l.Rezerwacje {
		if rez.DanePasazera.ID == p.ID {
			return errors.New("pasażer posiada już rezerwację na ten lot")
		}
	}

	nowaRezerwacja := Rezerwacja{
		NumerRezerwacji: numerRezerwacji,
		DanePasazera:    p,
		PrzypisanyLot:   l,
	}

	l.Rezerwacje = append(l.Rezerwacje, nowaRezerwacja)

	return nil
}

func (l *Lot) OdwolajRezerwacje(pasazerID int) error {
	for i, rez := range l.Rezerwacje {
		if rez.DanePasazera.ID == pasazerID {
			l.Rezerwacje = append(l.Rezerwacje[:i], l.Rezerwacje[i+1:]...)
			return nil
		}
	}
	return errors.New("nie znaleziono rezerwacji do odwołania")
}

func (system *WszystkieLoty) ZnajdzLotyZLubDo(port string) []*Lot {
	var wyniki []*Lot

	for _, lot := range system.DostepneLoty {
		if lot.Skad == port || lot.Dokad == port {
			wyniki = append(wyniki, lot)
		}
	}
	return wyniki
}

func (system *WszystkieLoty) ZnajdzRezerwacjePasazera(pasazerID int) []Rezerwacja {
	var wyniki []Rezerwacja

	for _, lot := range system.DostepneLoty {
		for _, rez := range lot.Rezerwacje {
			if rez.DanePasazera.ID == pasazerID {
				wyniki = append(wyniki, rez)
			}
		}
	}
	return wyniki
}

func WypiszWynikiWyszukiwania(wyszukiwarka Wyszukiwarka, port string) {
	znalezione := wyszukiwarka.ZnajdzLotyZLubDo(port)
	fmt.Printf("\nZnalezione loty dla portu: %s\n", port)

	if len(znalezione) == 0 {
		fmt.Println("Brak lotów.")
	}

	for _, lot := range znalezione {
		fmt.Println(lot)
	}
}

// main

func main() {
	pasazer1 := &Pasazer{ID: 1, Imie: "Jan", Nazwisko: "Kowalski"}
	pasazer2 := &Pasazer{ID: 2, Imie: "Anna", Nazwisko: "Nowak"}

	samolotDuzy := &Samolot{ID: "1", MaksymalnaLiczbaMiejsc: 150}
	samolotMaly := &Samolot{ID: "2", MaksymalnaLiczbaMiejsc: 1}

	lot1 := &Lot{NumerLotu: "LO111", Skad: "Warszawa", Dokad: "Paryz", Maszyna: samolotDuzy, Rezerwacje: []Rezerwacja{}}
	lot2 := &Lot{NumerLotu: "LO222", Skad: "Paryz", Dokad: "Londyn", Maszyna: samolotMaly, Rezerwacje: []Rezerwacja{}}
	lot3 := &Lot{NumerLotu: "LO333", Skad: "Londyn", Dokad: "Warszawa", Maszyna: samolotDuzy, Rezerwacje: []Rezerwacja{}}

	system := &WszystkieLoty{
		DostepneLoty: []*Lot{lot1, lot2, lot3},
	}

	fmt.Println("SYSTEM DZIAŁA...")

	fmt.Println("\n> Próba rezerwacji Jana Kowalskiego na lot LO111...")
	err := lot1.RezerwujMiejsce(pasazer1, "RES001")
	if err != nil {
		fmt.Println("Błąd:", err)
	} else {
		fmt.Println("Zarezerwowano.")
	}

	fmt.Println("\n> Próba PODWÓJNEJ rezerwacji Jana na ten sam lot...")
	err = lot1.RezerwujMiejsce(pasazer1, "RES002")
	if err != nil {
		fmt.Println("Zablokowano poprawnie. Powód:", err)
	}

	fmt.Println("\n> Anna rezerwuje mały lot (LO222)...")
	lot2.RezerwujMiejsce(pasazer2, "RES003")

	fmt.Println("> Jan próbuje zarezerwować ten sam mały lot (LO222), który jest już pełny...")
	err = lot2.RezerwujMiejsce(pasazer1, "RES004")
	if err != nil {
		fmt.Println("Zablokowano poprawnie. Powód:", err)
	}

	fmt.Println("\n--- Wszystkie rezerwacje pasażera ID: 1 (Jan) ---")
	rezerwacjeJana := system.ZnajdzRezerwacjePasazera(1)
	for _, r := range rezerwacjeJana {
		fmt.Printf("- %s (Nr rezerwacji: %s)\n", r.PrzypisanyLot, r.NumerRezerwacji)
	}

	fmt.Println("\n> Jan anuluje rezerwację na lot LO111...")
	err = lot1.OdwolajRezerwacje(1)
	if err == nil {
		fmt.Println("Odwołano rezerwację.")
	}

	WypiszWynikiWyszukiwania(system, "Paryz")
	WypiszWynikiWyszukiwania(system, "Madryt")
}
