# Turvalisus

## Praegune ohumudel

KoduMedia 0.6.4 eeldab **usaldatud koduvõrku**. Serveril puuduvad kasutajakontod, autentimine ja TLS.

Kõik, kes pääsevad Windowsi serveri porti 8096, saavad:

- kataloogi, metadata välju ja failinimesid vaadata;
- meediat esitada;
- kogusid ning allikakaustu lisada või eemaldada;
- skannimist käivitada;
- kohandatud kogu kustutada;
- fallback-failisirvijas serveri loetavaid kettatähti ja kaustanimesid vaadata;
- muuta FFmpeg ja FFprobe käivitusfaili teed;
- muuta TMDb tokenit ning IMDb URL-i või kohaliku faili teed;
- käivitada serveripoolse IMDb download'i;
- käivitada metadata hooldustöid.

Settings-, filesystem- ja metadata-maintenance endpoint'id on haldusfunktsioonid, kuigi 0.6.4-l pole veel admin-rolli. Rakenduse update endpoint on erandina piiratud ainult serveriarvuti loopback-brauserile.

## Ära tee praegu

- ära ava ruuteri porti 8096 internetti;
- ära kasuta autentimata avalikku tunnelit;
- ära pane serverit autentimata avaliku reverse proxy taha;
- ära käivita `kodumedia.exe` administraatori õigustes;
- ära käivita seda kasutajana, kellel on ligipääs saladustele või kaustadele, mida LAN-i kasutajad ei tohi näha;
- ära anna KoduMedia protsessile meediakaustades kirjutusõigust, kui seda pole vaja;
- ära vali FFmpeg/FFprobe teeks tundmatut või kontrollimata `.exe` faili;
- ära seadista IMDb allikaks usaldamata URL-i.

Brauseris salvestatud executable path käivitatakse serveris scan/playback/status operatsioonide ajal. IMDb remote source paneb serveri tegema välise HTTPS päringu. Seetõttu peab server olema ainult usaldatud LAN-is ja protsess töötama piiratud Windowsi kasutajana.

## Soovitatud Windowsi LAN-paigaldus

- kasuta eraldi tavalist Windowsi kasutajat või tulevikus piiratud teenusekontot;
- anna meediakaustadele NTFS-i või võrguketta õigustes ainult lugemisõigus;
- anna `data` kaustale kirjutusõigus ainult KoduMedia kasutajale;
- piira teiste lokaalsete kasutajate ligipääs failile `data\settings.json`;
- hoia `data\catalog.json`, `data\settings.json` ja vajadusel metadata cache'i regulaarset varukoopiat;
- luba Windows Defender Firewallis TCP 8096 ainult **Private network** profiilile;
- võimalusel piira tulemüürireegel oma kohaliku alamvõrguga;
- kasuta FFmpeg build'i, mille päritolu ja checksum on kontrollitud;
- ära lisa `.env`, `settings.json` ega päris meediafailide teid Git'i või tugiraportisse.

## TMDb token

- token salvestatakse lokaalselt `data\settings.json` faili;
- API ei tagasta tokenit tagasi;
- brauser saab ainult `tmdbTokenConfigured` ja maskitud viimased märgid;
- token liigub LAN-is HTTP kaudu ainult salvestamise hetkel, sest 0.6.4 ei paku TLS-i;
- tokenit ei kirjutata metadata job'i olekusse ega tavapärasesse logisse;
- süsteem ei sobi avalikku ega usaldamata võrku.

## Native Windowsi file picker

`POST /api/v1/filesystem/pick`:

- aktsepteerib ainult loopback-aadressilt tulnud HTTP ühendust;
- avab dialoogi serveriarvuti interaktiivses Windowsi sessioonis;
- kasutab kaustavalikul `FOS_PICKFOLDERS` režiimi;
- tagastab valitud absoluutse tee, mitte faili sisu;
- kasutaja cancel-toiming tagastab 204;
- valideerib failirežiimis lubatud laiendi.

Loopback-piirang tähendab, et teine LAN-seade ei saa serveriarvuti ekraanil native dialoogi avada. LAN-kliendile jääb fallback-browser.

Windows Service'i mitteinteraktiivses sessioonis ei pruugi native dialoog olla kasutatav. Sellisel juhul peab klient saama 503 ja kasutama fallback'i; service-mode kasutus ei tohi eeldada desktop-interaktsiooni.

## Kohaliku võrgu castimine

DLNA/UPnP seadmeotsing ja juhtimine on teadlikult piiratud koduvõrguga:

- SSDP vastusest võetakse vastu ainult HTTP(S) description URL, mis lahendub
  private-, loopback- või link-local IP-aadressile;
- description redirect ei tohi väljuda kohalikku võrku ning control URL peab
  jääma sama seadme hosti alla;
- XML- ja SOAP-vastused on suuruspiiriga ning telerile saadetakse ainult
  serveri loodud `/api/v1/` playback URL;
- cast API on nagu ülejäänud playback API autentimata usaldatud-LAN funktsioon.

Samas võrgus olev kasutaja saab seetõttu käivitada, juhtida või lõpetada
castimist leitud teleris. Porti 8096 ei tohi avalikku võrku avada. Google Casti
veebisaatja laaditakse ainult turvalises HTTPS-kontekstis; AirPlay seadmevalikut
haldab brauseri native API kasutaja otsese žesti järel.

## Fallback-failisirvija piirid

Fallback-browser:

- näitab serveriprotsessile nähtavaid kettatähti;
- kontrollib, et sisend oleks absoluutne olemasolev kaust;
- ei paku faili sisu allalaadimise endpoint'i;
- peidab `$Recycle.Bin` ja `System Volume Information` nimed;
- piirab ühe vastuse kirjete arvu;
- võib siiski paljastada muude loetavate kaustade ja failide nimed.

0.6.4-l puudub lubatud source-root'ide allowlist. See on enne mittekoduse või mittekasutaja-admin mudeli lisamist kohustuslik töö.


## Rakenduse uuendaja

`GET/POST /api/v1/system/update` on teadlikult rangem kui teised 0.6.4 haldusendpoint'id:

- TCP ühenduse lähteaadress peab olema loopback;
- HTTP `Host` peab olema `localhost`, `127.0.0.1` või loopback IPv6;
- `Origin` päise olemasolul peab see olema sama loopback-port;
- installipäring nõuab custom-headerit `X-KoduMedia-Update: install`;
- request body ja ZIP payload on suuruspiiriga;
- korraga saab olla ainult üks install.

Pakett peab sisaldama `release-manifest.json` faili. Kontrollitakse:

- manifesti skeemi, tootenime ja `windows-amd64` platvormi;
- et sihtversioon on kehtiv SemVer ja praegusest rangelt uuem;
- iga faili täpset suurust ning lowercase SHA-256 väärtust;
- et paketis ei ole manifestis kirjeldamata ega dubleeritud faile;
- et ZIP ei sisalda absoluutseid teid, `..` traversal'it, backslash-teid, alternate data stream'i kooloneid ega sümbollinke;
- et olemas on `kodumedia.exe`, `kodumedia-updater.exe` ja `VERSION`;
- et `data`, `.env`, `settings.json` ja `catalog.json` puuduvad payload'ist.

Paigaldaja varundab ainult manifestis kirjeldatud rakendusfailid. Uus server peab 60 sekundi jooksul tagastama health-endpoint'ilt täpse sihtversiooni. Vastasel juhul tapetakse uus protsess, taastatakse backup ning käivitatakse eelmine server. Update logid ja olek asuvad `data\updates` all.

**Piirang:** SHA-256 manifest tuvastab juhusliku või tahtliku paketimuudatuse ainult siis, kui manifest ise pärineb usaldatud kanalist. 0.6.4 ei kontrolli veel väljalaskja digitaalallkirja. Kasuta ainult usaldatud KoduMedia ZIP-i ja kontrolli eraldi avaldatud kontrollsummat. Kohalik pahatahtlik kasutaja, kellel on KoduMedia installi- või `data` kausta kirjutusõigus, jääb sellest ohumudelist väljapoole.


## Quality Engine'i cache ja hardware encoder'id

- JIT-cache asub serveri hallatavas `data\optimized-media` kaustas ning võib sisaldada algsest meediast tuletatud vaadatavat video- ja helisisu; `data` kausta tuleb kaitsta sama rangelt kui meediakogu ennast.
- Cache'i failiteed ei tulene kliendi vabast tekstist. Võti arvutatakse lähtefaili signatuurist, kvaliteediprofiilist ja valideeritud playback-parameetritest.
- Cache'i manifest ei anna õigust kirjutada lähtefaili kõrvale; väljund jääb `data` juure alla.
- LRU-koristus ei kustuta aktiivse encoder-session'i cache'i.
- VMAF analüüs kasutab serveri enda ajutist playlist-snapshot'i ega anna kliendile võimalust valida suvalist võrdlusfaili.
- QSV, NVENC ja AMF encoder'i olemasolu kontrollitakse päris ühekaadrilise käivitusega. Valitud välise FFmpegi executable'i usalduspiir jääb samaks: kontrollimata `.exe` võib töötada KoduMedia protsessi õigustes.
- Kvaliteediprofiili või cache'i mahu muutmine on autentimata LAN-is endiselt haldustoiming; pahatahtlik LAN-kasutaja võib põhjustada CPU/GPU- ja kettakoormust.

## Bundled FFmpeg ja meediatöötlus

- kaasatud FFmpeg/FFprobe binaarid on release manifestis SHA-256 ja suurusega kaetud;
- GPLv3 litsents, build-info ja upstream source commit on runtime-pakis kaasas;
- custom executable path jääb teadlikuks admin-seadeks ning võimaldab käivitada valitud binaari KoduMedia protsessi õigustes;
- meedia- ja subtiitrifailid on ebausaldusväärne sisend FFmpegile, seega serverit ei tohi käivitada administraatorina ning meediakaustadele tuleks anda ainult vajalik lugemisõigus;
- transkodeerimisel kustutatakse metadata, ignoreeritakse data-stream'e ning väljund kirjutatakse ainult serveri hallatavasse `data\transcodes` kausta;
- OpenSubtitles download URL peab olema HTTPS, response on suuruspiiriga ja failinimi puhastatakse enne salvestamist;
- OpenSubtitles ning TMDb/IMDb on välised teenused: otsinguandmed ja identifikaatorid väljuvad koduvõrgust ainult vastava funktsiooni kasutamisel.

DRM-kaitse eemaldamine ei ole toetatud. Playback pipeline ei tohi proovida Widevine/FairPlay/PlayReady kaitset mööda minna.

## IMDb remote dataset

- vaikeallikas on `https://datasets.imdbws.com/title.ratings.tsv.gz`;
- remote source peab olema absoluutne HTTPS URL;
- allalaadimine on piiratud 512 MiB-ga;
- response voogedastatakse kettale, mitte ei loeta tervikuna mällu;
- ajutine fail valideeritakse enne aktiivse cache'i asendamist;
- vigane või katkine download ei asenda viimast töötavat cache'i;
- HTTP 304 korral valideeritakse ka olemasolev kohalik fail;
- cache metadata ei sisalda TMDb tokenit;
- suur fail parsitakse serveris, mitte brauseris.

Praeguses usaldatud-LAN versioonis saab haldusõiguseta LAN-kasutaja siiski muuta IMDb HTTPS URL-i. Enne kasutajakontode lisamist tuleb see endpoint piirata admin-rollile ja soovitatavalt rakendada remote source allowlist.

## Võrgukettad ja UNC

KoduMedia näeb ainult draive ja UNC asukohti, millele protsessi Windowsi kasutajal on õigused. Interaktiivse kasutaja mapped drive ei pruugi olla nähtav teise kasutaja või Windows Service'i sessioonis. Anna õigused konkreetsele KoduMedia kontole ja eelista vähimate õiguste mudelit.

## Meediafailid

Skanner:

- loeb kataloogistruktuuri;
- loeb failide metadata FFprobe kaudu;
- ei kirjuta lähtefailidesse;
- ei kustuta lähtefaile;
- hoiab absoluutset failiteed serveri kataloogi eraldi kaardis ega väljasta seda API JSON-is.

API kuvab siiski `relativePath` ja algse failinime, mis võivad sisaldada isiklikku infot. Usaldamata kasutajale ei tohi 0.6.4 LAN-i ligipääsu anda.

## Olemasolevad tehnilised kaitsed

- request body suurus on piiratud;
- JSON decoder ei aktsepteeri tundmatuid välju;
- veebivastustel on `nosniff`, same-origin referrer ja frame-kaitse;
- CORS on vaikimisi suletud eraldi origin'itele;
- UPnP description ja control URL-id on piiratud kohaliku võrgu ning sama seadme hostiga;
- native picker on loopback-only;
- update install on loopback-only ning nõuab loopback Host/Origin kontrolli ja intent-headerit;
- update ZIP läbib manifesti, SemVer, path, suuruse ja SHA-256 kontrolli enne serveri sulgemist;
- update payload ei saa muuta `data` kausta ega `.env` faili;
- updater teeb rakendusfailidest backup'i ning kontrollib uut versiooni health-endpoint'iga enne õnnestumise märkimist;
- HLS sessiooni failitee kontroll väldib directory traversal'it;
- sünteetilist seriaali parent-kirjet ei saa streamida ega transkodeerida;
- source path peab olemas olema ja olema kataloog;
- runtime-seadete keel ja kohustuslikud teed valideeritakse teenusekihis;
- component checks kasutavad timeout'e;
- IMDb remote allikas peab kasutama HTTPS-i ja download'il on suuruspiir;
- cache ja JSON püsifailid kirjutatakse esmalt ajutisse faili;
- vana `sources: null` ja settings schema v1 parandatakse kontrollitult;
- salvestatud TMDb tokenit ei tagastata API vastuses;
- metadata hooldustööd serialiseeritakse, et vältida tahtmatut paralleelset ressursikoormust.

## Teadlikud puudused

- puudub autentimine ja autoriseerimine;
- puudub TLS;
- puudub CSRF kaitse muutvatele endpoint'idele;
- puudub rate limiting;
- puudub auditlogi;
- puudub source-root allowlist;
- puudub IMDb remote host allowlist;
- puudub executable path allowlist või signed-binary kontroll;
- protsessisisene job manager ei säilita restarti;
- native picker sõltub interaktiivsest Windowsi desktop-sessioonist;
- JSON kataloog ei ole transactional andmebaas.

## Enne kaugjuurdepääsu

Vajalikud tööd:

1. autentimine ja turvalised HttpOnly/SameSite cookie'd;
2. CSRF kaitse kõigile muutvatele päringutele;
3. TLS;
4. rate limiting;
5. admin/viewer rollid ja library-level ACL;
6. session revocation;
7. auditlogi;
8. source path'i lubatud juurte nimekiri;
9. settings/filesystem/maintenance endpoint'ide admin-only policy;
10. remote metadata source allowlist ja redirect-policy;
11. sõltuvuste regulaarne skannimine;
12. Windows Service'i õiguste ja uuendusprotsessi threat model;
13. eraldi turvaülevaatus.

## Turvavea raporteerimine

Ära lisa avalikku raportisse:

- TMDb tokenit;
- `.env` või `settings.json` sisu;
- päris koduvõrgu aadresse;
- Windowsi kasutajanimesid;
- meediafailide täielikke teid.

Kaasa minimaalne reprodutseerimisjuht, mõjutatud versioon ja see, kas probleem nõuab LAN-i ligipääsu või lokaalset Windowsi kontot.

## System tray ja Windowsi startup

Tray menüü töötab ainult serveriarvuti lokaalses Windowsi protsessis ega lisa LAN API-le uusi haldusendpoint'e. Sisselogimisel käivitamine kirjutab ainult praeguse kasutaja `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` võtmesse ning ei vaja administraatoriõigusi. Väärtus sisaldab parajasti töötava `kodumedia.exe` absoluutset teed. Olekukontroll arvestab ka sama kasutaja `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run` KoduMedia väärtust, et Windowsi Startup apps või Task Manageri kaudu keelatud kirjet mitte ekslikult lubatuna näidata. Rakenduses valiku sisselülitamine eemaldab ainult KoduMedia vana keelatud kinnituse; teiste startup-rakenduste väärtusi ei loeta ega muudeta.

KoduMedia hoiab üht kasutajaseansi-põhist named mutex'it, et juhuslik duplikaatkäivitus ei looks teist HTTP serverit. Teise käivituse korral avatakse olemasolev localhosti klient.
