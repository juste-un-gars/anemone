# Debug OnlyOffice - Suivi des tests

**Date :** 2026-02-14 → 2026-02-15
**Serveur de test :** FR2 (192.168.83.37)
**Version OO :** 9.2.1 (Docker `onlyoffice/documentserver`)
**Anemone :** v0.20.0-beta
**Statut :** RÉSOLU — Tous les problèmes corrigés

---

## Probleme initial (RÉSOLU)

~~Le container OnlyOffice ne telecharge JAMAIS le fichier a editer.~~
→ **RÉSOLU** : Le download serveur-a-serveur fonctionne depuis Test 6.

## Probleme 2 (RÉSOLU)

~~L'editeur affiche "Echec du telechargement" car l'URL `Editor.bin` pointe vers `http://localhost:9980/cache/...`~~
→ **RÉSOLU** : Architecture HTTPS direct port (session 21) + HTTP server sur docker0 bridge IP (session 22).

### Symptomes actuels
- Download serveur-a-serveur : **OK** (`OO download: serving file`)
- JWT : **OK** (`checkJwt success`)
- Callbacks : **OK** (status 1/4)
- `getBaseUrlByConnection host=localhost:9980` → **MAUVAIS** (devrait etre `192.168.83.37:8443`)
- Editor.bin URL = `http://localhost:9980/cache/...` → **INACCESSIBLE** au navigateur

---

## Architecture du flux

```
Navigateur
  |
  |--> GET /files/edit?share=X&path=Y  -->  Anemone (HTTPS :8443)
  |      Genere editor config JSON signe JWT contenant :
  |        document.url = http://host.docker.internal:8080/api/oo/download?token=JWT
  |        callbackUrl  = http://host.docker.internal:8080/api/oo/callback
  |        token = JWT signe avec oo_secret
  |
  |--> Charge /onlyoffice/web-apps/api.js  -->  Proxy inverse --> Container OO (:9980)
  |
  |--> DocsAPI.DocEditor(config) --> envoie config au container OO
          |
          |--> GET document.url  -->  Anemone (:8080 HTTP)   <-- JAMAIS EXECUTE
          |
          |--> POST callbackUrl -->  Anemone (:8080 HTTP)   <-- FONCTIONNE
```

---

## Etat actuel du container OO

### local.json
```json
{
  "services": {
    "CoAuthoring": {
      "secret": {
        "browser": { "string": "bbef0f4ea0c1269070e10b18d09f4dfd9fb39d9706cc5bb8ca6ac8d8c295b93b" },
        "inbox":   { "string": "bbef0f4ea0c1269070e10b18d09f4dfd9fb39d9706cc5bb8ca6ac8d8c295b93b" },
        "outbox":  { "string": "bbef0f4ea0c1269070e10b18d09f4dfd9fb39d9706cc5bb8ca6ac8d8c295b93b" },
        "session": { "string": "bbef0f4ea0c1269070e10b18d09f4dfd9fb39d9706cc5bb8ca6ac8d8c295b93b" }
      },
      "token": {
        "enable": {
          "browser": true,
          "request": { "inbox": true, "outbox": true }
        }
      },
      "request-filtering-agent": {
        "allowPrivateIPAddress": true,
        "allowMetaIPAddress": true
      },
      "requestDefaults": {
        "rejectUnauthorized": false
      }
    }
  }
}
```

### Etat DB Anemone (system_config)
```
oo_secret  = bbef0f4ea0c1269070e10b18d09f4dfd9fb39d9706cc5bb8ca6ac8d8c295b93b
oo_enabled = true
oo_url     = http://localhost:9980
```

---

## Tests effectues

### Test 1 : Fix URL HTTP (au lieu de HTTPS)
- **Changement :** download/callback URL = `http://host.docker.internal:8080` (avant : `https://<browser-host>:8443`)
- **Changement :** Serveur HTTP demarre auto quand OO active
- **Fichiers modifies :** `handlers_onlyoffice_api.go` (L294-296), `main.go` (L181)
- **Resultat :** ECHEC
- **Log OO :** `Error: DNS lookup 172.17.0.1(host:host.docker.internal) is not allowed. Because, It is private IP address.`
- **Conclusion :** Protection anti-SSRF bloque les IP privees

### Test 2 : Fix SSRF + JWT desactive
- **Changement :** `allowPrivateIPAddress: true`, `allowMetaIPAddress: true` dans local.json
- **Changement :** JWT token.enable.browser/inbox/outbox = false
- **Resultat :** ECHEC
- **Log OO :** Aucune nouvelle erreur SSRF (fix marche). Mais toujours pas de download.
- **Nouveau symptome :** Fenetre "Impossible d'enregistrer le document" (callbacks sans JWT rejetes)
- **Conclusion :** SSRF fixe mais pas suffisant

### Test 3 : Secret JWT synchronise
- **Changement :** Inject `oo_secret = bbef...b93b` dans DB Anemone
- **Changement :** JWT re-active dans container
- **Resultat :** ECHEC
- **Verification :** secret_len=64 confirme dans les logs Anemone
- **Log Anemone :** URLs generees correctement, callbacks recus, ZERO download
- **Log OO :** docservice ne log RIEN apres son demarrage (fige a 16:06 UTC)

### Test 4 : Debug log URL
- **Changement :** Ajout log `OO editor: URLs generated` avec downloadURL, callbackURL, secret_len
- **Resultat :** ECHEC
- **Log Anemone :**
  ```
  downloadURL = http://host.docker.internal:8080/api/oo/download?token=eyJhbGciOiJIUzI1NiIs...
  callbackURL = http://host.docker.internal:8080/api/oo/callback
  secret_len  = 64
  ```
- **Conclusion :** URLs et secret corrects cote Anemone

---

## Tests de connectivite

| Test | Commande | Resultat |
|------|----------|----------|
| Container --> Host HTTP | `docker exec onlyoffice-docs curl http://host.docker.internal:8080/` | 303 (OK) |
| Host --> Container | proxy `/onlyoffice/*` --> `http://localhost:9980` | OK (api.js charge) |
| Container --> Host callback | POST `/api/oo/callback` | OK (status 1/4 recus) |
| Container --> Host download | GET `/api/oo/download?token=...` | **FONCTIONNE** (depuis Test 6) |

---

## Anomalies RESOLUES

1. ~~Le docservice ne log plus rien~~ → Le log level etait WARN. En passant a DEBUG, on voit tout le flow.
2. ~~Les callbacks arrivent mais pas le download~~ → Le download serveur-a-serveur FONCTIONNE depuis le container recreé (Test 6). Le problème n'était pas le download.
3. **Status 1 puis 4** → Status 1 = "editing started", status 4 = "closed without changes". Le download serveur fonctionne mais le navigateur ne peut pas charger l'Editor.bin (voir anomalie active ci-dessous).

## Anomalie ACTIVE : Editor.bin URL incorrecte

Le serveur OO telecharge le fichier, le convertit en `Editor.bin`, puis renvoie au navigateur une URL cache :
```
Response command: {"type":"open","status":"ok","data":{"Editor.bin":"http://localhost:9980/cache/files/data/.../Editor.bin?md5=...&expires=..."}}
```
Le navigateur essaie de charger `http://localhost:9980/cache/...` → **ECHEC** car `localhost:9980` pointe vers la machine du CLIENT, pas vers FR2.

**Cause racine :** Le reverse proxy Anemone envoie `Host: localhost:9980` au container OO. OO utilise ce Host pour generer toutes ses URLs internes (cache, websocket).

**Log OO prouvant le probleme :**
```
getBaseUrlByConnection host=localhost:9980 x-forwarded-host=localhost:9980 x-forwarded-proto=http x-forwarded-prefix=undefined
```

**Fix tente :** Ajout `X-Forwarded-Host`, `X-Forwarded-Proto`, `X-Forwarded-Prefix` dans le Director du proxy.
**Resultat :** Les headers ne semblent PAS atteindre OO (toujours `x-forwarded-host=localhost:9980`).
**Hypothese :** Le `logger.Debug()` dans le proxy n'apparait pas dans les logs Anemone (log level = WARN), donc on ne peut pas confirmer que le code s'execute. Les connexions WebSocket pourraient aussi ne pas passer par le Director.

---

## Pistes RESOLUES

### Piste A : Log level OO trop bas — CONFIRME
Le docservice loggait au niveau WARN. En passant a DEBUG (modifier `/etc/onlyoffice/documentserver/log4js/production.json`), on voit tout le flow JWT, download, commands.
- **Commande :** `docker exec onlyoffice-docs python3 -c "import json; ... cfg['categories']['default']['level']='DEBUG' ..."`
- **Puis :** `docker exec onlyoffice-docs supervisorctl restart ds:docservice ds:converter`

### Piste B : Converter logs — VIDE
Le converter ne log que des "worker started". Le download initial est fait par le converter (pas le docservice) mais les erreurs n'apparaissent pas dans ses logs.

### Piste C : JWT structure — VALIDE
Les logs DEBUG OO montrent `checkJwt success` avec le payload complet decode. La structure JWT est correcte :
- Contient `document.url`, `document.key`, `document.fileType`, `document.title`, `document.permissions`
- Contient `documentType`, `editorConfig.callbackUrl`, `editorConfig.user`, `editorConfig.lang`, `editorConfig.mode`
- Signe avec HS256, secret correct (verifie par OO)

### Piste D : Example app — NON UTILISABLE
L'example app redirige vers `http://localhost:9980/example/editor?...` (meme probleme de Host). Inutile pour le debug.

### Piste E : Container fresh — FAIT (Test 6)
Container supprime et recree proprement. Le download serveur-a-serveur fonctionne avec un container frais (JWT_SECRET en env var suffit, pas besoin de patcher allowPrivateIPAddress manuellement).

### Piste F : IP directe — NON NECESSAIRE
Le download via `host.docker.internal` fonctionne. Pas besoin de passer a l'IP directe.

### Piste E : Supprimer le container et recreer proprement
Le container a ete modifie plusieurs fois. Un fresh start pourrait resoudre des problemes de config persistants.
- **Action :** `docker rm -f onlyoffice-docs` puis relancer depuis l'admin Anemone
- **Important :** Integrer les fixes (SSRF, secret) dans le code Anemone AVANT de recreer

### Piste F : Utiliser l'IP directe au lieu de host.docker.internal
`host.docker.internal` est une convention Docker qui n'est pas toujours supportee.
- **Action :** Utiliser `http://172.17.0.1:8080` (IP du bridge Docker) directement
- **Verification :** `docker exec onlyoffice-docs cat /etc/hosts | grep host.docker`

### Test 5 : Connectivite IP directe + Example app + Converter logs
- **Piste D :** Example app demarree (`supervisorctl start ds:example`)
- **Piste F :** `curl http://172.17.0.1:8080/api/oo/download?token=test` → **403 Forbidden** (attendu, token bidon mais endpoint atteignable)
- **Piste B :** Converter logs = rien (que des worker starts, aucune erreur)
- **Conclusion :** Connectivite OK des deux cotes. Le container PEUT atteindre Anemone. Le probleme n'est PAS reseau.

**A tester :** Example app via `https://192.168.83.37:8443/onlyoffice/example/`
Si example app fonctionne → probleme 100% dans le JWT/config d'Anemone (Piste C)

---

## Test 6 : Logs DEBUG + Fix reverse proxy headers

**Date :** 2026-02-14 17:26+

### Découverte majeure (logs DEBUG)

Les logs `docservice/out.log` en mode DEBUG révèlent :

1. **JWT vérifié avec succès** : `checkJwt success: decoded = {document.url, editorConfig...}` → la structure JWT est correcte
2. **Download fichier OK** : `Start command: {c:"open", url:"http://host.docker.internal:8080/api/oo/download?token=..."}` → `Response command: {type:"open", status:"ok", data:{Editor.bin:"http://localhost:9980/cache/..."}}`
3. **BUG** : L'URL du cache `Editor.bin` retournée au navigateur pointe vers `http://localhost:9980/cache/...` → le navigateur essaie `localhost:9980` sur SA propre machine
4. **Cause** : Le reverse proxy Anemone fait `req.Host = target.Host` (localhost:9980). OO voit `host=localhost:9980` et génère toutes ses URLs avec ce host.
5. **Log preuve** : `getBaseUrlByConnection host=localhost:9980 x-forwarded-host=localhost:9980 x-forwarded-proto=http x-forwarded-prefix=undefined`

### Fix appliqué

Fichier : `internal/web/handlers_admin_onlyoffice.go` (proxy Director)

Ajout des headers de forwarding :
```go
req.Header.Set("X-Forwarded-Host", origHost)    // host original du navigateur
req.Header.Set("X-Forwarded-Proto", origScheme)  // https
req.Header.Set("X-Forwarded-Prefix", "/onlyoffice")  // préfixe de chemin
```

OO devrait maintenant générer des URLs comme : `https://192.168.83.37:8443/onlyoffice/cache/...` au lieu de `http://localhost:9980/cache/...`

### Résultat : PARTIEL

**Progres majeur :** Le download serveur-a-serveur FONCTIONNE maintenant !
```
OO download: serving file path=/srv/anemone/shares/marc/backup/test.md user=2
```
C'est la PREMIERE fois que cette ligne apparait dans les logs.

**Mais :** L'erreur "Echec du telechargement" persiste cote navigateur.
- Le `open` command retourne `"data":{}` (pas d'Editor.bin URL) → le cache etait vide apres purge, le converter n'avait pas encore fini la conversion.
- Les premiers essais avaient `connection reset by peer` sur api.js (container pas pret).
- Erreur JS OO : `changesError: TypeError: can't access property "asc_selectSearchingResults", this.api is undefined` (isDocumentLoadComplete: false)

**Headers X-Forwarded :** Non confirmes. Le `logger.Debug()` dans le proxy est filtre par le log level WARN d'Anemone. Doit etre change en `logger.Info()` ou le log level doit etre temporairement abaisse.

**Container frais :** Le `JWT_SECRET` en env var suffit pour que le download fonctionne. Pas besoin de patcher manuellement `allowPrivateIPAddress` (OO 9.2 semble l'autoriser par defaut quand JWT est configure).

### Actions implementees (Test 7)

1. **`logger.Debug` → `logger.Info`** — FAIT. Les logs proxy seront visibles dans journalctl.
2. **`req.Host = incomingHost`** — FAIT. Au lieu de `target.Host` (localhost:9980), le proxy envoie le host du navigateur (192.168.83.37:8443). OO devrait generer les URLs Editor.bin avec ce host.
3. **X-Forwarded-Host/Proto/Prefix** — FAIT. Headers de forwarding envoyes pour completude.
4. **`PatchContainerConfig()`** — FAIT. Remplace `PatchTLSConfig()`. Applique automatiquement : TLS (rejectUnauthorized:false), SSRF (allowPrivateIPAddress:true, allowMetaIPAddress:true), log level DEBUG. Chaque container cree via l'admin est fonctionnel out-of-the-box.

### Resultat Test 7 : ECHEC — Host header NON transmis

**Nouveau binaire deploye et redemarré a 17:55:29. OO DEBUG actif.**

**Logs Anemone (17:56):**
```
OO editor: URLs generated downloadURL=http://host.docker.internal:8080/api/oo/download?token=... callbackURL=http://host.docker.internal:8080/api/oo/callback secret_len=64
OO callback received key=... status=1
OO callback received key=... status=4
```
**Pas de message "OO proxy"** dans les logs malgré le `logger.Info("OO proxy", ...)` dans le code.
Un `curl -sk https://localhost:8443/onlyoffice/web-apps/apps/api/documents/api.js` retourne 200, mais aucun log proxy.

**Logs OO DEBUG (16:56 UTC = 17:56 CET):**
```
getBaseUrlByConnection host=localhost:9980 x-forwarded-host=localhost:9980 x-forwarded-proto=http x-forwarded-prefix=undefined
Response command: {"type":"open","status":"ok","data":{"Editor.bin":"http://localhost:9980/cache/files/data/.../Editor.bin?..."}}
```
Le Host header est TOUJOURS `localhost:9980`. Les X-Forwarded-* headers ne passent PAS.

---

## Test 8 : Analyse nginx interne OO

### Decouverte : Nginx interne du container OO

Le container OO a un nginx interne (port 80) qui proxie vers le docservice (port 8000).

**`/etc/nginx/includes/http-common.conf` (extrait) :**
```nginx
proxy_set_header Host $http_host;
proxy_set_header X-Forwarded-Host $the_host;
proxy_set_header X-Forwarded-Proto $the_scheme;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
# PAS de proxy_set_header X-Forwarded-Prefix !
```

Nginx utilise `$http_host` (le Host header entrant). Il forward `X-Forwarded-Host` et `X-Forwarded-Proto` mais **PAS `X-Forwarded-Prefix`**.

### Maps nginx :
```nginx
map $http_host $this_host {
    "" $host;
    default $http_host;
}
map $http_x_forwarded_host $the_host {
    ~^([^,]+)  $1;
    default $http_x_forwarded_host;
    "" $this_host;
}
map $http_x_forwarded_prefix $the_prefix {
    default $http_x_forwarded_prefix;
}
```
`$the_prefix` est utilise dans les redirections nginx (`return 302 $the_scheme://$the_host$the_prefix/...`) mais PAS forwarde au docservice.

### Probleme : Pourquoi host=localhost:9980 ?

Si notre proxy envoie `Host: 192.168.83.37:8443`, nginx devrait forwarder `Host: 192.168.83.37:8443` au docservice. Mais OO voit `host=localhost:9980`.

**Hypothese A :** Le proxy Go ne transmet pas correctement le Host header (malgre `req.Host = incomingHost`)
**Hypothese B :** La connexion Socket.IO n'est PAS initiee par le navigateur via notre proxy — elle contourne notre proxy

Le fait qu'aucun log "OO proxy" n'apparaisse pour les requetes Socket.IO est suspect. Peut-etre que:
- `httputil.ReverseProxy` ne gere pas correctement les WebSocket upgrades
- Le navigateur se connecte directement a une URL incorrecte

### Bug PatchContainerConfig()

Le check `grep -q "rejectUnauthorized"` retourne vrai (le container a deja ce champ de l'env var JWT_SECRET) et skip le SSRF + debug patch. **Fix necessaire** : checker `allowPrivateIPAddress` au lieu de `rejectUnauthorized`.

---

## Test 9 : Tentative baseurl dans local.json

**Date :** 2026-02-14 18:00

### Action
Ajout `services.CoAuthoring.utils.baseurl = "https://192.168.83.37:8443/onlyoffice/"` dans `local.json` du container, puis restart docservice.

### Resultat : ECHEC
Les logs DEBUG OO montrent toujours `host=localhost:9980` et `Editor.bin=http://localhost:9980/cache/...`. La cle `utils.baseurl` n'est soit pas le bon nom, soit pas supportee par OO 9.2.

La config par defaut `default.json` montre que `utils` contient des cles comme `utils_common_fontdir`, `limits_image_types_upload` — rien lie aux URLs.

### Recherche OO source
- Le docservice est un binaire ELF compile (pas du Node.js lisible)
- `strings docservice | grep baseUrl` → aucun resultat
- La doc OO (helpcenter) ne mentionne PAS de config `baseurl` ou `siteUrl`
- **GitHub issue #810** : La solution Traefik consiste a :
  1. Forcer `X-Forwarded-Proto: https`
  2. Mettre le subpath dans `X-Forwarded-Host` (ex: `myhost/oods`)
  3. Retirer `X-Forwarded-Prefix` pour eviter le double-prefixe

### Decouverte api.js
- `getBasePath()` extrait l'URL de base depuis le tag `<script src>` qui charge api.js
- Si api.js est charge depuis `https://192.168.83.37:8443/onlyoffice/web-apps/apps/api/documents/api.js`, le base path = `https://192.168.83.37:8443/onlyoffice/web-apps/apps/`
- L'iframe de l'editeur est charge depuis ce base path
- La connexion Socket.IO est initiee depuis l'iframe

---

## RÉSOLUTION : HTTPS direct port (2026-02-14 18:30+)

### Approche choisie : Port séparé (piste 5)

Le proxy subpath `/onlyoffice/*` ne peut pas fonctionner correctement car :
1. `httputil.ReverseProxy` (Go) **ne gère pas WebSocket/Socket.IO**
2. OO utilise Socket.IO pour toute la communication éditeur
3. Les connexions WebSocket échouent silencieusement, OO voit `Host: localhost:9980`
4. La clé `utils.baseurl` dans `local.json` **n'existe pas** dans OO 9.2 (binaire compilé, pas Node.js)
5. Patcher le nginx interne d'OO n'aurait pas suffi (WebSocket contourne le Director du proxy Go)

### Solution implémentée

Le container OO sert HTTPS directement sur son port (`9980:443`).
Les certs TLS d'Anemone sont montés dans le container.
Le navigateur charge api.js depuis `https://{host}:9980/` — pas de proxy, pas de subpath.

```
Navigateur → https://FR2:9980  (OO direct, HTTPS, api.js + Socket.IO + Editor.bin)
Navigateur → https://FR2:8443  (Anemone, page éditeur, file browser)
OO container → http://host.docker.internal:8080  (download/callback, inchangé)
Anemone → https://FR2:9980  (download fichier édité lors du callback save, InsecureSkipVerify)
```

### Changements

| Fichier | Modification |
|---------|-------------|
| `internal/onlyoffice/docker.go` | `StartContainer(secret, ooURL, certPath, keyPath)` : map `port:443`, mount certs TLS |
| `internal/onlyoffice/docker.go` | `PatchContainerConfig()` simplifié : plus de baseURL, juste SSRF + TLS |
| `internal/web/handlers_admin_onlyoffice.go` | Passe `cfg.TLSCertPath/KeyPath` à StartContainer |
| `internal/web/handlers_onlyoffice_api.go` | Build OO URL externe `https://{browser_host}:{oo_port}`, override CSP, `InsecureSkipVerify` pour download, sudo mv fallback pour save |
| `web/templates/v2/v2_editor.html` | api.js chargé depuis `{{.OOURL}}`, message d'erreur si cert non accepté |

### Bugs trouvés et corrigés lors du test

1. **Editor.bin URL `localhost:9980`** → résolu par HTTPS direct port (pas de proxy)
2. **`x509: certificate signed by unknown authority`** → résolu par `InsecureSkipVerify: true` dans le client HTTP du callback
3. **`permission denied` sur save** → résolu par fallback `sudo /usr/bin/mv` (share d'un autre user)

### À tester

- [ ] Sauvegarde complète (callback status=2 → download → sudo mv → fichier sur disque)
- [ ] Réinstallation propre : container créé via admin UI (pas manuellement)
- [ ] Acceptation certificat OO (:9980) depuis navigateur client distant

---

## Analyse JWT (Piste C) - RÉSOLU

### Structure du code Anemone
```go
// handleFilesEdit construit le config :
editorConfig := map[string]interface{}{
    "document": { "fileType", "key", "title", "url", "permissions" },
    "documentType": "word/cell/slide",
    "editorConfig": { "callbackUrl", "lang", "mode", "user", "customization" },
}

// Signe le config complet comme JWT :
configToken = SignEditorConfig(secret, editorConfig)  // jwt.MapClaims(payload)

// Ajoute le token AU config :
editorConfig["token"] = configToken

// Serialise et envoie au template :
configJSON = json.Marshal(editorConfig)

// Template : new DocsAPI.DocEditor("editor-container", {{.ConfigJSON}})
```

### Ce que le navigateur recoit
```json
{
  "document": {
    "fileType": "md",
    "key": "MjpiYWNrdXBfbWFyYzp0ZXN0Lm1kOjE3NzEwODM2NDU=",
    "title": "test.md",
    "url": "http://host.docker.internal:8080/api/oo/download?token=eyJ...",
    "permissions": { "edit": true, "download": true }
  },
  "documentType": "word",
  "editorConfig": {
    "callbackUrl": "http://host.docker.internal:8080/api/oo/callback",
    "lang": "fr", "mode": "edit",
    "user": { "id": "2", "name": "franck" },
    "customization": { "goback": { "url": "/files?share=backup_marc" } }
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkb2N1..."
}
```

### Ce que le JWT token decode contient
Le meme objet SANS le champ "token" (car token est ajoute apres signature).
A verifier : est-ce qu'OO attend des claims additionnels (exp, iat, iss) ?

### Points a verifier
1. Le JWT contient-il `document.url` ? → OUI (signe avant ajout de "token")
2. Le JWT contient-il `editorConfig.callbackUrl` ? → OUI
3. Le JWT a-t-il des claims standard (exp, iat) ? → NON, seul le payload config
4. Le JWT utilise-t-il HS256 ? → OUI (jwt.SigningMethodHS256)
5. Le secret correspond-il ? → OUI (secret_len=64 = bbef...b93b)

---

## Fichiers modifies (non commites → commités dans ce commit)

| Fichier | Modification |
|---------|-------------|
| `internal/web/handlers_onlyoffice_api.go` | URL = `http://host.docker.internal:8080`, OO URL directe, InsecureSkipVerify, sudo mv |
| `cmd/anemone/main.go` | HTTP server auto-start si OO active |
| `internal/web/handlers_admin_onlyoffice.go` | Proxy conservé (legacy), passe cert paths à StartContainer |
| `internal/onlyoffice/docker.go` | HTTPS `port:443`, mount certs, PatchContainerConfig() simplifié |
| `web/templates/v2/v2_editor.html` | api.js depuis OO direct, message erreur cert |

---

## Commandes utiles

```bash
# Logs Anemone
sudo journalctl -u anemone -f

# Logs OO docservice
docker exec onlyoffice-docs tail -f /var/log/onlyoffice/documentserver/docservice/out.log

# Logs OO converter
docker exec onlyoffice-docs tail -f /var/log/onlyoffice/documentserver/converter/out.log

# Config OO actuelle
docker exec onlyoffice-docs cat /etc/onlyoffice/documentserver/local.json

# Tester connectivite container --> host
docker exec onlyoffice-docs curl -v http://host.docker.internal:8080/api/oo/download?token=test

# Verifier /etc/hosts dans le container
docker exec onlyoffice-docs cat /etc/hosts

# Decoder un JWT (remplacer TOKEN)
echo "TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | python3 -m json.tool

# Secret OO dans le container
docker exec onlyoffice-docs python3 -c "import json; print(json.load(open('/etc/onlyoffice/documentserver/local.json'))['services']['CoAuthoring']['secret'])"

# Secret OO dans Anemone
sqlite3 /srv/anemone/db/anemone.db "SELECT value FROM system_config WHERE key='oo_secret'"
```
