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
- [ ] Tenter création utilisateur avec 2 mots de passe différents
- [ ] **Résultat attendu**: ❌ Doit refuser
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

---

### Phase 3 : Création des utilisateurs de test

#### 3.1 - Création utilisateur "test" sur FR1
- [ ] Créer utilisateur "test" sur FR1
- [ ] Upload de quelques fichiers test
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 3.2 - Création utilisateur "test" sur FR2
- [ ] Créer utilisateur "test" sur FR2
- [ ] Upload de quelques fichiers test
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 3.3 - Création utilisateur "marc" sur FR1
- [ ] Créer utilisateur "marc" sur FR1
- [ ] Upload de quelques fichiers test
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 3.4 - Création utilisateur "marc" sur FR2
- [ ] Créer utilisateur "marc" sur FR2
- [ ] Upload de quelques fichiers test
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 4 : Tests de la corbeille

#### 4.1 - Test suppression fichier (utilisateur test sur FR1)
- [ ] Supprimer un fichier de l'utilisateur test
- [ ] Vérifier présence dans la corbeille
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 4.2 - Test restauration depuis corbeille
- [ ] Restaurer le fichier supprimé
- [ ] Vérifier que le fichier est revenu
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 4.3 - Test suppression définitive depuis corbeille
- [ ] Supprimer définitivement un fichier de la corbeille
- [ ] Vérifier qu'il n'est plus récupérable
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 5 : Tests de connexion de pairs (mauvais mot de passe)

#### 5.1 - Connexion FR1 → FR3 avec mauvais mot de passe
- [ ] Tenter connexion avec mauvais mot de passe
- [ ] **Résultat attendu**: ❌ Connexion refusée
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

#### 5.2 - Connexion FR2 → FR3 avec mauvais mot de passe
- [ ] Tenter connexion avec mauvais mot de passe
- [ ] **Résultat attendu**: ❌ Connexion refusée
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

---

### Phase 6 : Tests de connexion de pairs (correction et bon mot de passe)

#### 6.1 - Connexion FR1 → FR3 avec bon mot de passe
- [ ] Corriger le mot de passe
- [ ] Tenter connexion avec bon mot de passe
- [ ] **Résultat attendu**: ✅ Connexion réussie
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

#### 6.2 - Connexion FR2 → FR3 avec bon mot de passe
- [ ] Corriger le mot de passe
- [ ] Tenter connexion avec bon mot de passe
- [ ] **Résultat attendu**: ✅ Connexion réussie
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

---

### Phase 7 : Tests de connexion de pairs (changement bon → mauvais)

#### 7.1 - Changement mot de passe FR1 → FR3 (bon → mauvais)
- [ ] Changer le bon mot de passe pour un mauvais sur FR1
- [ ] **Résultat attendu**: ❌ Connexion ne doit plus fonctionner
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

#### 7.2 - Remise du bon mot de passe FR1 → FR3
- [ ] Remettre le bon mot de passe sur FR1
- [ ] **Résultat attendu**: ✅ Connexion doit refonctionner
- [ ] **Résultat obtenu**:
- [ ] Status: ⏳

---

### Phase 8 : Activation et test de la synchronisation

#### 8.1 - Activation synchro FR1 → FR3
- [ ] Activer la synchronisation automatique FR1 → FR3
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 8.2 - Activation synchro FR2 → FR3
- [ ] Activer la synchronisation automatique FR2 → FR3
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 8.3 - Vérification synchronisation FR1 → FR3
- [ ] Vérifier que les fichiers de FR1 sont bien synchronisés sur FR3
- [ ] Vérifier les logs de synchronisation
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 8.4 - Vérification synchronisation FR2 → FR3
- [ ] Vérifier que les fichiers de FR2 sont bien synchronisés sur FR3
- [ ] Vérifier les logs de synchronisation
- [ ] **Résultat**:
- [ ] Status: ⏳

---

### Phase 9 : Tests de restauration depuis FR3

#### 9.1 - Restauration fichiers utilisateur "test" depuis FR3 (backup FR1)
- [ ] Se connecter sur FR1 en tant que "test"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que les fichiers sont bien restaurés
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 9.2 - Restauration fichiers utilisateur "test" depuis FR3 (backup FR2)
- [ ] Se connecter sur FR2 en tant que "test"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que les fichiers sont bien restaurés
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 9.3 - Restauration fichiers utilisateur "marc" depuis FR3 (backup FR1)
- [ ] Se connecter sur FR1 en tant que "marc"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que les fichiers sont bien restaurés
- [ ] **Résultat**:
- [ ] Status: ⏳

#### 9.4 - Restauration fichiers utilisateur "marc" depuis FR3 (backup FR2)
- [ ] Se connecter sur FR2 en tant que "marc"
- [ ] Restaurer des fichiers depuis FR3
- [ ] Vérifier que les fichiers sont bien restaurés
- [ ] **Résultat**:
- [ ] Status: ⏳

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

(Aucun pour le moment)

### Améliorations suggérées

(Aucune pour le moment)

### Points d'attention

- Les tests de la Phase 16 valident que le système de backup/restore fonctionne de bout en bout
- La synchronisation doit être héritée correctement après disaster recovery
- Les noms de serveurs doivent être préservés lors de la restauration

---

## ✅ Résultat final

**Status**: ⏳ EN COURS

### Résumé

(À compléter à la fin des tests)

### Recommandations

(À compléter à la fin des tests)
