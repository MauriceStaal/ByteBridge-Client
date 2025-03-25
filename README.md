# ByteBridge-Client
De ByteBridge-Client is ontworpen om bestanden te synchroniseren tussen een clientmachine en de ByteBridge-server.

## Beperkingen

### Server:
1. **Dubbele Uploads met Verschillende Bestandsnamen**: Als hetzelfde bestand twee keer wordt geüpload met verschillende bestandsnamen, beschouwt de server dit als een update en overschrijft het eerste record in de database. Dit resulteert in twee kopieën van het bestand op zowel de client als de server, maar slechts één record in de database.
2. **Bestand Verwijdering bij Update**: Bij het uitvoeren van een update wordt het oude bestand op de server niet automatisch verwijderd.
3. **Mapbeheer**: Het aanmaken of verwijderen van mappen wordt nog niet ondersteund.

### Client:
1. **Geen Inhoudswijziging Detectie**: Wijzigingen in de inhoud van een bestand triggeren momenteel geen automatische update.
2. **WebSocket-ondersteuning Ontbreekt**: De client luistert momenteel niet naar WebSocket-gebeurtenissen voor real-time updates.


## TODO

- Integreren met de socket van de server.
- Bestanden verwijderen als ze niet meer op de server staan.
- Bestanden check bij opstarten client
- Subfolders
- --dev of --debug flag toevoegen voor logging in de terminal
- 204 status fixen
  
### Optioneel

- User laten selecteren welke folder te syncen i.p.v hardcoded de folder erin zetten.
