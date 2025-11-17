# 🗂️ Répertoire d'Audit Temporaire

**Date de création** : 2025-11-17
**Objectif** : Stocker temporairement les fichiers suspects avant suppression définitive

## ⚠️ Important

Ce répertoire contient des fichiers qui semblent **obsolètes ou inutilisés** après audit du code.

**NE PAS SUPPRIMER** avant validation finale.

---

## 📦 Contenu

### cmd/
Commandes CLI qui semblent inutilisées en production.

### binaries/
Binaires compilés correspondants aux commandes obsolètes.

---

## 🔄 Processus

1. **Audit** : Fichiers analysés et déplacés ici si suspects
2. **Review** : Vérification manuelle de chaque fichier
3. **Test** : Compilation et tests pour vérifier qu'aucune dépendance n'est cassée
4. **Décision finale** : Suppression définitive ou réintégration

---

## 📋 Fichiers déplacés

| Date | Fichier | Raison | Décision finale |
|------|---------|--------|-----------------|
| 2025-11-17 | `cmd/test-manifest/` | Programme de test/démo uniquement | À confirmer |
| 2025-11-17 | `binaries/test-manifest` | Binaire de test | À confirmer |
| 2025-11-17 | `web/templates/base.html` | Template non utilisé, vestige ancien | À confirmer |

---

**Dernière mise à jour** : 2025-11-17
