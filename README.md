# sanntidsheis
Heisprosjekt i TTK4145 Sanntidsprogrammering

## Moduler
- Nettverksmodul
- Watchdog?
- Kø/oppførsel
- Håndtering av input (knapper)
- Hardware interface (driveren er c-kode)

### Samarbeidende heiser
- Valg av strategi med 1, 2 og 3 heiser i systemet

### Mulige problem-caser
- Noen trekker ut en nettverkskabel (og senere plugger den tilbake)
- Et program kræsjer (bør ikke skje, men må tas hensyn til av de andre heisene), blir tilsvarende om strømmen på en hel arbeidsplass skrus av
- En heis "står fast" og gjør ikke som programmet ønsker
- Brukeren trykker som en gal på kontrollpanel(ene)
- Strømmen på en enkelt heis skrus av (og senere på igjen)

## Assumptions
At least one elevator is always alive
Stop button & Obstruction switch are disabled
No multiple simultaneous errors
No network partitioning
