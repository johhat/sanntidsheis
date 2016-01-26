# sanntidsheis
Heisprosjekt i TTK4145 Sanntidsprogrammering

## Moduler
- Nettverksmodul
- Watchdog?
- Kø/oppførsel (keyword REDUNDANS)
- Håndtering av input (knapper)
- Hardware interface (driveren er c-kode)

### Samarbeidende heiser
- Valg av strategi med 1, 2 og 3 heiser i systemet (prinsipielt n heiser)
- Må vi ha en master som tar seg av ordrehåndtering (delegering av oppgaver til hver heis), eller kan vi løse problemet uten? En løsning på dette problemet vil gjøre at "network partitioning ikke vil være noe problem
- Skal heiser fungere uten samarbeid hvis nettverk ikke fungerer?

### Mulige problem-caser
- Noen trekker ut en nettverkskabel (og senere plugger den tilbake)
- Et program kræsjer (bør ikke skje, men må tas hensyn til av de andre heisene), blir tilsvarende om strømmen på en hel arbeidsplass skrus av
- En heis "står fast" og gjør ikke som programmet ønsker
- Brukeren trykker som en gal på kontrollpanel(ene)
- Strømmen på en enkelt heis skrus av (og senere på igjen)

## Assumptions
- At least one elevator is always alive
- Stop button & Obstruction switch are disabled
- No multiple simultaneous errors
- No network partitioning

## Choices
- When stopping at a flow, all orders are cleared (everyone enters/exits the elevator)
- What does the elevator do if it cannot connect to the network during initialization? 
