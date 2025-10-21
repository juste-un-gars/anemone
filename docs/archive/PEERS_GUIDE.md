# 🤝 Guide de Gestion des Pairs - Anemone

Ce guide explique comment connecter plusieurs serveurs Anemone pour qu'ils se sauvegardent mutuellement de manière sécurisée.

## 📱 Nouvelle Méthode : Interface Web avec QR Code

### Avantages

- ✅ **Simple** : Pas besoin de ligne de commande
- ✅ **Rapide** : Scanner un QR code suffit
- ✅ **Sécurisé** : Protection optionnelle par PIN (4-8 chiffres)
- ✅ **Automatique** : Configuration WireGuard et SSH automatique
- ✅ **Intelligent** : Attribution automatique des IPs VPN

## 🚀 Configuration Rapide

### Méthode 1 : Scanner en Temps Réel (Réseau Local)

**Si les deux serveurs sont accessibles en même temps :**

**Sur le serveur B** :
1. Accédez à `http://localhost:3000/peers`
2. Cochez "Protéger avec un PIN"
3. Cliquez sur "Générer le QR Code"

**Sur le serveur A** :
1. Accédez à `http://localhost:3000/peers`
2. Cliquez sur "📷 Scanner QR Code"
3. Scannez le QR affiché sur l'écran de B
4. Entrez le PIN communiqué oralement
5. ✅ **Terminé !**

### Méthode 2 : Envoi par Email/WhatsApp (Recommandé)

**Cette méthode est SÉCURISÉE et pratique !**

**Sur le serveur B** :
1. Accédez à `http://localhost:3000/peers`
2. ☑️ Cochez "Protéger avec un PIN" (recommandé)
3. Cliquez sur **"Générer le QR Code"**
4. Notez le PIN affiché (ex: `653421`)
5. Cliquez sur **"💾 Télécharger QR Code"**
6. **Envoyez l'image** par email/WhatsApp/Signal à Alice
7. **Appelez Alice** au téléphone et donnez-lui le PIN oralement

**Sur le serveur A** :
1. **Recevez l'image** du QR code par email/WhatsApp
2. Accédez à `http://localhost:3000/peers`
3. Cliquez sur **"📷 Scanner QR Code"**
4. **Scannez l'image** affichée sur votre écran (avec le téléphone)
   - Ou importez l'image si votre scanner le permet
5. Entrez le PIN communiqué par téléphone
6. ✅ **Connexion établie !**

### Pourquoi cette méthode est sécurisée ?

| Canal | Contenu | Sécurité si intercepté |
|-------|---------|------------------------|
| 📧 **Email/WhatsApp** | QR code chiffré | ⚠️ Inutile sans le PIN |
| ☎️ **Téléphone** | PIN uniquement | ⚠️ Inutile sans le QR |
| 🔒 **Les deux ensemble** | Invitation complète | ✅ **Connexion possible** |

**Principe de sécurité** : Un attaquant doit compromettre **DEUX canaux différents** simultanément (quasi impossible).

## 🔐 Sécurité avec PIN

### Pourquoi utiliser un PIN ?

Le PIN protège les informations sensibles dans le QR code (clés publiques, IP VPN, endpoint). Même si quelqu'un intercepte le QR code, il ne pourra pas le déchiffrer sans le PIN.

### Types de protection

| Mode | Sécurité | Facilité | Usage recommandé |
|------|----------|----------|------------------|
| **Sans PIN** | ⚠️ Faible | ✅ Très simple | Réseau local privé uniquement |
| **PIN 4-6 chiffres** | ✅ Bon | ✅ Simple | Usage général (recommandé) |
| **PIN 8 chiffres** | ✅✅ Excellent | ⚠️ Moins pratique | Environnements sensibles |

### Comment communiquer le PIN ?

**Canaux sécurisés recommandés** :
- ☎️ **Téléphone** (appel vocal)
- 📱 **SMS** (si le réseau mobile est sécurisé)
- 🔒 **Signal/Telegram** (messagerie chiffrée)
- 💬 **En personne**

**À éviter** :
- ❌ Email non chiffré
- ❌ QR code partagé publiquement
- ❌ Messagerie non sécurisée

## 📋 Informations Techniques

### Contenu de l'invitation

```json
{
  "version": 2,
  "encrypted": true,
  "salt": "base64...",
  "nonce": "base64...",
  "data": "base64...",  // Données chiffrées
  "hint": "PIN 6 chiffres"
}
```

**Données chiffrées** :
```json
{
  "node_name": "anemone-bob",
  "vpn_ip": "10.8.0.2",
  "wireguard_pubkey": "ABC123...",
  "ssh_pubkey": "ssh-rsa AAAA...",
  "endpoint": "bob.duckdns.org:51820",
  "timestamp": "2025-10-17T14:30:00Z"
}
```

### Attribution automatique des IPs VPN

Le système attribue automatiquement les IPs VPN :
- Premier serveur : `10.8.0.1`
- Deuxième serveur : `10.8.0.2`
- Troisième serveur : `10.8.0.3`
- etc.

**Vérification des conflits** : Le système vérifie automatiquement qu'une IP n'est pas déjà utilisée avant de l'attribuer.

## 🛠️ Configuration Manuelle (Avancé)

Si vous ne pouvez pas utiliser la caméra ou préférez la saisie manuelle :

### Option 1 : Copier/Coller le JSON

1. Sur serveur B : Cliquez sur **"📋 Copier le code JSON"**
2. Envoyez le JSON via un canal sécurisé
3. Sur serveur A : Cliquez sur **"✏️ Saisie manuelle"**
4. Collez le JSON et entrez le PIN

### Option 2 : API REST

```bash
# Générer une invitation (serveur B)
curl -X POST http://localhost:3000/api/peers/generate-invitation \
  -H "Content-Type: application/json" \
  -d '{"pin": "123456"}'

# Ajouter un pair (serveur A)
curl -X POST http://localhost:3000/api/peers/add \
  -H "Content-Type: application/json" \
  -d '{
    "invitation_data": {...},
    "pin": "123456"
  }'
```

## 📊 Gestion des Pairs

### Voir la liste des pairs

```bash
# Via l'interface web
http://localhost:3000/peers

# Via l'API
curl http://localhost:3000/api/peers/list
```

### Vérifier le statut d'un pair

L'interface affiche automatiquement :
- 🟢 **Connecté** : Le pair est joignable via VPN
- 🔴 **Déconnecté** : Le pair n'est pas accessible

### Supprimer un pair

1. Interface web : Cliquez sur **"Supprimer"** à côté du pair
2. Confirmez la suppression
3. La configuration WireGuard est automatiquement mise à jour

## 🔍 Diagnostic

### Le QR code ne se scanne pas

**Solutions** :
- Vérifiez que la caméra est autorisée dans le navigateur
- Augmentez la luminosité de l'écran
- Rapprochez/éloignez la caméra
- Utilisez la saisie manuelle en alternative

### Le pair ne se connecte pas (🔴 Déconnecté)

**Vérifications** :

1. **Endpoint public** : Vérifiez que l'endpoint est correct
   ```bash
   # Sur le serveur distant
   curl http://localhost:3000/api/peers/local-info
   ```

2. **Pare-feu** : Le port 51820/UDP doit être ouvert
   ```bash
   sudo ufw allow 51820/udp
   ```

3. **WireGuard** : Vérifiez que le tunnel est actif
   ```bash
   docker exec anemone-wireguard wg show
   ```

4. **Ping VPN** : Testez la connectivité directe
   ```bash
   docker exec anemone-restic ping 10.8.0.2
   ```

### Erreur "PIN incorrect"

**Causes possibles** :
- Le PIN a été mal saisi
- Le QR code a été régénéré (nouveau PIN)
- Les données ont été corrompues

**Solution** : Régénérez une nouvelle invitation avec un nouveau PIN.

### Erreur "IP VPN déjà utilisée"

**Cause** : Deux pairs tentent d'utiliser la même IP VPN.

**Solution** :
1. Vérifiez les IPs utilisées : `curl http://localhost:3000/api/peers/next-ip`
2. Régénérez l'invitation (une nouvelle IP sera attribuée automatiquement)

## 🔄 Mise à Jour depuis l'Ancienne Méthode

Si vous utilisiez le script `add-peer.sh`, vous pouvez maintenant utiliser l'interface web :

**Migration** :
1. Les pairs existants restent configurés
2. Les nouveaux pairs peuvent être ajoutés via l'interface web
3. Vous pouvez supprimer et ré-ajouter les anciens pairs via l'interface

**Avantages** :
- Plus besoin de saisir manuellement les clés
- Attribution automatique des IPs VPN
- Configuration en un clic

## 📚 API Reference

### Endpoints disponibles

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/peers/generate-invitation` | POST | Génère une invitation chiffrée |
| `/api/peers/add` | POST | Ajoute un pair depuis une invitation |
| `/api/peers/list` | GET | Liste tous les pairs |
| `/api/peers/{name}/status` | GET | Statut d'un pair spécifique |
| `/api/peers/{name}` | DELETE | Supprime un pair |
| `/api/peers/local-info` | GET | Informations locales du serveur |
| `/api/peers/next-ip` | GET | Prochaine IP VPN disponible |

### Exemple complet

```bash
# 1. Générer une invitation avec PIN auto (serveur B)
INVITE=$(curl -s -X POST http://localhost:3000/api/peers/generate-invitation \
  -H "Content-Type: application/json" \
  -d '{"pin_length": 6}')

echo $INVITE | jq -r '.pin'  # Affiche le PIN

# 2. Ajouter le pair (serveur A)
curl -X POST http://localhost:3000/api/peers/add \
  -H "Content-Type: application/json" \
  -d "{
    \"invitation_data\": $(echo $INVITE | jq '.invitation'),
    \"pin\": \"$(echo $INVITE | jq -r '.pin')\"
  }"

# 3. Vérifier le statut
curl http://localhost:3000/api/peers/anemone-bob/status
```

## 🔒 Bonnes Pratiques

### Sécurité

1. **Toujours utiliser un PIN** pour les invitations
2. **Communiquer le PIN séparément** du QR code
3. **Vérifier l'endpoint public** avant de générer l'invitation
4. **Supprimer les pairs inutilisés** régulièrement

### Performance

1. **Limiter le nombre de pairs** à 10-15 pour de meilleures performances
2. **Utiliser des endpoints DynDNS** pour les IPs dynamiques
3. **Configurer le keepalive** (déjà fait automatiquement à 25s)

### Organisation

1. **Nommer clairement les serveurs** (node name descriptif)
2. **Documenter la topologie** réseau pour référence future
3. **Tester la connectivité** après chaque ajout de pair

## 🆘 Support

Pour plus d'aide :
- Documentation générale : `README.md`
- Guide d'interconnexion détaillé : `INTERCONNEXION_GUIDE.md`
- Troubleshooting : `TROUBLESHOOTING.md`
- Issues GitHub : https://github.com/votre-repo/anemone/issues
