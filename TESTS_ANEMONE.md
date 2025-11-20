# Tests Anemone - Session 25 (Finale)

**Date**: 20 Nov 2025
**Objectif**: Validation complète du bon fonctionnement d'Anemone
**Statut**: 🟡 EN COURS

## 🖥️ Infrastructure de test

| Serveur | IP | Rôle | Langue |
|---------|-----|------|--------|
| FR1 | 192.168.83.16 | Serveur principal 1 | Français |
| FR2 | 192.168.83.37 | Serveur principal 2 | Anglais |
| FR3 | 192.168.83.38 | Backup pour FR1 et FR2 | - |
| FR4 | 192.168.83.45 | Restauration de FR1 | - |
| FR5 | 192.168.83.46 | Restauration de FR2 | - |

## 👥 Utilisateurs de test

- **test** : Créé sur FR1 et FR2 (deux personnes différentes, même nom)
- **marc** : Créé sur FR1 et FR2 (deux personnes différentes, même nom)

---

## 📋 Plan de tests

### Phase 1 : Configuration initiale et validation de base

#### 1.1 - Installation et configuration FR1 (français)
- [x] Installation d'Anemone sur FR1
- [x] Configuration en français
- [x] Création admin
- [x] Status: ✅ RÉUSSI

#### 1.2 - Installation et configuration FR2 (anglais)
- [x] Installation d'Anemone sur FR2
- [x] Configuration en anglais
- [x] Création admin
- [x] Status: ✅ RÉUSSI

#### 1.3 - Installation FR3 (backup)
- [x] Installation d'Anemone sur FR3
- [x] Configuration comme serveur backup
- [x] Status: ✅ RÉUSSI

---

### Phase 2 : Tests de validation des mots de passe

#### 2.1 - Validation création admin (mots de passe différents)
- [x] Tenter création admin avec 2 mots de passe différents
- [x] **Résultat attendu**: ❌ Doit refuser
- [x] **Résultat obtenu**: ✅ Système refuse bien la création
- [x] Status: ✅ RÉUSSI

#### 2.2 - Validation création utilisateur (mots de passe différents)
- [x] Tenter création utilisateur avec 2 mots de passe différents
- [x] **Résultat attendu**: ❌ Doit refuser
- [x] **Résultat obtenu**: ✅ Système refuse bien la création
- [x] Status: ✅ RÉUSSI
- [x] **Bonus**: Test suppression/recréation utilisateur → OK
- [x] **Bonus**: Test suppression utilisateur avec sync → Données protégées par chiffrement mais restent sur pairs (voir notes RGPD)

---

### Phase 3 : Création des utilisateurs de test

#### 3.1 - Création utilisateur "test" sur FR1
- [x] Créer utilisateur "test" sur FR1
- [x] Upload de quelques fichiers test
- [x] **Résultat**: ✅ Utilisateur créé avec succès
- [x] Status: ✅ RÉUSSI

#### 3.2 - Création utilisateur "test" sur FR2
- [x] Créer utilisateur "test" sur FR2
- [x] Upload de quelques fichiers test
- [x] **Résultat**: ✅ Utilisateur créé avec succès (personne différente, même nom)
- [x] Status: ✅ RÉUSSI

#### 3.3 - Création utilisateur "marc" sur FR1
- [x] Créer utilisateur "marc" sur FR1
- [x] Upload de quelques fichiers test
- [x] **Résultat**: ✅ Utilisateur créé avec succès
- [x] Status: ✅ RÉUSSI

#### 3.4 - Création utilisateur "marc" sur FR2
- [x] Créer utilisateur "marc" sur FR2
- [x] Upload de quelques fichiers test
- [x] **Résultat**: ✅ Utilisateur créé avec succès (personne différente, même nom)
- [x] Status: ✅ RÉUSSI

#### 3.5 - Synchronisation et vérification isolation
- [x] Synchroniser FR1 et FR2 vers FR3
- [x] Vérifier que chaque utilisateur voit UNIQUEMENT ses propres backups
- [x] **Observation**: Sur FR3, répertoires avec ID unique (ex: 4_test, 5_test)
- [x] **Résultat**: ✅ Isolation parfaite - Aucune fuite de données entre utilisateurs
- [x] Status: ✅ RÉUSSI

---

### Phase 4 : Tests de la corbeille

#### 4.1 - Test suppression fichier (utilisateur test sur FR1)
- [x] Supprimer un fichier de l'utilisateur test
- [x] Vérifier présence dans la corbeille
- [x] **Résultat**: ✅ Fichier bien présent dans la corbeille
- [x] Status: ✅ RÉUSSI

#### 4.2 - Test restauration depuis corbeille
- [x] Restaurer le fichier supprimé
- [x] Vérifier que le fichier est revenu
- [x] **Résultat**: ✅ Fichier restauré avec succès
- [x] Status: ✅ RÉUSSI

#### 4.3 - Test suppression définitive depuis corbeille
- [x] Supprimer définitivement un fichier de la corbeille
- [x] Vérifier qu'il n'est plus récupérable
- [x] **Résultat**: ✅ Fichier définitivement supprimé, non récupérable
- [x] Status: ✅ RÉUSSI

---

### Phase 5 : Tests de connexion de pairs (mauvais mot de passe)

#### 5.1 - Connexion FR1 → FR3 avec mauvais mot de passe
- [x] Tenter connexion avec mauvais mot de passe
- [x] **Résultat attendu**: ❌ Connexion refusée
- [x] **Résultat obtenu**: ✅ Connexion refusée correctement
- [x] Status: ✅ RÉUSSI

#### 5.2 - Connexion FR2 → FR3 avec mauvais mot de passe
- [x] Tenter connexion avec mauvais mot de passe
- [x] **Résultat attendu**: ❌ Connexion refusée
- [x] **Résultat obtenu**: ✅ Connexion refusée correctement
- [x] Status: ✅ RÉUSSI

---

### Phase 6 : Tests de connexion de pairs (correction et bon mot de passe)

#### 6.1 - Connexion FR1 → FR3 avec bon mot de passe
- [x] Corriger le mot de passe
- [x] Tenter connexion avec bon mot de passe
- [x] **Résultat attendu**: ✅ Connexion réussie
- [x] **Résultat obtenu**: ✅ Connexion réussie
- [x] Status: ✅ RÉUSSI

#### 6.2 - Connexion FR2 → FR3 avec bon mot de passe
- [x] Corriger le mot de passe
- [x] Tenter connexion avec bon mot de passe
- [x] **Résultat attendu**: ✅ Connexion réussie
- [x] **Résultat obtenu**: ✅ Connexion réussie
- [x] Status: ✅ RÉUSSI

---

### Phase 7 : Tests de connexion de pairs (changement bon → mauvais)

#### 7.1 - Changement mot de passe FR1 → FR3 (bon → mauvais)
- [x] Changer le bon mot de passe pour un mauvais sur FR1
- [x] **Résultat attendu**: ❌ Connexion ne doit plus fonctionner
- [x] **Résultat obtenu**: ✅ Connexion ne fonctionne plus
- [x] Status: ✅ RÉUSSI

#### 7.2 - Remise du bon mot de passe FR1 → FR3
- [x] Remettre le bon mot de passe sur FR1
- [x] **Résultat attendu**: ✅ Connexion doit refonctionner
- [x] **Résultat obtenu**: ✅ Connexion refonctionne
- [x] Status: ✅ RÉUSSI

---

### Phase 8 : Activation et test de la synchronisation

#### 8.1 - Activation synchro FR1 → FR3
- [x] Activer la synchronisation automatique FR1 → FR3
- [x] **Résultat**: ✅ Synchronisation activée avec succès
- [x] Status: ✅ RÉUSSI

#### 8.2 - Activation synchro FR2 → FR3
- [x] Activer la synchronisation automatique FR2 → FR3
- [x] **Résultat**: ✅ Synchronisation activée avec succès
- [x] Status: ✅ RÉUSSI

#### 8.3 - Vérification synchronisation FR1 → FR3
- [x] Vérifier que les fichiers de FR1 sont bien synchronisés sur FR3
- [x] Vérifier les logs de synchronisation
- [x] **Résultat**: ✅ Fichiers bien synchronisés
- [x] Status: ✅ RÉUSSI

#### 8.4 - Vérification synchronisation FR2 → FR3
- [x] Vérifier que les fichiers de FR2 sont bien synchronisés sur FR3
- [x] Vérifier les logs de synchronisation
- [x] **Résultat**: ✅ Fichiers bien synchronisés
- [x] Status: ✅ RÉUSSI

---

### Phase 9 : Tests de restauration depuis FR3

#### 9.1 - Restauration fichiers utilisateur "test" depuis FR3 (backup FR1)
- [x] Se connecter sur FR1 en tant que "test"
- [x] Restaurer des fichiers depuis FR3
- [x] Vérifier que les fichiers sont bien restaurés
- [x] **Résultat**: ✅ Fichiers restaurés avec succès
- [x] Status: ✅ RÉUSSI

#### 9.2 - Restauration fichiers utilisateur "test" depuis FR3 (backup FR2)
- [x] Se connecter sur FR2 en tant que "test"
- [x] Restaurer des fichiers depuis FR3
- [x] Vérifier que les fichiers sont bien restaurés
- [x] **Résultat**: ✅ Fichiers restaurés avec succès
- [x] Status: ✅ RÉUSSI

#### 9.3 - Restauration fichiers utilisateur "marc" depuis FR3 (backup FR1)
- [x] Se connecter sur FR1 en tant que "marc"
- [x] Restaurer des fichiers depuis FR3
- [x] Vérifier que les fichiers sont bien restaurés
- [x] **Résultat**: ✅ Fichiers restaurés avec succès
- [x] Status: ✅ RÉUSSI

#### 9.4 - Restauration fichiers utilisateur "marc" depuis FR3 (backup FR2)
- [x] Se connecter sur FR2 en tant que "marc"
- [x] Restaurer des fichiers depuis FR3
- [x] Vérifier que les fichiers sont bien restaurés
- [x] **Résultat**: ✅ Fichiers restaurés avec succès
- [x] Status: ✅ RÉUSSI

---

### Phase 10 : Préparation disaster recovery

#### 10.1 - Sauvegarde complète FR1
- [ ] Générer le fichier de restauration pour FR1
- [ ] Copier le fichier de restauration en lieu sûr
- [ ] **Fichier**:
- [ ] Status: ⏳

#### 10.2 - Sauvegarde complète FR2
- [ ] Générer le fichier de restauration pour FR2
- [ ] Copier le fichier de restauration en lieu sûr
- [ ] **Fichier**:
- [ ] Status: ⏳

#### 10.3 - Arrêt FR1 et FR2
- [ ] Arrêter le service Anemone sur FR1
- [ ] Arrêter le service Anemone sur FR2
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 11 : Disaster Recovery - Tentative avec mauvais mot de passe

#### 11.1 - Installation FR4 (restauration FR1) - Mauvais mot de passe
- [ ] Installer Anemone sur FR4
- [ ] Lancer script de restauration avec fichier FR1
- [ ] Entrer un **mauvais mot de passe**
- [ ] **Résultat attendu**: ❌ Échec de restauration
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

#### 11.2 - Installation FR5 (restauration FR2) - Mauvais mot de passe
- [ ] Installer Anemone sur FR5
- [ ] Lancer script de restauration avec fichier FR2
- [ ] Entrer un **mauvais mot de passe**
- [ ] **Résultat attendu**: ❌ Échec de restauration
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

---

### Phase 12 : Disaster Recovery - Tentative avec bon mot de passe

#### 12.1 - Restauration FR4 (depuis backup FR1) - Bon mot de passe
- [ ] Relancer script de restauration sur FR4
- [ ] Entrer le **bon mot de passe**
- [ ] Vérifier que la restauration se termine correctement après l'erreur précédente
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 12.2 - Restauration FR5 (depuis backup FR2) - Bon mot de passe
- [ ] Relancer script de restauration sur FR5
- [ ] Entrer le **bon mot de passe**
- [ ] Vérifier que la restauration se termine correctement après l'erreur précédente
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 13 : Vérification post-restauration

#### 13.1 - Vérification nom serveur FR4
- [ ] Vérifier que FR4 a bien le nom "FR1" dans Anemone
- [ ] **Nom attendu**: FR1
- [ ] **Nom obtenu**:
- [ ] Status: ⏳

#### 13.2 - Vérification nom serveur FR5
- [ ] Vérifier que FR5 a bien le nom "FR2" dans Anemone
- [ ] **Nom attendu**: FR2
- [ ] **Nom obtenu**:
- [ ] Status: ⏳

#### 13.3 - Vérification fichiers utilisateur "test" sur FR4
- [ ] Se connecter en tant que "test" sur FR4
- [ ] Vérifier que tous les fichiers sont présents
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 13.4 - Vérification fichiers utilisateur "test" sur FR5
- [ ] Se connecter en tant que "test" sur FR5
- [ ] Vérifier que tous les fichiers sont présents
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 13.5 - Vérification fichiers utilisateur "marc" sur FR4
- [ ] Se connecter en tant que "marc" sur FR4
- [ ] Vérifier que tous les fichiers sont présents
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 13.6 - Vérification fichiers utilisateur "marc" sur FR5
- [ ] Se connecter en tant que "marc" sur FR5
- [ ] Vérifier que tous les fichiers sont présents
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 14 : Tests de fonctionnement post-restauration

#### 14.1 - Test partages et uploads sur FR4
- [ ] Uploader de nouveaux fichiers en tant que "test"
- [ ] Vérifier le fonctionnement des partages
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 14.2 - Test partages et uploads sur FR5
- [ ] Uploader de nouveaux fichiers en tant que "test"
- [ ] Vérifier le fonctionnement des partages
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 14.3 - Test corbeille sur FR4
- [ ] Supprimer un fichier
- [ ] Restaurer depuis la corbeille
- [ ] Supprimer définitivement
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 14.4 - Test corbeille sur FR5
- [ ] Supprimer un fichier
- [ ] Restaurer depuis la corbeille
- [ ] Supprimer définitivement
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 15 : Vérification synchronisation post-restauration

#### 15.1 - Vérification synchro FR4 → FR3 (héritée de FR1)
- [ ] Vérifier que la synchronisation est activée vers FR3
- [ ] Vérifier que les nouveaux fichiers se synchronisent
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 15.2 - Vérification synchro FR5 → FR3 (héritée de FR2)
- [ ] Vérifier que la synchronisation est activée vers FR3
- [ ] Vérifier que les nouveaux fichiers se synchronisent
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 16 : Tests de restauration depuis FR3 post-disaster recovery

#### 16.1 - Restauration depuis FR3 vers FR4 (utilisateur test)
- [ ] Se connecter sur FR4 en tant que "test"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que la restauration fonctionne
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 16.2 - Restauration depuis FR3 vers FR5 (utilisateur test)
- [ ] Se connecter sur FR5 en tant que "test"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que la restauration fonctionne
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 16.3 - Restauration depuis FR3 vers FR4 (utilisateur marc)
- [ ] Se connecter sur FR4 en tant que "marc"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que la restauration fonctionne
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 16.4 - Restauration depuis FR3 vers FR5 (utilisateur marc)
- [ ] Se connecter sur FR5 en tant que "marc"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que la restauration fonctionne
- [ ] **Résultat**:
- [ ] Status: ⏳

---

## 📊 Statistiques des tests

- **Total de tests**: 0/0
- **Tests réussis**: 0 ✅
- **Tests échoués**: 0 ❌
- **Tests en avertissement**: 0 ⚠️
- **Tests en cours**: 0 ⏳

---

## 🔍 Tests bonus (si temps disponible)

### B1 - Quotas utilisateurs
- [ ] Configurer un quota pour un utilisateur
- [ ] Tenter de dépasser le quota
- [ ] Vérifier les avertissements
- [ ] Status: ⏳

### B2 - Vérification chiffrement sur FR3
- [ ] Se connecter sur FR3
- [ ] Vérifier que les fichiers sauvegardés sont chiffrés (non lisibles en clair)
- [ ] Status: ⏳

### B3 - Upload/Download fichiers volumineux
- [ ] Tester upload d'un gros fichier (>100MB)
- [ ] Tester download
- [ ] Status: ⏳

### B4 - Changement de mot de passe utilisateur
- [ ] Se connecter en tant qu'utilisateur
- [ ] Changer son mot de passe
- [ ] Se reconnecter avec le nouveau mot de passe
- [ ] Status: ⏳

### B5 - Vérification des logs
- [ ] Vérifier les logs après chaque opération majeure
- [ ] Chercher des erreurs ou warnings suspects
- [ ] Status: ⏳

### B6 - Suppression automatique 30 jours (test long)
- [ ] Configurer une suppression automatique à 30 jours
- [ ] ⚠️ **Note**: Ce test prend 30 jours minimum
- [ ] Status: ⏳

---

## 📝 Notes et observations

### Problèmes rencontrés

#### ⚠️ Dashboard utilisateur - Erreur template (RÉSOLU)
- **Symptôme**: Internal Server Error sur dashboard utilisateur
- **Cause**: Fonction T dans router.go ne supportait pas les paramètres de substitution
- **Solution**: Utilisation du FuncMap() du Translator avec support des paramètres
- **Commit**: 08bafee
- **Status**: ✅ RÉSOLU

### Améliorations suggérées

#### 🔒 CRITIQUE - Suppression utilisateur et RGPD
- **Problème identifié**: Quand un utilisateur est supprimé sur le serveur principal :
  - ✅ Données locales supprimées correctement
  - ✅ Partages SMB supprimés correctement
  - ❌ Backups restent sur les serveurs pairs (FR3)
  - ✅ **Protection par chiffrement**: Un nouvel utilisateur avec le même nom ne peut PAS déchiffrer les anciennes données (clé différente)
  - ❌ **Problème RGPD**: Violation du droit à l'oubli (Article 17) - les données doivent être supprimées même si chiffrées

- **Test effectué**:
  1. Créé utilisateur "test" avec des fichiers
  2. Synchronisé sur FR3
  3. Supprimé "test" sur FR1
  4. Recréé "test" avec mot de passe différent
  5. ✅ Les anciennes données sont visibles dans "Parcourir les backups" mais NON déchiffrables
  6. ❌ Les données restent stockées sur FR3 (problème RGPD)

- **Solution à implémenter**:
  - Option A: Suppression immédiate sur les pairs via API lors de la suppression utilisateur
  - Option B: Marquage "deleted" + suppression automatique après X jours
  - Option C: Confirmation admin "Supprimer aussi les backups sur les pairs ?"

- **Priorité**: HAUTE (conformité RGPD)
- **Status**: À implémenter après les tests Session 25

#### ⚠️ IMPORTANT - Synchronisation des suppressions de fichiers
- **Problème identifié**: Quand un fichier est supprimé (mis à la corbeille) sur le serveur principal :
  - ✅ Fichier va bien dans la corbeille locale
  - ✅ Interface web filtre correctement (fichier n'apparaît pas dans "Restaurer")
  - ❌ Fichier reste physiquement présent sur FR3 (serveur pair)
  - ❌ **Impact**: Fichiers orphelins qui consomment de l'espace disque inutilement
  - ❌ **Impact RGPD**: Même problème que pour les utilisateurs supprimés

- **Test effectué**:
  1. Supprimé des fichiers de l'utilisateur "test" sur FR1 (mis à la corbeille)
  2. Vérifié sur FR3 : fichiers toujours présents physiquement dans le répertoire
  3. Testé "Restaurer" via interface web : fichiers n'apparaissent pas (bon)
  4. Conclusion : Logique de filtrage OK, mais synchronisation des suppressions manquante

- **Solution à implémenter**:
  - Option A: Synchroniser les suppressions (corbeille) vers les pairs
  - Option B: Synchroniser les suppressions définitives vers les pairs
  - Option C: Les deux (recommandé)

- **Priorité**: MOYENNE-HAUTE (espace disque + cohérence des données)
- **Status**: À implémenter après les tests Session 25

### Points d'attention

- Les tests de la Phase 16 valident que le système de backup/restore fonctionne de bout en bout
- La synchronisation doit être héritée correctement après disaster recovery
- Les noms de serveurs doivent être préservés lors de la restauration

### Observations positives

#### 🔒 Système d'ID unique pour les utilisateurs
- **Observation**: Sur FR3, chaque utilisateur a un répertoire avec ID unique (ex: `4_test`, `5_test`)
- **Avantage**: Même si un utilisateur est supprimé puis recréé avec le même nom, les données sont isolées
- **Sécurité**:
  - ✅ Impossible de créer deux utilisateurs avec le même nom sur un serveur
  - ✅ Utilisateurs multi-serveurs (test@FR1 et test@FR2) sont bien distincts
  - ✅ Chaque utilisateur ne voit QUE ses propres backups lors de la restauration
  - ✅ Aucune fuite de données entre utilisateurs
  - ✅ Clés de chiffrement uniques par utilisateur/création
- **Status**: Excellente architecture de sécurité ✅

---

## ✅ Résultat final

**Status**: ⏳ EN COURS

### Résumé

(À compléter à la fin des tests)

### Recommandations

(À compléter à la fin des tests)
