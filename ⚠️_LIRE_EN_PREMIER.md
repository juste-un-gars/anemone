# ⚠️ ACTION REQUISE AVANT PROCHAINE UTILISATION

**Date** : 2025-10-29 09:30
**Status** : 🔴 MIGRATION NÉCESSAIRE

---

## 🚨 Problème actuel

Les partages SMB ne sont **pas accessibles** car les données sont dans `/home/franck/anemone/data/`.

Le répertoire `/home/franck` a des permissions `700` qui empêchent les utilisateurs SMB d'y accéder.

**Erreur Samba** :
```
chdir_current_service: vfs_ChDir(/home/franck/anemone/data/shares/test/backup)
failed: Permission non accordée
```

---

## ✅ Solution : Migration vers /srv/anemone

### Fichiers à lire AVANT de continuer :

1. **`MIGRATION_PLAN.md`** ← Plan détaillé étape par étape (15-30 min)
2. **`SESSION_STATE.md`** ← Historique complet + diagnostic

---

## 🎯 Résumé migration (ultra rapide)

```bash
# 1. Arrêter
killall anemone

# 2. Créer destination
sudo mkdir -p /srv/anemone
sudo chown franck:franck /srv/anemone

# 3. Déplacer données
mv ~/anemone/data/* /srv/anemone/

# 4. Permissions
sudo chown -R test:test /srv/anemone/shares/test/
sudo chmod 755 /srv/anemone/shares/test/

# 5. Redémarrer avec nouveau chemin
cd ~/anemone
ANEMONE_DATA_DIR=/srv/anemone ./anemone

# 6. Tester depuis Windows
# Connecter à \\192.168.83.132\backup_test
```

---

## 📁 Fichiers documentation

| Fichier | Description |
|---------|-------------|
| `MIGRATION_PLAN.md` | Plan de migration complet avec troubleshooting |
| `SESSION_STATE.md` | État actuel + historique des sessions |
| `README.md` | Documentation générale du projet |
| `QUICKSTART.md` | Guide de démarrage rapide |

---

## ⏱️ Prochaine session

**Priorité 1** : Suivre `MIGRATION_PLAN.md`
**Temps estimé** : 15-30 minutes
**Risque** : Faible (rollback facile)

---

**⚠️ Ne pas utiliser le NAS avant migration !**
**⚠️ Les partages SMB ne fonctionneront pas !**
