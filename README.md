# sanntidsheis
Heisprosjekt i TTK4145 Sanntidsprogrammering

## Moduler
- Nettverksmodul
- Manager
- Heis
- Driver
- Trygg lagring

### Samarbeidende heiser
- Valg av strategi med 1, 2 og 3 heiser i systemet (prinsipielt n heiser)
- Må vi ha en master som tar seg av ordrehåndtering (delegering av oppgaver til hver heis), eller kan vi løse problemet uten? En løsning på dette problemet vil gjøre at "network partitioning ikke vil være noe problem
- Skal heiser fungere uten samarbeid hvis nettverk ikke fungerer?

### Mulige problem-caser
- Noen trekker ut en nettverkskabel (og senere plugger den tilbake), ordre legges til mens en heis er koblet fra nettverket
- Et program kræsjer/henger seg/feiler/gjør rare ting på grunn av en ukjent bug
- En heis "står fast" og gjør ikke som programmet ønsker (drivsnoren har røket, noen holder igjen heisen med hånda etc.) eller noe flytter heisen med hånda. Skal vi da gå inn i en "tilkall service"-modus der heisen permanent deaktiveres til den aktivt reaktiveres? Eller skal den prøve flere ganger
- Brukeren trykker som en gal på kontrollpanel(ene)
- Strømmen på en enkelt heis skrus av (og senere på igjen), ordre legges til mens dette skjer
- Strømmen på en hel arbeidsplass skrus av, ordre legges til mens dette skjer
- Kosmisk stråling endrer en bit i minnet
- Hvis man bruker harddisker: harddiskkræsj, korrupt fil

En ordre skal aldri mistes! Dette gjelder også interne ordre (som må tas når den aktuelle heisen kommer tilbake til normal drift).

## Assumptions
- At least one elevator is always alive
- Stop button & Obstruction switch are disabled
- No multiple simultaneous errors
- No network partitioning

## Choices
- When stopping at a floor, all orders are cleared (everyone enters/exits the elevator)
- What does the elevator do if it cannot connect to the network during initialization? 
