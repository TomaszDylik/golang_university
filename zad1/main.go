package main

import (
	"fmt"
	"math/rand"
)

func zad1() {
	// Część 1
	wygrane_zostaje := 0
	wygrane_zmienia := 0
	for i := 1; i <= 100000; i++ {
		wybranaLiczba := rand.Intn(3)
		wygranaLiczba := rand.Intn(3)

		if wygranaLiczba == wybranaLiczba {
			wygrane_zostaje++
		} else {
			wygrane_zmienia++
		}
	}
	fmt.Printf("Wygrane, gdy zostaje: %d\n", wygrane_zostaje)
	fmt.Printf("Wygrane, gdy zmienia: %d\n", wygrane_zmienia)

	// Część 2
	wygrane_zostaje = 0
	wygrane_zmienia = 0
	N := 100
	k := 90

	for i := 1; i <= 100000; i++ {
		nagrodaPudlo := rand.Intn(N)
		wybranePudlo := rand.Intn(N)

		otwarte := make([]int, N)
		otwarteLiczba := 0
		for j := 0; j < N && otwarteLiczba < k; j++ {
			if j != wybranePudlo && j != nagrodaPudlo {
				otwarte[j] = 1
				otwarteLiczba++
			}
		}

		if wybranePudlo == nagrodaPudlo {
			wygrane_zostaje++
		}

		pozostale := []int{}
		for j := 0; j < N; j++ {
			if j != wybranePudlo && otwarte[j] == 0 {
				pozostale = append(pozostale, j)
			}
		}

		nowePudlo := pozostale[rand.Intn(len(pozostale))]
		if nowePudlo == nagrodaPudlo {
			wygrane_zmienia++
		}
	}

	fmt.Printf("\n%d pudeł, %d otwieranych\n", N, k)
	fmt.Printf("Wygrane, gdy zostaje: %d\n", wygrane_zostaje)
	fmt.Printf("Wygrane, gdy zmienia: %d\n", wygrane_zmienia)
}

func main() {
	zad1()
}
