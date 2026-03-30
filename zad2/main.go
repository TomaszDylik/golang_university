package main

import (
	"fmt"
	"sort"
)

// typy danych
type Ocena struct {
	JurorID string
	Wartosc float64
}

type Utwor struct {
	Nazwa string
	Oceny []Ocena
}

type Uczestnik struct {
	ID        int
	Imie      string
	Repertuar []Utwor
}

// przypisywanie do uczestników
func DodajUtwor(u Uczestnik, nazwaUtworu string) Uczestnik {
	nowyRepertuar := append([]Utwor(nil), u.Repertuar...)

	nowyUtwor := Utwor{
		Nazwa: nazwaUtworu,
		Oceny: []Ocena{},
	}
	nowyRepertuar = append(nowyRepertuar, nowyUtwor)

	return Uczestnik{
		ID:        u.ID,
		Imie:      u.Imie,
		Repertuar: nowyRepertuar,
	}
}

func DodajOcene(u Uczestnik, nazwaUtworu string, ocena Ocena) Uczestnik {
	nowyRepertuar := make([]Utwor, len(u.Repertuar))

	for i, ut := range u.Repertuar {
		nowyUtwor := Utwor{
			Nazwa: ut.Nazwa,
			Oceny: append([]Ocena(nil), ut.Oceny...),
		}

		if ut.Nazwa == nazwaUtworu && ocena.Wartosc >= 0 && ocena.Wartosc <= 25 {
			nowyUtwor.Oceny = append(nowyUtwor.Oceny, ocena)
		}

		nowyRepertuar[i] = nowyUtwor
	}

	return Uczestnik{
		ID:        u.ID,
		Imie:      u.Imie,
		Repertuar: nowyRepertuar,
	}
}

// strategia oceniania
type StrategiaOceniania func(oceny []Ocena) float64

func ZwyklaSrednia(oceny []Ocena) float64 {
	if len(oceny) == 0 {
		return 0
	}

	suma := 0.0
	for _, o := range oceny {
		suma += o.Wartosc
	}

	return suma / float64(len(oceny))
}

func SredniaChopinowska(oceny []Ocena) float64 {
	if len(oceny) == 0 {
		return 0
	}

	suma := 0.0
	for _, o := range oceny {
		suma += o.Wartosc
	}
	sredniaWstepna := suma / float64(len(oceny))

	// margines
	const margines = 2.5
	sumaPoKorekcie := 0.0

	for _, o := range oceny {
		skorygowanaWartosc := o.Wartosc

		// za wysoka
		if skorygowanaWartosc > sredniaWstepna+margines {
			skorygowanaWartosc = sredniaWstepna + margines
		}
		// za niska
		if skorygowanaWartosc < sredniaWstepna-margines {
			skorygowanaWartosc = sredniaWstepna - margines
		}

		sumaPoKorekcie += skorygowanaWartosc
	}

	// srednia chopinowska
	return sumaPoKorekcie / float64(len(oceny))
}

// obliczanie wyników uczestnika
func ObliczWynikUczestnika(u Uczestnik, strategia StrategiaOceniania) float64 {
	wynikCalkowity := 0.0

	for _, utwor := range u.Repertuar {
		wynikCalkowity += strategia(utwor.Oceny)
	}

	return wynikCalkowity
}

// sortowanie uczestnikow z sort.slice
func PosortujUczestnikow(uczestnicy []Uczestnik, strategia StrategiaOceniania) []Uczestnik {
	kopia := append([]Uczestnik(nil), uczestnicy...)

	sort.Slice(kopia, func(i, j int) bool {
		wynikI := ObliczWynikUczestnika(kopia[i], strategia)
		wynikJ := ObliczWynikUczestnika(kopia[j], strategia)

		return wynikI > wynikJ
	})

	return kopia
}

// najlepszy w danym utworze
func NajlepszyWKonkretnymUtworze(uczestnicy []Uczestnik, nazwaUtworu string, strategia StrategiaOceniania) (Uczestnik, float64) {

	var najlepszy Uczestnik
	var maxWynik float64 = -1.0

	for _, u := range uczestnicy {
		for _, ut := range u.Repertuar {
			if ut.Nazwa == nazwaUtworu {
				wynik := strategia(ut.Oceny)

				if wynik > maxWynik {
					maxWynik = wynik
					najlepszy = u
				}
				break
			}
		}
	}

	return najlepszy, maxWynik
}

func main() {
	uczestnicy := []Uczestnik{
		{ID: 1, Imie: "uczestnik1", Repertuar: nil},
		{ID: 2, Imie: "uczestnik2", Repertuar: nil},
		{ID: 3, Imie: "uczestnik3", Repertuar: nil},
	}

	listaUtworow := []string{"utwor1", "utwor2", "utwor3"}

	// wypelnianie uczestnikow
	for i := range uczestnicy {
		u := uczestnicy[i]

		for _, nazwaUtworu := range listaUtworow {
			u = DodajUtwor(u, nazwaUtworu)

			// zroznicowanie ocen
			bazaOceny := 15.0 + float64(i*2)

			u = DodajOcene(u, nazwaUtworu, Ocena{"J1", bazaOceny})
			u = DodajOcene(u, nazwaUtworu, Ocena{"J2", bazaOceny + 2})
			u = DodajOcene(u, nazwaUtworu, Ocena{"J3", 25})
			u = DodajOcene(u, nazwaUtworu, Ocena{"J4", 2})
			u = DodajOcene(u, nazwaUtworu, Ocena{"J5", bazaOceny + 1})
		}

		uczestnicy[i] = u
	}

	posortowani := PosortujUczestnikow(uczestnicy, SredniaChopinowska)

	fmt.Println("Ranking z użyciem średniej chopinowskiej:")
	for pozycja, u := range posortowani {
		wynik := ObliczWynikUczestnika(u, SredniaChopinowska)
		fmt.Printf("%d. %s - Punkty: %.2f\n", pozycja+1, u.Imie, wynik)
	}

	fmt.Println("\nRanking z użyciem zwykłej średniej:")
	posortowaniZwykla := PosortujUczestnikow(uczestnicy, ZwyklaSrednia)
	for pozycja, u := range posortowaniZwykla {
		wynik := ObliczWynikUczestnika(u, ZwyklaSrednia)
		fmt.Printf("%d. %s - Punkty: %.2f\n", pozycja+1, u.Imie, wynik)
	}

	// wyznaczanie uczestnika z najwyzsza liczba pkt za utwor
	szukanyUtwor := "utwor1"
	fmt.Printf("\nNajlepsze wykonanie utworu: %s\n", szukanyUtwor)

	najlepszy, wynikZaUtwor := NajlepszyWKonkretnymUtworze(uczestnicy, szukanyUtwor, SredniaChopinowska)

	fmt.Printf("Zwycięzca: %s (Wynik: %.2f)\n", najlepszy.Imie, wynikZaUtwor)
}
