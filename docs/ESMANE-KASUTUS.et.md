# KoduMedia esmase kasutuse juhend

See juhend on mõeldud valmis Windowsi versiooni kasutajale. Programmi
kasutamiseks ei ole vaja PowerShelli ega programmeerimisoskust.

## Enne alustamist

Vaja on:

- Windows 10 või Windows 11 x64 arvutit, mis jääb KoduMedia serveriks;
- filme, seriaale või muusikat selle arvuti kettal või arvutile ligipääsetaval
  võrgukettal;
- tänapäevast veebibrauserit;
- teiste seadmetega kasutamiseks sama usaldatud koduvõrku;
- soovi korral tasuta TMDb, AcoustID ja OpenSubtitles.com kontosid.

Serveriarvuti peab KoduMedia kasutamise ajal töötama. Televiisorisse või telefoni
ei ole KoduMedia rakendust vaja paigaldada; veebiliides avatakse brauseris.

## 1. Laadi KoduMedia alla

1. Ava GitHubis KoduMedia leht.
2. Vali paremal **Releases**.
3. Ava uusim väljalase.
4. Laadi Assets alt `kodumedia-<versioon>-windows-x64.zip`.
5. Soovi korral laadi alla ka samanimeline `.sha256` kontrollfail.

Ära laadi kasutamiseks GitHubi automaatselt loodud **Source code** ZIP-i. See on
arendajatele ja ei sisalda valmis Windowsi programmi.

## 2. Paki ZIP täielikult lahti

1. Tee allalaaditud ZIP-il paremklõps.
2. Vali **Extract All... / Ekstrakti kõik...**.
3. Vali kaust, kuhu sul on kirjutusõigus, näiteks
   `C:\Users\SinuNimi\KoduMedia`.
4. Ava lahtipakitud kaust.

Ära käivita programmi otse ZIP-i eelvaatest. KoduMedia peab saama oma kausta
seadeid, logisid ja vahemälu kirjutada.

## 3. Käivita server

1. Tee topeltklõps failil `start-kodumedia.cmd`.
2. Kui Windows Defender Firewall küsib luba, märgi ainult
   **Private networks / Privaatvõrgud**.
3. Ava serveriarvutis brauseriga <http://localhost:8096>.

Windows võib esmakordsel käivitamisel näidata SmartScreeni hoiatust, sest
rakendus ei ole veel koodiallkirjastatud. Veendu, et fail pärineb ametlikust
`mihkelwow/kodumedia` repost ja kontrolli vajaduse korral väljalaske SHA-256
summat.

## 4. Ava „Esmane seadistus”

1. Ava KoduMedia veebiliideses **Seaded**.
2. Leia **KoduMedia töövalmidus**.
3. Ava **Esmane seadistus**.

Juhend näitab sammude kaupa, mis on juba valmis ja mis vajab veel seadistamist.

## 5. Lisa metaandmete ligipääs

### TMDb

TMDb annab filmidele ja seriaalidele pealkirjad, kirjeldused, näitlejad,
posterid ning taustapildid.

1. Loo tasuta TMDb konto.
2. Taotle konto seadetes API ligipääs.
3. Kopeeri pikk **API Read Access Token** ehk Bearer token, mitte lühike v3 API
   Key.
4. Ava KoduMedias **Seaded → Metaandmed**.
5. Kleebi token vastavasse paroolitüüpi väljale.
6. Salvesta seaded ja kasuta kontrollimise nuppu.

Token salvestatakse sinu serveriarvuti kohalikku `data` kausta. Seda ei ole
KoduMedia lähtekoodis ega GitHubi repos.

### OpenSubtitles.com — valikuline

Subtiitrite otsimiseks:

1. loo OpenSubtitles.com konto;
2. loo Consumer API võti;
3. sisesta KoduMedia seadetes API-võti, kasutajanimi ja parool;
4. vali eelistatud subtiitrikeeled.

### AcoustID — valikuline

AcoustID aitab tundmatuid muusikafaile audio fingerprint'i kaudu tuvastada.
KoduMedia tasuta Shazami-põhine tuvastus ei vaja eraldi API-võtit; AcoustID on
täiendav fallback.

## 6. Lisa meediakaustad

1. Ava **Seaded → Meediakogud ja kaustad**.
2. Vali sobiv kogu: filmid, seriaalid või muusika.
3. Lisa vastav kaust, näiteks:

   ```text
   D:\Filmid
   D:\Seriaalid
   D:\Muusika
   ```

4. Salvesta seaded.
5. Vajuta esmase seadistuse lõpus **Skänni meediakogud**.

Kõiki kogusid saab skännida ka serveriarvuti system tray KoduMedia ikooni
menüüst.

Skannimine ei kopeeri ega muuda originaalfaile. KoduMedia loeb tehnilise info
ning loob oma metaandmed ja vahemälu `data` kausta.

## 7. Kasuta KoduMediat teises seadmes

Telefon, tahvel, sülearvuti ja serveriarvuti peavad olema samas koduvõrgus.

Serveriarvuti kohaliku IPv4-aadressi leiab ilma käsureata:

1. ava Windowsis **Settings / Seaded**;
2. vali **Network & Internet**;
3. ava kasutatav Wi-Fi või Etherneti ühendus;
4. leia **IPv4 address**, näiteks `192.168.1.50`.

Teises seadmes ava:

```text
http://192.168.1.50:8096
```

Asenda näite IP enda serveriarvuti aadressiga. Aadress võib ruuteri
taaskäivitamisel muutuda; püsivaks kasutuseks saab ruuteris määrata DHCP
reservation'i.

## 8. Esita või cast'i telerisse

1. Ava film või episood.
2. Vajuta **Esita**.
3. Ava playeris castimise/seadme nupp.
4. Vali nähtav AirPlay, Google Cast või DLNA seade.

### AirPlay

- iPhone/iPad ja teler peavad olema samas võrgus.
- Teleris peab AirPlay olema lubatud.
- KoduMedia kasutab toetatud Apple'i brauseris native HLS-i.

### Samsung ja muu DLNA

- Teler ja KoduMedia server peavad olema samas võrgus.
- Teler peab lubama sama võrgu seadmete ühendusi.
- Esituse algus võib võtta 12–20 sekundit, sest server valmistab sujuvama
  esituse jaoks puhvri.

### Google Cast

Google Casti brauserisaatja vajab turvalist HTTPS-konteksti. Tavalisel kohalikul
`http://...:8096` aadressil kasuta võimaluse korral AirPlayd või DLNA-d.

## 9. Käivita KoduMedia koos Windowsiga

Klõpsa serveriarvuti system tray KoduMedia ikoonil ja luba Windowsiga koos
käivitumine. Samast menüüst saab avada veebiliidese, skännida meediakogud või
serveri sulgeda.

## 10. Uuenda KoduMediat

1. Laadi GitHub Releases lehelt alla uus täielik
   `kodumedia-<versioon>-windows-x64.zip`.
2. Ava serveriarvutis KoduMedia.
3. Vali **Seaded → Süsteemi olek → KoduMedia uuendamine**.
4. Vali allalaaditud ZIP tervikuna.

Uuendaja kontrollib manifesti ja failide SHA-256 summasid, säilitab `data`
kausta ning taastab ebaõnnestumise korral eelmise rakenduse versiooni.

## Levinud probleemid

### Teine seade ei ava KoduMediat

- kontrolli, et serveriarvuti ja teine seade on samas võrgus;
- kontrolli, et KoduMedia töötab serveriarvutis;
- proovi serveriarvutis esmalt `http://localhost:8096`;
- luba Windows Firewallis KoduMedia ainult Private networks võrgus;
- veendu, et kasutad serveriarvuti praegust IPv4-aadressi.

### Filme või muusikat ei kuvata

- kontrolli meediakausta teed;
- vajuta **Skänni meediakogud**;
- kontrolli, et serveriarvutil on kausta lugemisõigus;
- võrguketta puhul kontrolli, et ketas on ühendatud sama Windowsi kasutaja all.

### Posterid ja kirjeldused puuduvad

- kontrolli TMDb API Read Access Tokenit;
- kontrolli internetiühendust;
- käivita metaandmete uuendamine või uus skannimine.

### Teler ühendub, aga pilt ei alga

- oota DLNA puhul vähemalt 20 sekundit;
- veendu, et teler ja server on samas võrgus;
- luba teleris uue seadme ühendus;
- proovi teist faili, et eristada seadme- ja codec'i ühilduvust;
- kontrolli KoduMedia logi cast-plaani ja vea kohta.

### Video puhverdamine jätkub

- eelista serveriarvutil Etherneti ühendust;
- kasuta teleris tugevat 5 GHz Wi-Fi ühendust;
- kontrolli, kas serveri protsessor või GPU jõuab transkodeerida;
- proovi madalamat kvaliteediprofiili;
- kontrolli, kas originaalfail sobib telerile Direct Play jaoks.

## Privaatsus ja turvalisus

- Ära ava porti `8096` internetti.
- KoduMedial pole veel kasutajakontosid ega TLS-i.
- API-võtmed ja seaded asuvad ainult serveriarvuti `data` kaustas.
- Ära jaga logisid enne isiklike failiteede ja võimalike võtmete eemaldamist.
- Varunda `data` kaust, kui soovid säilitada seadistuse ja vaatamisajaloo.
