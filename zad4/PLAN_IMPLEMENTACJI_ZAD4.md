# Plan implementacji zadania 4

## 1. Co wynika z polecenia i uwag prowadzacego

Po analizie `zadanie4 1.pdf`, diagramu `mermaid-diagram.pdf` i dodatkowych uwag trzeba przyjac nastepujace zalozenia projektowe:

1. Kazdy wazny element systemu ma dzialac w osobnej gorutynie.
2. Komunikacja ma isc przez kanaly, a nie przez zmienne globalne.
3. System ma dwie skale czasu:
   - `WeatherStep = 5ms`, czyli okolo 5 minut symulacji,
   - `GridStep = 100ms`, czyli 1 godzina symulacji.
4. W jednej godzinie symulacji miesci sie dokladnie `12` krokow pogodowych.
5. W projekcie wystarczy jeden rodzaj OZE. Dla uproszczenia przyjmuje farme wiatrowa.
6. `Predictor` ma byc mostem miedzy skala pogodowa i skala decyzji GridHub.
7. `GridHub` jest centralnym miejscem podejmowania decyzji.
8. `ESS` i elektrownia konwencjonalna sa wykonawcami decyzji GridHub.
9. Konsumenci maja dzialac w modelu `fan-in` i wysylac `DemandReport` do jednego kanalu.
10. System ma obslugiwac dynamiczna rejestracje nowych konsumentow.
11. `DataLogger` ma zapisywac dane asynchronicznie i przy zamykaniu dopchnac wszystko z kanalu przed `Flush()`.
12. Zapis do pliku ma byc robiony przez `encoding/csv` albo `encoding/json`.

## 2. Co pokazuje diagram

Diagram dzieli system na cztery warstwy:

1. Warstwa wejsciowa:
   - `WeatherStation`
   - `Broadcaster Pub/Sub`
   - `Farma OZE`
   - `Predictor`

2. Warstwa decyzyjna:
   - `GridHub`

3. Warstwa wykonawcza i bufor:
   - `Magazyn energii ESS`
   - `Elektrownia weglowa / konwencjonalna`
   - `Konsumenci`

4. Warstwa archiwizacji:
   - `DataLogger CSV/JSON`

Najwazniejszy przeplyw jest taki:

1. `WeatherStation -> Broadcaster`
2. `Broadcaster -> OZE`
3. `Broadcaster -> Predictor`
4. `OZE -> GridHub`
5. `Predictor -> GridHub`
6. `Konsumenci -> GridHub`
7. `GridHub -> Konsumenci`
8. `GridHub -> ESS`
9. `ESS -> GridHub`
10. `GridHub -> Elektrownia`
11. `Elektrownia -> GridHub`
12. `GridHub -> DataLogger`

## 3. Ocena obecnego szkicu w folderze zad4

To, co jest teraz, jest dobrym poczatkiem, ale nie pokrywa jeszcze calego planu:

1. `consts.go` jest poprawny i ma juz dobra skale czasu, lacznie z poprawka `12 krokow na godzine`.
2. `interfaces.go` zawiera wymagane interfejsy, ale logger powinien pracowac na strukturze logu, a nie na zwyklym stringu.
3. `models.go` mial za malo struktur komunikacyjnych. Brakowalo kontraktow dla ESS, elektrowni, logowania, produkcji i dynamicznej rejestracji.
4. `system.go` pokazywal tylko fragment kanalow. Brakowalo m.in. `registrationChan`, kanalow ESS, kanalow elektrowni i odpowiedzi dla konsumentow.
5. Brakowalo pliku startowego w glownym folderze `zad4`, wiec etap 1 nie byl domkniety nawet jako minimalny szkielet.

## 4. Docelowy podzial implementacji na etapy

Poniżej jest plan pracy, ktory mozna potem realizowac krok po kroku.

### Etap 1. Fundamenty architektury

Cel:
Przygotowac kontrakty calego systemu bez pisania logiki symulacji.

Zakres:

1. Stale czasu.
2. Typy danych przesylanych przez kanaly.
3. Interfejsy komponentow.
4. Mapa wszystkich kanalow.
5. `main.go` z `context.Context`, sygnalem i `WaitGroup`.

Pliki:

1. `consts.go`
2. `models.go`
3. `interfaces.go`
4. `system.go`
5. `main.go`

Jak to dziala:

1. Program startuje i tworzy kontekst anulowania.
2. Tworzona jest mapa kanalow zgodna z diagramem.
3. Jeszcze nie ma prawdziwej logiki, ale wiadomo juz, jak beda polaczone komponenty.
4. Po `Ctrl+C` wszystkie gorutyny koncza dzialanie przez `ctx.Done()`.

Warunek zaliczenia etapu:

1. Kod sie kompiluje.
2. W projekcie sa wszystkie podstawowe DTO i kanaly.
3. Jest gotowy plan dalszej implementacji.

### Etap 2. WeatherStation + Broadcaster + jedna farma OZE

Cel:
Uruchomic warstwe wejsciowa zgodnie z diagramem.

Zakres:

1. `WeatherStation` generuje ciagly stan pogody co `WeatherStep`.
2. `Broadcaster` rozglasza dane do subskrybentow przez `select` z `default`.
3. Jedna farma wiatrowa odbiera pogodę i wylicza aktualna moc.
4. Farma wysyla `ProductionReport` do `GridHub`.

Jak to dziala:

1. WeatherStation trzyma stan wewnetrzny, a nie losuje wszystkiego od zera.
2. Broadcaster rozsyla dane jednoczesnie do Predictora i farmy wiatrowej.
3. Jesli subskrybent nie nadaza, jego paczka moze zostac porzucona.
4. To zapobiega blokowaniu calego systemu.

Warunek zaliczenia etapu:

1. W konsoli widac przeplyw danych pogodowych.
2. Farma raportuje moc.
3. Broadcaster nie blokuje sie na wolnym odbiorcy.

### Etap 3. Predictor

Cel:
Dodac most miedzy szybka pogoda i wolniejsza siecia.

Zakres:

1. Predictor subskrybuje pogode w rytmie `WeatherStep`.
2. Buforuje ostatnie `12` odczytow.
3. Raz na `GridStep` tworzy `ForecastReport`.
4. Prognoza dotyczy kilku krokow do przodu, np. `ForecastHorizon = 5`.

Jak to dziala:

1. Predictor zbiera godzine historii pogodowej.
2. Na tej podstawie liczy trend wzrostu lub spadku produkcji OZE.
3. Nie wysyla prognozy po kazdym odczycie, tylko raz na krok sieciowy.

Warunek zaliczenia etapu:

1. `forecastChan` dziala poprawnie.
2. GridHub dostaje jedna prognoze na jeden `GridStep`.

### Etap 4. GridHub - rdzen decyzyjny

Cel:
Zrobic centralna petle `select`, ktora zbiera dane i liczy bilans.

Zakres:

1. Odbior `ProductionReport`.
2. Odbior `ForecastReport`.
3. Odbior `DemandReport`.
4. Obliczanie bilansu co `GridStep`.
5. Przygotowanie miejsc pod ESS, elektrownie i load shedding.

Jak to dziala:

1. GridHub zbiera ostatni stan produkcji, popytu i prognozy.
2. Na tickerze godzinowym liczy, czy jest nadwyzka czy niedobor.
3. Jeszcze bez pelnej reakcji wykonawczej, ale z czytelna struktura decyzji.

Warunek zaliczenia etapu:

1. GridHub stabilnie pracuje w petli `select`.
2. W logach widac bilans sieci co `GridStep`.

### Etap 5. ESS i elektrownia konwencjonalna

Cel:
Dodac warstwe wykonawcza.

Zakres:

1. `ESS` reaguje na `Charge` i `Discharge`.
2. Pilnuje granic `SoC` od `0.0` do `1.0`.
3. Elektrownia przyjmuje `Start/Stop`.
4. Elektrownia ma stan `Off`, `WarmUp`, `On`.
5. GridHub zaczyna sterowac tymi komponentami.

Jak to dziala:

1. Przy nadwyzce energia trafia do ESS.
2. Przy niedoborze GridHub probuje rozladowac ESS.
3. Gdy prognoza przewiduje spadek OZE, GridHub moze wczesniej uruchomic elektrownie.

Warunek zaliczenia etapu:

1. ESS nie przeładowuje sie i nie schodzi poniżej zera.
2. Elektrownia ma widoczny etap rozgrzewania.

### Etap 6. Konsumenci + fan-in + dynamiczna rejestracja

Cel:
Dodac pelna warstwe odbiorcow energii.

Zakres:

1. Trzy typy konsumentow: `Critical`, `Industrial`, `Residential`.
2. Kazdy konsument jest osobna gorutyna.
3. Kazdy wysyla `DemandReport` do wspolnego `demandChan`.
4. GridHub odpowiada przez indywidualny kanal zwrotny.
5. Nowy konsument moze zostac dodany przez `registrationChan` bez restartu systemu.

Jak to dziala:

1. Konsument wylicza swoje zapotrzebowanie co `GridStep`.
2. Wysyla raport do GridHub.
3. Czeka na `SupplyStatus`.
4. Dynamiczna rejestracja aktualizuje mape konsumentow w GridHub.

Warunek zaliczenia etapu:

1. Wszystkie profile konsumentow dzialaja.
2. Fan-in dziala stabilnie.
3. Nowy konsument moze dolaczyc w trakcie symulacji.

### Etap 7. Load Shedding

Cel:
Dodac obsluge awarii i niedoboru mocy.

Zakres:

1. Sortowanie konsumentow od najnizszego priorytetu.
2. Odcinanie odbiorcow az do powrotu bilansu.
3. Wysylanie `SupplyStatus{AllocatedMW: 0, LoadShed: true}`.
4. Przywrocenie zasilania w kolejnych iteracjach, gdy bilans wraca do normy.

Jak to dziala:

1. GridHub sprawdza, czy sama produkcja i ESS wystarczaja.
2. Jesli nie, najpierw odcina `Residential`, potem `Industrial`, a `Critical` zostawia na koniec.
3. Zdarzenie odciecia jest od razu logowane.

Warunek zaliczenia etapu:

1. Load shedding dziala deterministycznie wedlug priorytetow.
2. Odcieci konsumenci dostaja poprawna odpowiedz.

### Etap 8. DataLogger + CSV/JSON + zamykanie systemu

Cel:
Domknac projekt pod wzgledem trwalosci danych i shutdownu.

Zakres:

1. Dedykowany worker loggera.
2. `encoding/csv` albo `encoding/json`.
3. `bufio.Writer`.
4. Przy shutdownie logger odbiera i zapisuje wszystko, co jeszcze jest w kanale.
5. Dopiero potem robi `Flush()` i zamyka plik.

Jak to dziala:

1. Inne gorutyny wrzucaja wpisy do `logChan`.
2. Logger zapisuje asynchronicznie.
3. Po `cancel()` system zamyka producentow logow.
4. Logger schodzi jako ostatni.

Warunek zaliczenia etapu:

1. Dane trafiaja do `CSV` albo `JSON`.
2. Po przerwaniu programu nic nie ginie z loggera.

### Etap 9. Raportowanie i test calosci

Cel:
Sprawdzic, czy cala symulacja dziala stabilnie.

Zakres:

1. Raport stanu w konsoli co `N` krokow `GridStep`.
2. Scenariusze testowe:
   - normalna praca,
   - spadek wiatru,
   - puste ESS,
   - rozgrzewanie elektrowni,
   - load shedding,
   - dolaczenie nowego konsumenta,
   - zamkniecie `Ctrl+C`.

Warunek zaliczenia etapu:

1. System przechodzi symulacje bez deadlocka.
2. Logger zapisuje koncowe dane.
3. Dzialanie zgadza sie z diagramem i poleceniem.

## 5. Kolejnosc pracy ze mna krok po kroku

Najrozsadniej robic to tak:

1. Domknac etap 1.
2. Zrobic WeatherStation i Broadcaster.
3. Dodac jedna farme wiatrowa.
4. Dopisac Predictora.
5. Zrobic GridHub bez load sheddingu.
6. Dodac ESS i elektrownie.
7. Dopisac konsumentow i rejestracje dynamiczna.
8. Dopisac load shedding.
9. Dopisac logger CSV/JSON i test integracyjny.

## 6. Proponowane commity

1. `zad4: domknij kontrakty i kanaly etapu 1`
   - pliki: `consts.go`, `models.go`, `interfaces.go`, `system.go`, `main.go`
   - sens: spójny fundament pod cala reszte

2. `zad4: dodaj weather station i broadcaster`
   - pliki: nowy plik warstwy pogodowej + ewentualnie `system.go`
   - sens: uruchomienie pierwszego realnego przeplywu danych

3. `zad4: dodaj farme wiatrowa i raport produkcji`
   - pliki: implementacja OZE
   - sens: pierwszy realny producent energii

4. `zad4: dodaj predictor i prognozy gridstep`
   - pliki: predictor
   - sens: most miedzy skalami czasu

5. `zad4: dodaj gridhub i bilansowanie`
   - pliki: gridhub
   - sens: centralna logika decyzyjna

6. `zad4: dodaj ess i elektrownie konwencjonalna`
   - pliki: ess, plant
   - sens: wykonawcy decyzji GridHub

7. `zad4: dodaj konsumentow i dynamiczna rejestracje`
   - pliki: consumers
   - sens: domkniecie warstwy popytu

8. `zad4: dodaj load shedding oraz logger csv json`
   - pliki: logger, gridhub, consumers
   - sens: scenariusze awaryjne i archiwizacja

## 7. Co jest rozpoczęte teraz

W tym kroku zaczety jest juz etap 1:

1. uzupelnione sa brakujace DTO komunikacyjne,
2. logger ma juz strukture wpisu zamiast zwyklego stringa,
3. jest kompletna mapa kanalow zgodna z diagramem,
4. jest glowny plik startowy z graceful shutdown,
5. szkic zostaje prosty i bez logiki biznesowej, zeby nie wyprzedzac kolejnych etapow.

## 8. Stan po obecnej zmianie

Teraz rusza etap 2 w wersji minimalnej i studenckiej:

1. dodana jest `WeatherStation`,
2. dodany jest prosty `Broadcaster` z `select` i `default`,
3. dodana jest jedna `WindFarm`,
4. dane pogodowe i raport mocy sa widoczne w konsoli,
5. `Predictor`, `GridHub`, `ESS` i logger nadal sa zostawione na pozniejsze etapy.

Proponowany commit dla tej zmiany:

`zad4: dodaj etap 2 z pogoda broadcasterem i farma wiatrowa`

Pliki:

1. `weather.go`
2. `system.go`
3. `main.go`
4. `PLAN_IMPLEMENTACJI_ZAD4.md`

Rationale:

1. etap 2 daje pierwszy realny przeplyw danych przez kanaly,
2. nadal nie robi projektu zbyt rozbudowanego,
3. zostawia proste miejsce na kolejny etap z Predictorem.

## 9. Stan po kolejnym kroku

Teraz rusza etap 3 w wersji minimalnej:

1. dodany jest prosty `SimplePredictor`,
2. predictor zbiera `12` odczytow pogodowych,
3. po kazdym takim pakiecie wysyla `ForecastReport`,
4. prognoza jest na razie tylko wypisywana w konsoli, bo `GridHub` jeszcze nie istnieje,
5. dzieki temu kolejny etap bedzie mogl juz czytac gotowy `forecastChan`.

Proponowany commit dla tej zmiany:

`zad4: dodaj prosty predictor i podglad prognoz`

Pliki:

1. `predictor.go`
2. `system.go`
3. `main.go`
4. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 10. Stan po obecnym kroku

Teraz rusza etap 4 w wersji minimalnej:

1. dodany jest prosty `GridHub`,
2. `GridHub` zbiera `ProductionReport` i `ForecastReport`,
3. raz na `GridStep` wypisuje prosty bilans,
4. na tym etapie popyt jest jeszcze pomijany, wiec bilans = produkcja,
5. to przygotowuje miejsce pod kolejny etap z konsumentami.

Proponowany commit dla tej zmiany:

`zad4: dodaj prosty gridhub i bilansowanie`

Pliki:

1. `gridhub.go`
2. `system.go`
3. `main.go`
4. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 11. Stan po obecnym kroku

Teraz dochodzi prosty popyt przed pelnym etapem konsumentow:

1. dodany jest jeden `SimpleConsumer`,
2. konsument wysyla `DemandReport` co `GridStep`,
3. `GridHub` odbiera popyt i odsyla prosty `SupplyStatus`,
4. bilans przestaje byc sama produkcja i zaczyna liczyc `produkcja - popyt`,
5. dalej jest to wersja uproszczona, bez rejestracji dynamicznej i bez load sheddingu.

Proponowany commit dla tej zmiany:

`zad4: dodaj prostego konsumenta i popyt w gridhub`
Pliki:

1. `consumer.go`
2. `gridhub.go`
3. `system.go`
4. `main.go`
5. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 12. Etap 6 — fan-in dla wielu konsumentow (ZROBIONE)

Przed load sheddingiem i loggerem domkniety zostal fan-in popytu:

1. sa trzy profile konsumentow (`Critical`, `Industrial`, `Residential`),
2. kazdy dziala jako osobna gorutyna i wysyla `DemandReport` do jednego `DemandChan`,
3. `GridHub` trzyma popyt w mapie i liczy sume zapotrzebowania,
4. bilans przechodzi na realne `produkcja - popyt` dla wielu odbiorcow,
5. ten krok przygotowal baze pod etap 7 (load shedding).

Proponowany commit dla tej zmiany:

`zad4: dodaj fan-in dla wielu prostych konsumentow`

Pliki:

1. `consumer.go`
2. `gridhub.go`
3. `system.go`
4. `main.go`
5. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 13. Etap 7 — load shedding w GridHub (ZROBIONE)

GridHub teraz rozdziela dostepna moc wedlug priorytetu odbiorcy:

1. w metodzie `allocate()` lista zadan z `pendingDemand` jest sortowana po priorytecie rosnaco (1=Critical pierwszy),
2. kazdy odbiorca otrzymuje moc az do wyczerpania billansu,
3. jesli pradu zabraknie, odbiorca dostaje `SupplyStatus{LoadShed: true, AllocatedMW: 0}` z powodem `"load shedding"`,
4. kazdy odciecia jest rowniez logowane do `logOut` jako `LogEntry{Event: "load_shed"}` (non-blocking select),
5. tiker GridStep loguje zdarzenie `"balance"` z aktualnym bilansem zanim wywola `allocate()`.

Proponowany commit dla tej zmiany:

`zad4: dodaj load shedding wedlug priorytetu w GridHub`

Pliki:

1. `gridhub.go`
2. `system.go`
3. `main.go`
4. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 14. Etap 8 — DataLogger JSON lines (ZROBIONE)

Asynchroniczny zapis zdarzen sieciowych do pliku `grid_log.jsonl`:

1. nowy plik `logger.go` z typem `FileDataLogger` (implementuje interfejs `DataLogger`),
2. gorutyna czyta `LogEntry` z kanalu i koduje jako JSON lines przez `bufio.Writer` + `json.NewEncoder`,
3. przy `ctx.Done()` pusta kanal w petli drenujacej (drain loop) przed powrotem,
4. `defer writer.Flush()` zapewnia wyczyszczenie bufora po zakonczeniu,
5. `SystemChannels` zawiera `LogChan chan LogEntry` (bufor 64),
6. `GridHub` otrzymuje `logOut chan<- LogEntry` i wysyla `balance` oraz `load_shed` zdarzenia non-blocking,
7. `DataLogger` jest uruchamiany jako pierwszy (przed `WeatherStation`) zeby nie zgubic wczesnych zdarzen.

Weryfikacja: po ~700 ms pracy i hard kill (Stop-Process) plik mial 73 KB poprawnych JSON lines. Przy graceful shutdown (Ctrl+C) drain loop dopychuje to co zostalo w buforze.

Proponowany commit dla tej zmiany:

`zad4: dodaj data logger json lines i logowanie zdarzen`

Pliki:

1. `logger.go`
2. `gridhub.go`
3. `system.go`
4. `main.go`
5. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 15. Etap 9 (mini) — dynamiczna rejestracja i test calosci (ZROBIONE)

Dodany zostal najmniejszy kolejny krok zgodny z etapem 9:

1. `GridHub` odbiera `ConsumerRegistration` z `registrationIn`,
2. nowa rejestracja trafia do mapy `registered` i jest logowana eventem `consumer_registered`,
3. po starcie systemu uruchamiany jest prosty scenariusz dynamicznego dolaczenia `residential_dyn_1`,
4. nowy konsument zaczyna wysylac popyt bez restartu programu,
5. etap dalej zostaje prosty (bez dorabiania nowych komponentow).

Proponowany commit dla tej zmiany:

`zad4: dodaj dynamiczna rejestracje konsumenta w runtime`

Pliki:

1. `gridhub.go`
2. `system.go`
3. `main.go`
4. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 16. Etap 9 (mini) — raport stanu co N krokow (ZROBIONE)

Najmniejszy kolejny krok z etapu 9 to czytelny raport stanu w konsoli:

1. w `consts.go` dodana jest stala `GridReportEvery = 5`,
2. `GridHub` ma pomocnicza metode `printReport(...)`,
3. co 5 krokow `GridStep` wypisywana jest linia `[REPORT]` z produkcja, popytem, prognoza, bilansem i liczba aktywnych odbiorcow,
4. ten sam stan jest dodatkowo logowany do `DataLogger` jako event `report`,
5. zmiana nie rozbudowuje architektury i zostaje w obecnym prostym stylu projektu.

Weryfikacja: krotki runtime pokazal linie `[REPORT]` dla `step=5` i `step=10`.

Proponowany commit dla tej zmiany:

`zad4: dodaj okresowy raport stanu gridu`

Pliki:

1. `consts.go`
2. `gridhub.go`
3. `system.go`
4. `main.go`
5. `PLAN_IMPLEMENTACJI_ZAD4.md`

## 17. Etap 5 i domkniecie projektu — ESS, plant i raport koncowy (ZROBIONE)

Projekt zostal domkniety tak, zeby pokrywac brakujaca warstwe wykonawcza i finalny przebieg calosci:

1. dodany jest `SimpleESS`, ktory reaguje na `charge` i `discharge`, pilnuje granic `SoC` oraz raportuje stan do `GridHub`,
2. dodana jest `SimplePlant` z etapami `Off`, `WarmUp`, `On` oraz prostym czasem rozgrzewania liczonym w krokach `GridStep`,
3. `GridHub` odbiera statusy ESS i plantu, wysyla komendy sterujace i liczy dostepna moc jako suma produkcji OZE, plantu i mozliwego rozladowania ESS,
4. co `GridReportEvery` krokow wypisywany jest raport `[REPORT]`, a przy zamknieciu `GridHub` drukuje podsumowanie `load_shed`, liczby konsumentow, `SoC` i stanu plantu,
5. poprawiony jest tez start dynamicznego konsumenta, zeby nie dodawac do `WaitGroup` po rozpoczeciu `Wait()`.

Weryfikacja:

1. `go build` przechodzi,
2. runtime pokazuje `ESS` charge/discharge, `PLANT` warm-up/on, `LOAD SHED`, dynamiczna rejestracje i raporty `[REPORT]`,
3. graceful shutdown konczy wszystkie gorutyny i zostawia dane w `grid_log.jsonl`.

Proponowany commit dla tej zmiany:

`ex4: finish grid simulation`

Pliki:

1. `consts.go`
2. `consumer.go`
3. `ess.go`
4. `plant.go`
5. `gridhub.go`
6. `logger.go`
7. `system.go`
8. `main.go`
9. `PLAN_IMPLEMENTACJI_ZAD4.md`