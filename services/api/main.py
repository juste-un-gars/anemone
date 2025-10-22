#!/usr/bin/env python3
"""
Anemone API - Interface de monitoring et gestion avec setup sécurisé
"""

import os
import yaml
import json
import subprocess
import secrets
import shutil
import base64
from datetime import datetime
from pathlib import Path
from typing import Dict
from fastapi import FastAPI, HTTPException, Form, Request, Body
from fastapi.responses import HTMLResponse, RedirectResponse, JSONResponse, FileResponse, Response
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel
from starlette.middleware.base import BaseHTTPMiddleware
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend
from typing import Optional, List
from peer_manager import PeerManager
from quota_manager import QuotaManager
from crypto_utils import generate_random_pin, validate_pin
from translations import get_all_texts

# Configuration
CONFIG_PATH = os.getenv('CONFIG_PATH', '/config/config.yaml')
WEB_LANGUAGE = os.getenv('WEB_LANGUAGE', 'fr').lower()
if WEB_LANGUAGE not in ['fr', 'en']:
    WEB_LANGUAGE = 'fr'
SETUP_COMPLETED = Path('/config/.setup-completed')
RESTIC_ENCRYPTED = Path('/config/.restic.encrypted')
RESTIC_SALT = Path('/config/.restic.salt')

app = FastAPI(title="Anemone API", version="1.0.0")

# Initialiser le gestionnaire de pairs
peer_manager = PeerManager(config_path=CONFIG_PATH)

# Initialiser le gestionnaire de quotas
quota_manager = QuotaManager(config_path=CONFIG_PATH)

# Templates
templates = Jinja2Templates(directory=str(Path(__file__).parent / "templates"))

# ===== Utilitaires =====

def is_setup_completed() -> bool:
    return SETUP_COMPLETED.exists()

def get_system_key() -> str:
    # IMPORTANT : Utiliser ANEMONE_SYSTEM_ID (partagé entre API et Restic) au lieu de HOSTNAME
    # HOSTNAME diffère entre conteneurs à cause de network_mode: "service:wireguard"
    return os.getenv('ANEMONE_SYSTEM_ID', 'anemone')

def generate_restic_key() -> str:
    return secrets.token_urlsafe(32)

def encrypt_restic_key(key: str) -> bool:
    try:
        # Vérifier que le dossier config existe et est accessible en écriture
        config_dir = Path('/config')
        if not config_dir.exists():
            print(f"ERROR: Config directory does not exist: {config_dir}", flush=True)
            return False

        # Test d'écriture
        try:
            test_file = config_dir / '.test_write'
            test_file.touch()
            test_file.unlink()
        except Exception as e:
            print(f"ERROR: Cannot write to config directory: {e}", flush=True)
            return False

        system_key = get_system_key()
        print(f"DEBUG: System key obtained (length: {len(system_key)})", flush=True)

        salt = secrets.token_bytes(32)
        print(f"DEBUG: Salt generated", flush=True)

        # Derive encryption key using PBKDF2
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=salt,
            iterations=100000,
            backend=default_backend()
        )
        derived_key = kdf.derive(f"{system_key}".encode())
        print(f"DEBUG: Key derived", flush=True)

        # Generate IV for AES-CBC
        iv = secrets.token_bytes(16)

        # Encrypt using AES-256-CBC
        cipher = Cipher(
            algorithms.AES(derived_key),
            modes.CBC(iv),
            backend=default_backend()
        )
        encryptor = cipher.encryptor()
        print(f"DEBUG: Cipher initialized", flush=True)

        # Pad the key to be multiple of 16 bytes (AES block size)
        key_bytes = key.encode()
        padding_length = 16 - (len(key_bytes) % 16)
        padded_key = key_bytes + bytes([padding_length] * padding_length)
        print(f"DEBUG: Key padded (length: {len(padded_key)})", flush=True)

        # Encrypt
        encrypted = encryptor.update(padded_key) + encryptor.finalize()
        print(f"DEBUG: Encryption complete", flush=True)

        # Save encrypted data (IV + encrypted data)
        RESTIC_ENCRYPTED.write_bytes(iv + encrypted)
        print(f"DEBUG: Encrypted key saved to {RESTIC_ENCRYPTED}", flush=True)

        # Save salt as hex for compatibility
        RESTIC_SALT.write_text(salt.hex())
        print(f"DEBUG: Salt saved to {RESTIC_SALT}", flush=True)

        SETUP_COMPLETED.touch()
        print(f"DEBUG: Setup marker created at {SETUP_COMPLETED}", flush=True)

        return True
    except Exception as e:
        import traceback
        print(f"ERROR encrypting key: {e}", flush=True)
        print(f"Traceback: {traceback.format_exc()}", flush=True)
        return False

# ===== Middleware =====

class BasicAuthMiddleware(BaseHTTPMiddleware):
    """HTTP Basic Authentication (optionnel, activé si WEB_PASSWORD est défini)"""
    async def dispatch(self, request: Request, call_next):
        web_password = os.getenv('WEB_PASSWORD', '').strip()

        # Si pas de mot de passe configuré, passer
        if not web_password:
            return await call_next(request)

        # Vérifier l'en-tête Authorization
        auth_header = request.headers.get('Authorization')

        if not auth_header or not auth_header.startswith('Basic '):
            return Response(
                content='Authentication required',
                status_code=401,
                headers={'WWW-Authenticate': 'Basic realm="Anemone"'}
            )

        try:
            # Décoder les credentials (format: "Basic base64(username:password)")
            credentials = base64.b64decode(auth_header[6:]).decode('utf-8')
            username, password = credentials.split(':', 1)

            # Vérifier le mot de passe (username ignoré, seul le mot de passe compte)
            if password == web_password:
                return await call_next(request)
        except:
            pass

        # Credentials invalides
        return Response(
            content='Invalid credentials',
            status_code=401,
            headers={'WWW-Authenticate': 'Basic realm="Anemone"'}
        )

class SetupMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        path = request.url.path

        if not is_setup_completed() and not path.startswith('/setup'):
            return RedirectResponse('/setup', status_code=302)

        if is_setup_completed() and path.startswith('/setup'):
            return RedirectResponse('/', status_code=302)

        return await call_next(request)

app.add_middleware(SetupMiddleware)
app.add_middleware(BasicAuthMiddleware)

# ===== Routes Setup =====

@app.get("/setup", response_class=HTMLResponse)
async def setup_page():
    html = """<!DOCTYPE html>
<html><head><title>🪸 Anemone - Setup</title>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
       min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
.container { background: white; border-radius: 16px; padding: 40px; max-width: 600px; width: 100%;
             box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
h1 { color: #333; margin-bottom: 10px; }
.option { border: 2px solid #e0e0e0; border-radius: 12px; padding: 24px; margin-bottom: 16px;
          cursor: pointer; transition: all 0.3s; }
.option:hover { border-color: #667eea; background: #f8f9ff; }
.option.selected { border-color: #667eea; background: #f0f3ff; }
button { width: 100%; padding: 16px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
         color: white; border: none; border-radius: 8px; font-size: 1.1em; cursor: pointer; margin-top: 20px; }
</style></head><body>
<div class="container">
<h1>🪸 Anemone</h1>
<p style="margin-bottom:40px">Configuration initiale</p>
<div class="option" onclick="select('new')">
<h3>🆕 Nouveau serveur</h3><p>Générer une clé</p></div>
<div class="option" onclick="select('restore')">
<h3>♻️ Restauration</h3><p>J'ai déjà une clé</p></div>
<button onclick="next()">Continuer</button>
</div>
<script>
function select(m) { document.querySelectorAll('.option').forEach(e => e.classList.remove('selected'));
                     event.currentTarget.classList.add('selected'); window.mode = m; }
function next() { window.location = '/setup/' + (window.mode || 'new'); }
select('new');
</script></body></html>"""
    return HTMLResponse(html)

@app.get("/setup/new", response_class=HTMLResponse)
async def setup_new():
    key = generate_restic_key()
    html = f"""<!DOCTYPE html>
<html><head><title>🪸 Clé générée</title><meta charset="utf-8">
<style>
* {{ margin: 0; padding: 0; box-sizing: border-box; }}
body {{ font-family: -apple-system, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        min-height: 100vh; padding: 20px; }}
.container {{ background: white; border-radius: 16px; padding: 40px; max-width: 700px; margin: 0 auto; }}
.warning {{ background: #fff3cd; border-left: 4px solid #ffc107; padding: 16px; margin: 20px 0; }}
.key {{ background: #f8f9fa; border: 2px solid #dee2e6; border-radius: 8px; padding: 20px; margin: 20px 0;
        word-break: break-all; font-family: monospace; }}
.actions {{ display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; margin: 20px 0; }}
.actions button {{ padding: 12px; background: white; border: 2px solid #667eea; color: #667eea;
                    border-radius: 8px; cursor: pointer; }}
button {{ width: 100%; padding: 16px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          color: white; border: none; border-radius: 8px; font-size: 1.1em; cursor: pointer; }}
button:disabled {{ background: #ccc; }}
</style></head><body>
<div class="container">
<h1>✅ Clé générée</h1>
<div class="warning">⚠️ SAUVEGARDEZ CETTE CLÉ MAINTENANT</div>
<div class="key" id="key">{key}</div>
<div class="actions">
<button onclick="copyKey()">📋 Copier</button>
<button onclick="dl()">💾 Télécharger</button>
</div>
<label style="display:block; margin:20px 0;">
<input type="checkbox" id="ok" onchange="document.getElementById('btn').disabled=!this.checked">
J'ai sauvegardé ma clé</label>
<form method="POST" action="/setup/complete">
<input type="hidden" name="key" value="{key}">
<button id="btn" disabled>Continuer</button>
</form>
</div>
<script>
function copyKey() {{
  const text = document.getElementById('key').textContent;

  // Vérifier si on est en contexte sécurisé (HTTPS ou localhost)
  const isSecure = window.isSecureContext ||
                   window.location.hostname === 'localhost' ||
                   window.location.hostname === '127.0.0.1';

  if (isSecure && navigator.clipboard) {{
    // Méthode moderne pour HTTPS/localhost
    navigator.clipboard.writeText(text)
      .then(() => {{
        alert('✅ Clé copiée dans le presse-papier !');
      }})
      .catch(() => {{
        // Si ça échoue, afficher le modal
        showCopyDialog(text);
      }});
  }} else {{
    // Pour HTTP via IP, afficher directement le modal
    showCopyDialog(text);
  }}
}}

function showCopyDialog(text) {{
  const overlay = document.createElement('div');
  overlay.style.cssText = 'position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.5); z-index:9999; display:flex; align-items:center; justify-content:center; padding:20px';
  overlay.onclick = (e) => {{ if (e.target === overlay) overlay.remove(); }};

  const dialog = document.createElement('div');
  dialog.style.cssText = 'background:white; padding:30px; border-radius:16px; box-shadow:0 10px 40px rgba(0,0,0,0.3); max-width:600px; width:100%';

  const title = document.createElement('h3');
  title.style.cssText = 'margin-bottom:16px; color:#333; text-align:center';
  title.textContent = '📋 Copiez cette clé manuellement';

  const info = document.createElement('p');
  info.style.cssText = 'margin-bottom:12px; color:#666; text-align:center';
  info.innerHTML = '⚠️ La copie automatique nécessite HTTPS ou localhost<br>Sélectionnez le texte ci-dessous et utilisez Ctrl+C (ou Cmd+C sur Mac)';

  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.readOnly = true;
  textarea.style.cssText = 'width:100%; height:120px; padding:12px; font-family:monospace; font-size:13px; border:2px solid #667eea; border-radius:8px; resize:vertical; margin:12px 0';

  const button = document.createElement('button');
  button.textContent = 'Fermer';
  button.style.cssText = 'width:100%; padding:14px; background:linear-gradient(135deg, #667eea 0%, #764ba2 100%); color:white; border:none; border-radius:8px; font-size:1em; cursor:pointer';
  button.onclick = () => overlay.remove();

  dialog.appendChild(title);
  dialog.appendChild(info);
  dialog.appendChild(textarea);
  dialog.appendChild(button);
  overlay.appendChild(dialog);
  document.body.appendChild(overlay);

  // Sélectionner automatiquement le texte
  textarea.focus();
  textarea.select();
}}

function dl() {{
  const blob = new Blob([document.getElementById('key').textContent], {{type: 'text/plain'}});
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url; a.download = 'anemone-key.txt';
  a.click(); URL.revokeObjectURL(url);
}}
</script></body></html>"""
    return HTMLResponse(html)

@app.get("/setup/restore", response_class=HTMLResponse)
async def setup_restore():
    html = """<!DOCTYPE html>
<html><head><title>🪸 Restauration</title><meta charset="utf-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
       min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
.container { background: white; border-radius: 16px; padding: 40px; max-width: 600px; }
input { width: 100%; padding: 12px; border: 2px solid #dee2e6; border-radius: 8px; font-family: monospace; }
button { width: 100%; padding: 16px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
         color: white; border: none; border-radius: 8px; font-size: 1.1em; cursor: pointer; margin-top: 20px; }
</style></head><body>
<div class="container">
<h1>♻️ Restauration</h1>
<p style="margin:20px 0">Collez votre clé Restic :</p>
<form method="POST" action="/setup/complete">
<input type="password" name="key" required placeholder="Depuis Bitwarden...">
<button>Valider</button>
</form>
</div></body></html>"""
    return HTMLResponse(html)

@app.post("/setup/complete")
async def setup_complete(key: str = Form(...)):
    key = key.strip()
    if len(key) < 20:
        raise HTTPException(400, "Clé invalide")
    
    if not encrypt_restic_key(key):
        raise HTTPException(500, "Erreur lors du chiffrement")
    
    html = """<!DOCTYPE html>
<html><head><title>✅ Terminé</title><meta http-equiv="refresh" content="5;url=/">
<style>
body { font-family: -apple-system, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
       min-height: 100vh; display: flex; align-items: center; justify-content: center; }
.container { background: white; border-radius: 16px; padding: 40px; max-width: 600px; text-align: center; }
h1 { color: #28a745; font-size: 2.5em; }
.success { background: #d4edda; padding: 16px; margin: 20px 0; border-radius: 4px; }
a { display: inline-block; margin-top: 20px; padding: 12px 24px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white; text-decoration: none; border-radius: 8px; }
</style></head><body>
<div class="container">
<h1>✅ Configuration terminée</h1>
<div class="success">La clé a été enregistrée de manière sécurisée</div>
<p>⚠️ Cette page ne s'affichera plus jamais</p>
<a href="/">Dashboard</a>
<p style="margin-top:20px; color:#666">Redirection dans 5s...</p>
</div></body></html>"""
    return HTMLResponse(html)

# ===== Routes principales =====

def load_config() -> Dict:
    try:
        with open(CONFIG_PATH, 'r') as f:
            return yaml.safe_load(f)
    except:
        return {}

@app.get("/", response_class=HTMLResponse)
async def root():
    config = load_config()
    name = config.get('node', {}).get('name', 'Anemone')
    t = get_all_texts(WEB_LANGUAGE)
    html = f"""<!DOCTYPE html>
<html><head><title>🪸 {name}</title>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
* {{ margin: 0; padding: 0; box-sizing: border-box; }}
body {{
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    padding: 20px;
}}
.container {{ max-width: 1200px; margin: 0 auto; }}
.header {{ text-align: center; color: white; margin-bottom: 30px; }}
.header h1 {{ font-size: 2.5em; margin-bottom: 10px; }}
.nav {{
    display: flex;
    gap: 10px;
    justify-content: center;
    margin-bottom: 30px;
    flex-wrap: wrap;
}}
.nav a {{
    background: rgba(255,255,255,0.2);
    color: white;
    padding: 12px 24px;
    border-radius: 8px;
    text-decoration: none;
    transition: all 0.3s;
    font-weight: 500;
}}
.nav a:hover {{ background: rgba(255,255,255,0.3); transform: translateY(-2px); }}
.nav a.active {{ background: rgba(255,255,255,0.4); font-weight: 600; }}
.grid {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }}
.card {{
    background: white;
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 10px 30px rgba(0,0,0,0.2);
}}
.card h2 {{ color: #333; margin-bottom: 16px; }}
.stat {{ display: flex; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid #eee; }}
.stat:last-child {{ border-bottom: none; }}
.stat-label {{ color: #666; }}
.stat-value {{ font-weight: 600; color: #333; }}
.status {{ padding: 4px 12px; border-radius: 12px; font-size: 0.9em; }}
.status-up {{ background: #d4edda; color: #155724; }}
.status-down {{ background: #f8d7da; color: #721c24; }}
.progress {{
    background: #e0e0e0;
    border-radius: 8px;
    height: 8px;
    margin-top: 8px;
    overflow: hidden;
}}
.progress-bar {{
    background: linear-gradient(90deg, #667eea, #764ba2);
    height: 100%;
    transition: width 0.3s;
}}
.loading {{ text-align: center; color: #999; padding: 20px; }}
</style>
</head><body>
<div class="container">
    <div class="header">
        <h1>🪸 {name}</h1>
        <p>{t['dashboard_title']}</p>
    </div>

    <div class="nav">
        <a href="/" class="active">🏠 {t['home']}</a>
        <a href="/peers">👥 {t['peers']}</a>
        <a href="/recovery">🔄 Recovery</a>
        <a href="/api/status">📊 {t['api_status']}</a>
    </div>

    <div class="grid">
        <!-- VPN Status -->
        <div class="card">
            <h2>🔒 {t['vpn_status']}</h2>
            <div id="vpn-status" class="loading">{t['loading']}</div>
        </div>

        <!-- Storage -->
        <div class="card">
            <h2>💾 {t['storage']}</h2>
            <div id="storage-stats" class="loading">{t['loading']}</div>
        </div>

        <!-- Disk Usage -->
        <div class="card">
            <h2>💿 {t['disk']}</h2>
            <div id="disk-stats" class="loading">{t['loading']}</div>
        </div>

        <!-- Restic Snapshots Status -->
        <div class="card">
            <h2>📦 Snapshots Restic</h2>
            <div id="restic-status" class="loading">{t['loading']}</div>
        </div>
    </div>
</div>

<script>
// Traductions injectées côté serveur
const t = {{
    status: "{t['status']}",
    active: "{t['active']}",
    inactive: "{t['inactive']}",
    connected_peers: "{t['connected_peers']}",
    data_local: "{t['data_local']}",
    backup_saved: "{t['backup_saved']}",
    backups_received: "{t['backups_received']}",
    total: "{t['total']}",
    used: "{t['used']}",
    free: "{t['free']}",
    loading_error: "{t['loading_error']}"
}};

async function loadStats() {{
    try {{
        const res = await fetch('/api/system/stats');
        const data = await res.json();

        // VPN Status
        const vpnHtml = `
            <div class="stat">
                <span class="stat-label">${{t.status}}</span>
                <span class="status ${{data.vpn.status === 'up' ? 'status-up' : 'status-down'}}">
                    ${{data.vpn.status === 'up' ? '🟢 ' + t.active : '🔴 ' + t.inactive}}
                </span>
            </div>
            <div class="stat">
                <span class="stat-label">${{t.connected_peers}}</span>
                <span class="stat-value">${{data.vpn.peers_connected}}</span>
            </div>
        `;
        document.getElementById('vpn-status').innerHTML = vpnHtml;

        // Storage Stats
        const storageHtml = `
            <div class="stat">
                <span class="stat-label">📁 ${{t.data_local}}</span>
                <span class="stat-value">${{data.storage.data.formatted}}</span>
            </div>
            <div class="stat">
                <span class="stat-label">💼 ${{t.backup_saved}}</span>
                <span class="stat-value">${{data.storage.backup.formatted}}</span>
            </div>
            <div class="stat">
                <span class="stat-label">📦 ${{t.backups_received}}</span>
                <span class="stat-value">${{data.storage.backups.formatted}}</span>
            </div>
            <div class="stat" style="margin-top:8px; padding-top:16px; border-top:2px solid #eee">
                <span class="stat-label"><strong>${{t.total}}</strong></span>
                <span class="stat-value"><strong>${{data.storage.total.formatted}}</strong></span>
            </div>
        `;
        document.getElementById('storage-stats').innerHTML = storageHtml;

        // Disk Stats
        const diskPercent = data.disk.percent;
        const diskHtml = `
            <div class="stat">
                <span class="stat-label">${{t.used}}</span>
                <span class="stat-value">${{data.disk.used.formatted}}</span>
            </div>
            <div class="stat">
                <span class="stat-label">${{t.free}}</span>
                <span class="stat-value">${{data.disk.free.formatted}}</span>
            </div>
            <div class="stat">
                <span class="stat-label">${{t.total}}</span>
                <span class="stat-value">${{data.disk.total.formatted}}</span>
            </div>
            <div class="progress">
                <div class="progress-bar" style="width: ${{diskPercent}}%"></div>
            </div>
            <div style="text-align:center; margin-top:8px; color:#666; font-size:0.9em">
                ${{diskPercent}}% ${{t.used.toLowerCase()}}
            </div>
        `;
        document.getElementById('disk-stats').innerHTML = diskHtml;

    }} catch (err) {{
        console.error(err);
        const errorMsg = '<p style="color:red">' + t.loading_error + '</p>';
        document.getElementById('vpn-status').innerHTML = errorMsg;
        document.getElementById('storage-stats').innerHTML = errorMsg;
        document.getElementById('disk-stats').innerHTML = errorMsg;
    }}
}}

async function loadResticStatus() {{
    try {{
        const res = await fetch('/api/restic/status');
        const data = await res.json();

        if (data.total_targets === 0) {{
            document.getElementById('restic-status').innerHTML =
                '<p style="color:#999;text-align:center">Aucune destination de backup configurée</p>';
            return;
        }}

        let resticHtml = '';

        // Afficher un indicateur global
        const globalEmoji = data.global_status === 'ok' ? '🟢' :
                           data.global_status === 'warning' ? '🟡' :
                           data.global_status === 'error' ? '🔴' : '⚪';

        resticHtml += `
            <div class="stat" style="background:#f8f9fa;border-radius:8px;padding:12px;margin-bottom:12px">
                <span class="stat-label"><strong>État global</strong></span>
                <span class="stat-value">${{globalEmoji}} ${{data.global_status.toUpperCase()}}</span>
            </div>
        `;

        // Afficher chaque target de backup
        data.targets.forEach(target => {{
            const statusColor = target.status === 'ok' ? '#28a745' :
                               target.status === 'warning' ? '#ffc107' :
                               target.status === 'not_initialized' ? '#6c757d' :
                               '#dc3545';
            const statusEmoji = target.status === 'ok' ? '✅' :
                               target.status === 'warning' ? '⚠️' :
                               target.status === 'not_initialized' ? '⏸️' :
                               '❌';

            resticHtml += `
                <div class="stat" style="display:block;padding:16px">
                    <div style="display:flex;justify-content:space-between;align-items:flex-start">
                        <div>
                            <div style="font-weight:600;color:#333">${{statusEmoji}} ${{target.name}}</div>
                            <div style="font-size:0.85em;color:#666;margin-top:4px">
                                ${{target.last_snapshot ?
                                    target.last_snapshot.time_formatted + ' (' + target.last_snapshot.age_hours + 'h)' :
                                    target.message || 'Aucun snapshot'}}
                            </div>
                        </div>
                        <div style="text-align:right">
                            <div style="font-size:0.85em;color:${{statusColor}};font-weight:600">
                                ${{target.status.replace('_', ' ').toUpperCase()}}
                            </div>
                            ${{target.last_snapshot ?
                                '<div style="font-size:0.75em;color:#999;margin-top:4px">' + target.last_snapshot.id + '</div>' :
                                ''}}
                        </div>
                    </div>
                    ${{target.status !== 'ok' ? `
                        <div style="margin-top:12px;display:flex;gap:8px;flex-wrap:wrap">
                            <button
                                onclick="testConnection('${{target.name}}')"
                                style="flex:1;min-width:120px;padding:8px 12px;background:#667eea;color:white;border:none;border-radius:6px;cursor:pointer;font-size:0.85em"
                            >
                                🔍 Diagnostiquer
                            </button>
                            ${{target.status === 'not_initialized' || target.status === 'error' ? `
                                <button
                                    onclick="initRepository('${{target.name}}')"
                                    style="flex:1;min-width:120px;padding:8px 12px;background:#28a745;color:white;border:none;border-radius:6px;cursor:pointer;font-size:0.85em"
                                >
                                    🚀 Initialiser
                                </button>
                            ` : ''}}
                        </div>
                    ` : ''}}
                </div>
            `;
        }});

        resticHtml += `
            <div style="margin-top:12px;text-align:center">
                <a href="/recovery" style="color:#667eea;text-decoration:none;font-size:0.9em">
                    🛟 Sauvegardes de configuration →
                </a>
            </div>
        `;

        document.getElementById('restic-status').innerHTML = resticHtml;

    }} catch (err) {{
        console.error('Erreur chargement Restic:', err);
        document.getElementById('restic-status').innerHTML =
            '<p style="color:#dc3545;text-align:center">⚠️ Erreur de chargement</p>';
    }}
}}

async function testConnection(peerName) {{
    const btn = event.target;
    const originalText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '⏳ Test en cours...';

    try {{
        const response = await fetch(`/api/restic/test-connection/${{peerName}}`, {{
            method: 'POST'
        }});
        const data = await response.json();

        // Afficher le diagnostic dans une modal ou alert
        let message = `📊 Diagnostic de ${{peerName}}\\n\\n`;
        message += `Statut global: ${{data.overall_message}}\\n\\n`;
        message += `Tests effectués:\\n`;

        if (data.tests.ping) {{
            message += `  🌐 Ping: ${{data.tests.ping.status}} - ${{data.tests.ping.message}}\\n`;
        }}
        if (data.tests.ssh) {{
            message += `  🔑 SSH: ${{data.tests.ssh.status}} - ${{data.tests.ssh.message}}\\n`;
            if (data.tests.ssh.fix) {{
                message += `     ➡️ Solution: ${{data.tests.ssh.fix}}\\n`;
            }}
        }}
        if (data.tests.sftp) {{
            message += `  📁 SFTP: ${{data.tests.sftp.status}} - ${{data.tests.sftp.message}}\\n`;
        }}
        if (data.tests.restic) {{
            message += `  📦 Restic: ${{data.tests.restic.status}} - ${{data.tests.restic.message}}\\n`;
            if (data.tests.restic.fix) {{
                message += `     ➡️ Solution: ${{data.tests.restic.fix}}\\n`;
            }}
        }}

        alert(message);

        // Rafraîchir le statut
        loadResticStatus();

    }} catch (err) {{
        alert(`❌ Erreur lors du test: ${{err.message}}`);
    }} finally {{
        btn.disabled = false;
        btn.innerHTML = originalText;
    }}
}}

async function initRepository(peerName) {{
    if (!confirm(`Voulez-vous initialiser le repository Restic sur ${{peerName}} ?\\n\\nCette opération peut prendre quelques secondes.`)) {{
        return;
    }}

    const btn = event.target;
    const originalText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '⏳ Initialisation...';

    try {{
        const response = await fetch(`/api/restic/init-repository/${{peerName}}`, {{
            method: 'POST'
        }});
        const data = await response.json();

        if (data.status === 'success' || data.status === 'already_initialized') {{
            alert(`✅ ${{data.message}}`);
            // Rafraîchir le statut
            loadResticStatus();
        }} else {{
            alert(`❌ Échec de l'initialisation:\\n${{data.message}}\\n\\nErreur: ${{data.error}}`);
        }}

    }} catch (err) {{
        alert(`❌ Erreur lors de l'initialisation: ${{err.message}}`);
    }} finally {{
        btn.disabled = false;
        btn.innerHTML = originalText;
    }}
}}

// Charger au démarrage
loadStats();
loadResticStatus();

// Rafraîchir toutes les 30 secondes
setInterval(loadStats, 30000);
setInterval(loadResticStatus, 60000); // Restic toutes les 60s (plus lent)
</script>
</body></html>"""
    return HTMLResponse(html)

@app.get("/health")
async def health():
    return {"status": "healthy", "setup_completed": is_setup_completed()}

@app.get("/api/status")
async def get_status():
    config = load_config()
    return {
        "node": {
            "name": config.get('node', {}).get('name', 'unknown'),
            "role": config.get('node', {}).get('role', 'unknown'),
        },
        "timestamp": datetime.now().isoformat(),
        "setup_completed": is_setup_completed()
    }

@app.get("/peers", response_class=HTMLResponse)
async def peers_page(request: Request):
    """Page de gestion des pairs"""
    t = get_all_texts(WEB_LANGUAGE)
    return templates.TemplateResponse("peers.html", {"request": request, "t": t})

# ===== API Gestion des pairs =====

class InvitationRequest(BaseModel):
    pin: Optional[str] = None
    pin_length: Optional[int] = 6

class AddPeerRequest(BaseModel):
    invitation_data: dict
    pin: Optional[str] = None

@app.post("/api/peers/generate-invitation")
async def generate_invitation(request: InvitationRequest):
    """Génère une invitation avec PIN optionnel"""
    try:
        pin = request.pin

        # Si aucun PIN fourni mais protection demandée, générer automatiquement
        if pin is None and request.pin_length:
            pin = generate_random_pin(request.pin_length)

        # Valider le PIN si fourni
        if pin and not validate_pin(pin):
            raise HTTPException(400, "PIN invalide (doit être 4-8 chiffres)")

        invitation = peer_manager.generate_invitation(pin)

        return {
            "success": True,
            "invitation": invitation,
            "pin": pin if pin else None,
            "protected": invitation.get("encrypted", False)
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur lors de la génération: {str(e)}")

@app.post("/api/peers/add")
async def add_peer(request: AddPeerRequest):
    """Ajoute un pair depuis une invitation"""
    try:
        # Log détaillé pour debug
        print(f"DEBUG: Received invitation_data: {request.invitation_data}", flush=True)
        print(f"DEBUG: PIN provided: {'Yes' if request.pin else 'No'}", flush=True)

        result = peer_manager.add_peer_from_invitation(
            request.invitation_data,
            request.pin
        )
        return {
            "success": True,
            "peer": result
        }
    except ValueError as e:
        print(f"ERROR (ValueError): {str(e)}", flush=True)
        raise HTTPException(400, str(e))
    except Exception as e:
        import traceback
        print(f"ERROR (Exception): {str(e)}", flush=True)
        print(f"Traceback: {traceback.format_exc()}", flush=True)
        raise HTTPException(500, f"Erreur lors de l'ajout: {str(e)}")

@app.get("/api/peers/list")
async def list_peers():
    """Liste tous les pairs configurés"""
    try:
        peers = peer_manager.list_peers()
        return {
            "success": True,
            "peers": peers,
            "count": len(peers)
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.delete("/api/peers/{peer_name}")
async def remove_peer(peer_name: str):
    """Supprime un pair"""
    try:
        success = peer_manager.remove_peer(peer_name)
        if not success:
            raise HTTPException(404, "Pair non trouvé")
        return {"success": True, "message": f"Pair {peer_name} supprimé"}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.get("/api/peers/{peer_name}/status")
async def get_peer_status(peer_name: str):
    """Récupère le statut d'un pair"""
    try:
        status = peer_manager.get_peer_status(peer_name)
        return status
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.get("/api/peers/local-info")
async def get_local_info():
    """Récupère les informations locales du serveur"""
    try:
        info = peer_manager.get_local_info()
        return {
            "success": True,
            "info": info
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.get("/api/peers/next-ip")
async def get_next_ip():
    """Récupère la prochaine IP VPN disponible"""
    try:
        next_ip = peer_manager.get_next_available_vpn_ip()
        used_ips = peer_manager.get_used_vpn_ips()
        return {
            "success": True,
            "next_ip": next_ip,
            "used_ips": used_ips
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

# ===== API Quotas Disque =====

@app.get("/api/quotas/check")
async def check_quotas():
    """Vérifie les quotas de tous les pairs"""
    try:
        quota_info = quota_manager.check_all_quotas()
        return {
            "success": True,
            "quotas": quota_info
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.get("/api/quotas/check/{peer_name}")
async def check_peer_quota(peer_name: str):
    """Vérifie le quota d'un pair spécifique"""
    try:
        within_quota, info = quota_manager.check_peer_quota(peer_name)
        return {
            "success": True,
            "quota": info
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.post("/api/quotas/enforce")
async def enforce_quotas():
    """Applique les quotas pour tous les pairs qui les dépassent"""
    try:
        quota_info = quota_manager.check_all_quotas()
        enforced = []

        for quota in quota_info:
            if not quota['within_quota'] and not quota['unlimited']:
                success = quota_manager.enforce_quota(quota['peer_name'])
                if success:
                    enforced.append(quota['peer_name'])

        return {
            "success": True,
            "enforced_peers": enforced,
            "message": f"{len(enforced)} pair(s) bloqué(s) pour dépassement de quota"
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.post("/api/quotas/restore/{peer_name}")
async def restore_peer_access(peer_name: str):
    """Restaure l'accès SSH pour un pair"""
    try:
        success = quota_manager.restore_access(peer_name)
        return {
            "success": success,
            "message": f"Accès restauré pour {peer_name}" if success else "Aucune modification nécessaire"
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.get("/api/quotas/config")
async def get_quota_config():
    """Récupère la configuration actuelle du quota"""
    try:
        with open(CONFIG_PATH, 'r') as f:
            config = yaml.safe_load(f)

        max_size = config.get('restic_server', {}).get('max_size_per_peer', '10GB')
        return {
            "success": True,
            "max_size_per_peer": max_size
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.post("/api/quotas/config")
async def update_quota_config(max_size_per_peer: str = Form(...)):
    """Met à jour la configuration du quota global"""
    try:
        # Valider la valeur
        valid_values = ["5GB", "10GB", "20GB", "50GB", "100GB", "200GB", "500GB", "0"]
        if max_size_per_peer not in valid_values:
            raise ValueError(f"Valeur invalide. Doit être l'une de: {', '.join(valid_values)}")

        # Charger la config actuelle
        with open(CONFIG_PATH, 'r') as f:
            config = yaml.safe_load(f)

        # Mettre à jour
        if 'restic_server' not in config:
            config['restic_server'] = {}

        config['restic_server']['max_size_per_peer'] = max_size_per_peer

        # Sauvegarder
        with open(CONFIG_PATH, 'w') as f:
            yaml.dump(config, f, default_flow_style=False)

        return {
            "success": True,
            "message": f"Quota mis à jour: {max_size_per_peer}",
            "max_size_per_peer": max_size_per_peer
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

# ===== API Statistiques Système =====

def get_directory_size(path: str) -> int:
    """Calcule la taille d'un répertoire en octets"""
    try:
        if not os.path.exists(path):
            return 0
        total = 0
        for dirpath, dirnames, filenames in os.walk(path):
            for filename in filenames:
                filepath = os.path.join(dirpath, filename)
                if os.path.exists(filepath):
                    total += os.path.getsize(filepath)
        return total
    except:
        return 0

def format_bytes(bytes_size: int) -> str:
    """Formate une taille en octets vers une unité lisible"""
    for unit in ['B', 'KB', 'MB', 'GB', 'TB']:
        if bytes_size < 1024.0:
            return f"{bytes_size:.2f} {unit}"
        bytes_size /= 1024.0
    return f"{bytes_size:.2f} PB"

@app.get("/api/system/stats")
async def get_system_stats():
    """Récupère les statistiques système"""
    try:
        # Taille des répertoires
        data_size = get_directory_size("/data")
        backup_size = get_directory_size("/backup")
        backups_size = get_directory_size("/backups")

        # Espace disque total
        disk_usage = shutil.disk_usage("/")

        # Statut VPN
        vpn_status = "unknown"
        try:
            result = subprocess.run(
                ["docker", "exec", "anemone-core", "wg", "show"],
                capture_output=True,
                timeout=5
            )
            vpn_status = "up" if result.returncode == 0 else "down"
        except:
            vpn_status = "down"

        # Nombre de pairs connectés
        peers_connected = 0
        try:
            peers = peer_manager.list_peers()
            for peer in peers:
                status = peer_manager.get_peer_status(peer["name"])
                if status.get("status") == "connected":
                    peers_connected += 1
        except:
            pass

        return {
            "success": True,
            "storage": {
                "data": {
                    "bytes": data_size,
                    "formatted": format_bytes(data_size)
                },
                "backup": {
                    "bytes": backup_size,
                    "formatted": format_bytes(backup_size)
                },
                "backups": {
                    "bytes": backups_size,
                    "formatted": format_bytes(backups_size)
                },
                "total": {
                    "bytes": data_size + backup_size + backups_size,
                    "formatted": format_bytes(data_size + backup_size + backups_size)
                }
            },
            "disk": {
                "total": {
                    "bytes": disk_usage.total,
                    "formatted": format_bytes(disk_usage.total)
                },
                "used": {
                    "bytes": disk_usage.used,
                    "formatted": format_bytes(disk_usage.used)
                },
                "free": {
                    "bytes": disk_usage.free,
                    "formatted": format_bytes(disk_usage.free)
                },
                "percent": round((disk_usage.used / disk_usage.total) * 100, 1)
            },
            "vpn": {
                "status": vpn_status,
                "peers_connected": peers_connected
            }
        }
    except Exception as e:
        raise HTTPException(500, f"Erreur: {str(e)}")

@app.get("/api/vpn/status")
async def get_vpn_status():
    """Récupère le statut détaillé du VPN WireGuard"""
    try:
        result = subprocess.run(
            ["docker", "exec", "anemone-core", "wg", "show"],
            capture_output=True,
            text=True,
            timeout=5
        )

        if result.returncode != 0:
            return {
                "status": "down",
                "message": "VPN non actif"
            }

        # Parser la sortie de wg show
        output = result.stdout
        peers_info = []
        current_peer = None

        for line in output.split('\n'):
            line = line.strip()
            if line.startswith('peer:'):
                if current_peer:
                    peers_info.append(current_peer)
                current_peer = {"public_key": line.split(':')[1].strip()}
            elif current_peer and ':' in line:
                key, value = line.split(':', 1)
                key = key.strip()
                value = value.strip()
                if key == 'endpoint':
                    current_peer['endpoint'] = value
                elif key == 'latest handshake':
                    current_peer['latest_handshake'] = value
                elif key == 'transfer':
                    current_peer['transfer'] = value

        if current_peer:
            peers_info.append(current_peer)

        return {
            "status": "up",
            "peers": peers_info,
            "peers_count": len(peers_info)
        }
    except Exception as e:
        return {
            "status": "error",
            "message": str(e)
        }

@app.post("/api/vpn/restart")
async def restart_vpn():
    """Redémarre WireGuard puis Restic pour reconnecter au VPN

    Nécessaire après ajout de peers ou modification de configuration WireGuard
    car Restic utilise network_mode: "service:wireguard" et doit se reconnecter
    au nouveau namespace réseau après un restart de WireGuard.
    """
    try:
        import time

        # Étape 1 : Redémarrer Core (WireGuard)
        result_wg = subprocess.run(
            ["docker", "restart", "anemone-core"],
            capture_output=True,
            text=True,
            timeout=30
        )

        if result_wg.returncode != 0:
            return {
                "success": False,
                "message": f"Erreur lors du redémarrage de WireGuard: {result_wg.stderr}"
            }

        # Étape 2 : Attendre 5 secondes que Core soit complètement prêt
        time.sleep(5)

        # Étape 3 : Note - Plus besoin de redémarrer Restic séparément car maintenant dans le même conteneur
        # Dans la v2.0, WireGuard, SFTP et Restic sont tous dans anemone-core
        # Le redémarrage du conteneur core suffit
        result_restic = subprocess.run(
            ["docker", "exec", "anemone-core", "supervisorctl", "-c", "/etc/supervisord.conf", "restart", "restic"],
            capture_output=True,
            text=True,
            timeout=30
        )

        if result_restic.returncode != 0:
            return {
                "success": False,
                "message": f"WireGuard redémarré, mais erreur lors du redémarrage de Restic: {result_restic.stderr}"
            }

        return {
            "success": True,
            "message": "VPN redémarré avec succès. WireGuard et Restic sont reconnectés."
        }

    except subprocess.TimeoutExpired:
        return {
            "success": False,
            "message": "Timeout lors du redémarrage (>30s)"
        }
    except Exception as e:
        return {
            "success": False,
            "message": f"Erreur inattendue: {str(e)}"
        }

@app.get("/api/config/export")
async def export_configuration():
    """
    Exporte la configuration complète du serveur dans un fichier chiffré

    Inclut :
    - config.yaml (configuration complète)
    - Clés WireGuard (VPN)
    - Clés SSH
    - Clé Restic chiffrée + salt

    Le fichier est chiffré avec la clé Restic pour sécurité maximale
    """
    import tarfile
    import tempfile
    import io

    try:
        # Vérifier que le setup est complété
        if not Path("/config/.setup-completed").exists():
            raise HTTPException(status_code=400, detail="Setup not completed")

        # Lire la clé Restic déchiffrée pour chiffrer l'export
        restic_key = subprocess.run(
            ["python3", "/scripts/decrypt_key.py"],
            capture_output=True,
            text=True,
            check=True
        ).stdout.strip()

        if not restic_key:
            raise HTTPException(status_code=500, detail="Failed to decrypt Restic key")

        # Créer une archive tar.gz en mémoire
        tar_buffer = io.BytesIO()

        with tarfile.open(fileobj=tar_buffer, mode='w:gz') as tar:
            # Ajouter config.yaml
            if Path("/config/config.yaml").exists():
                tar.add("/config/config.yaml", arcname="config.yaml")

            # Ajouter les clés WireGuard
            wg_dir = Path("/config/wireguard")
            if wg_dir.exists():
                for key_file in ["private.key", "public.key"]:
                    key_path = wg_dir / key_file
                    if key_path.exists():
                        tar.add(str(key_path), arcname=f"wireguard/{key_file}")

            # Ajouter les clés SSH
            ssh_dir = Path("/config/ssh")
            if ssh_dir.exists():
                for ssh_file in ["id_rsa", "id_rsa.pub", "authorized_keys"]:
                    ssh_path = ssh_dir / ssh_file
                    if ssh_path.exists():
                        tar.add(str(ssh_path), arcname=f"ssh/{ssh_file}")

            # Ajouter la clé Restic chiffrée et le salt
            if Path("/config/.restic.encrypted").exists():
                tar.add("/config/.restic.encrypted", arcname=".restic.encrypted")
            if Path("/config/.restic.salt").exists():
                tar.add("/config/.restic.salt", arcname=".restic.salt")

        # Récupérer le contenu de l'archive
        tar_buffer.seek(0)
        tar_data = tar_buffer.read()

        # Chiffrer l'archive avec la clé Restic
        # Générer un IV aléatoire
        iv = secrets.token_bytes(16)

        # Dériver une clé de chiffrement depuis la clé Restic
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=b"anemone-config-export",  # Salt fixe pour l'export
            iterations=100000,
            backend=default_backend()
        )
        encryption_key = kdf.derive(restic_key.encode())

        # Chiffrer avec AES-256-CBC
        cipher = Cipher(
            algorithms.AES(encryption_key),
            modes.CBC(iv),
            backend=default_backend()
        )
        encryptor = cipher.encryptor()

        # Padding PKCS7
        block_size = 16
        padding_length = block_size - (len(tar_data) % block_size)
        padded_data = tar_data + bytes([padding_length] * padding_length)

        # Chiffrer
        encrypted_data = encryptor.update(padded_data) + encryptor.finalize()

        # Combiner IV + données chiffrées
        final_data = iv + encrypted_data

        # Générer le nom de fichier avec timestamp
        timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")

        # Lire le nom du nœud depuis config
        node_name = "anemone"
        try:
            with open("/config/config.yaml") as f:
                config = yaml.safe_load(f)
                node_name = config.get("node", {}).get("name", "anemone")
        except:
            pass

        filename = f"anemone-backup-{node_name}-{timestamp}.enc"

        # Retourner le fichier chiffré
        return Response(
            content=final_data,
            media_type="application/octet-stream",
            headers={
                "Content-Disposition": f'attachment; filename="{filename}"',
                "X-Backup-Timestamp": timestamp,
                "X-Backup-Node": node_name
            }
        )

    except subprocess.CalledProcessError as e:
        raise HTTPException(status_code=500, detail=f"Command error: {e.stderr}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Export failed: {str(e)}")


# ===== PHASE 3 : DISASTER RECOVERY AVANCÉ =====

@app.get("/api/recovery/backups")
async def list_recovery_backups():
    """
    Liste tous les backups disponibles (locaux + peers) avec métadonnées
    Phase 3 : Interface web de recovery
    """
    try:
        import json

        backups = {
            "local": [],
            "peers": {},
            "total": 0,
            "metadata": {
                "scanned_at": datetime.now().isoformat(),
                "node_name": "anemone"
            }
        }

        # Lire le nom du serveur
        try:
            with open("/config/config.yaml") as f:
                config = yaml.safe_load(f)
                backups["metadata"]["node_name"] = config.get("server", {}).get("name", "anemone")
        except:
            pass

        # Backups locaux
        local_dir = Path("/config-backups/local")
        if local_dir.exists():
            for backup_file in sorted(local_dir.glob("*.enc"), key=lambda x: x.stat().st_mtime, reverse=True):
                stat = backup_file.stat()
                backups["local"].append({
                    "filename": backup_file.name,
                    "path": str(backup_file),
                    "size": stat.st_size,
                    "size_mb": round(stat.st_size / (1024 * 1024), 2),
                    "mtime": stat.st_mtime,
                    "mtime_iso": datetime.fromtimestamp(stat.st_mtime).isoformat(),
                    "location": "local"
                })

        # Backups des peers (stockés localement)
        config_backups_dir = Path("/config-backups")
        if config_backups_dir.exists():
            for peer_dir in config_backups_dir.iterdir():
                if peer_dir.is_dir() and peer_dir.name != "local":
                    peer_name = peer_dir.name
                    backups["peers"][peer_name] = []

                    for backup_file in sorted(peer_dir.glob("*.enc"), key=lambda x: x.stat().st_mtime, reverse=True):
                        stat = backup_file.stat()
                        backups["peers"][peer_name].append({
                            "filename": backup_file.name,
                            "path": str(backup_file),
                            "size": stat.st_size,
                            "size_mb": round(stat.st_size / (1024 * 1024), 2),
                            "mtime": stat.st_mtime,
                            "mtime_iso": datetime.fromtimestamp(stat.st_mtime).isoformat(),
                            "location": f"peer:{peer_name}"
                        })

        # Compter le total
        backups["total"] = len(backups["local"]) + sum(len(peer_backups) for peer_backups in backups["peers"].values())

        # Découvrir les backups sur les peers distants (via discover-backups.py)
        try:
            result = subprocess.run(
                ["python3", "/scripts/discover-backups.py", "--json"],
                capture_output=True,
                text=True,
                timeout=30,
                check=False
            )

            if result.returncode == 0 and result.stdout.strip():
                remote_data = json.loads(result.stdout)
                backups["remote"] = remote_data.get("backups", [])
                backups["total"] += len(backups.get("remote", []))
        except Exception as e:
            print(f"Warning: Could not discover remote backups: {e}")
            backups["remote"] = []

        return JSONResponse(content=backups)

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to list backups: {str(e)}")


@app.post("/api/recovery/verify")
async def verify_backup_integrity(backup_path: str = Body(..., embed=True)):
    """
    Vérifie l'intégrité d'un fichier de backup
    Phase 3 : Vérification d'intégrité
    """
    try:
        backup_file = Path(backup_path)

        if not backup_file.exists():
            raise HTTPException(status_code=404, detail="Backup file not found")

        # Vérifications de base
        checks = {
            "exists": backup_file.exists(),
            "readable": os.access(backup_file, os.R_OK),
            "size_valid": backup_file.stat().st_size > 0,
            "is_file": backup_file.is_file(),
            "extension": backup_file.suffix == ".enc"
        }

        # Vérifier que le fichier peut être lu
        try:
            with open(backup_file, 'rb') as f:
                # Lire les premiers bytes (IV)
                iv = f.read(16)
                checks["has_iv"] = len(iv) == 16

                # Vérifier qu'il y a des données après l'IV
                first_block = f.read(16)
                checks["has_data"] = len(first_block) > 0
        except Exception as e:
            checks["read_error"] = str(e)

        # Score d'intégrité
        passed_checks = sum(1 for v in checks.values() if v is True)
        total_checks = len([v for v in checks.values() if isinstance(v, bool)])
        integrity_score = (passed_checks / total_checks * 100) if total_checks > 0 else 0

        return JSONResponse(content={
            "backup_path": str(backup_file),
            "checks": checks,
            "integrity_score": round(integrity_score, 2),
            "status": "valid" if integrity_score == 100 else "warning" if integrity_score > 50 else "invalid",
            "verified_at": datetime.now().isoformat()
        })

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Verification failed: {str(e)}")


@app.get("/api/recovery/history")
async def get_backup_history(days: int = 30):
    """
    Retourne l'historique des backups avec métadonnées détaillées
    Phase 3 : Historique multi-versions
    """
    try:
        history = {
            "backups": [],
            "stats": {
                "total_backups": 0,
                "total_size_mb": 0,
                "oldest_backup": None,
                "newest_backup": None,
                "locations": {}
            },
            "period_days": days,
            "generated_at": datetime.now().isoformat()
        }

        cutoff_time = datetime.now().timestamp() - (days * 24 * 60 * 60)

        # Scanner tous les backups (locaux et peers)
        all_backups = []

        # Backups locaux
        local_dir = Path("/config-backups/local")
        if local_dir.exists():
            for backup_file in local_dir.glob("*.enc"):
                stat = backup_file.stat()
                if stat.st_mtime >= cutoff_time:
                    all_backups.append({
                        "filename": backup_file.name,
                        "path": str(backup_file),
                        "size": stat.st_size,
                        "size_mb": round(stat.st_size / (1024 * 1024), 2),
                        "timestamp": stat.st_mtime,
                        "timestamp_iso": datetime.fromtimestamp(stat.st_mtime).isoformat(),
                        "location": "local",
                        "type": "local"
                    })

        # Backups des peers
        config_backups_dir = Path("/config-backups")
        if config_backups_dir.exists():
            for peer_dir in config_backups_dir.iterdir():
                if peer_dir.is_dir() and peer_dir.name != "local":
                    for backup_file in peer_dir.glob("*.enc"):
                        stat = backup_file.stat()
                        if stat.st_mtime >= cutoff_time:
                            all_backups.append({
                                "filename": backup_file.name,
                                "path": str(backup_file),
                                "size": stat.st_size,
                                "size_mb": round(stat.st_size / (1024 * 1024), 2),
                                "timestamp": stat.st_mtime,
                                "timestamp_iso": datetime.fromtimestamp(stat.st_mtime).isoformat(),
                                "location": f"peer:{peer_dir.name}",
                                "type": "peer",
                                "peer_name": peer_dir.name
                            })

        # Trier par date (plus récent d'abord)
        all_backups.sort(key=lambda x: x["timestamp"], reverse=True)

        # Statistiques
        if all_backups:
            history["stats"]["total_backups"] = len(all_backups)
            history["stats"]["total_size_mb"] = round(sum(b["size_mb"] for b in all_backups), 2)
            history["stats"]["oldest_backup"] = all_backups[-1]["timestamp_iso"]
            history["stats"]["newest_backup"] = all_backups[0]["timestamp_iso"]

            # Compter par location
            for backup in all_backups:
                loc = backup["location"]
                history["stats"]["locations"][loc] = history["stats"]["locations"].get(loc, 0) + 1

        history["backups"] = all_backups

        return JSONResponse(content=history)

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to get history: {str(e)}")


@app.get("/recovery", response_class=HTMLResponse)
async def recovery_page(request: Request):
    """
    Interface web graphique pour la gestion du disaster recovery
    Phase 3 : Interface web de recovery
    """
    return templates.TemplateResponse("recovery.html", {
        "request": request,
        "title": "Disaster Recovery - Anemone",
        "language": WEB_LANGUAGE
    })


@app.post("/api/recovery/test-notification")
async def test_notification(
    notification_type: str = Body(...),
    config: dict = Body(...)
):
    """
    Teste une configuration de notification (email ou webhook)
    Phase 3 : Notifications optionnelles
    """
    try:
        if notification_type == "email":
            # Test email (optionnel)
            smtp_server = config.get("smtp_server")
            smtp_port = config.get("smtp_port", 587)
            smtp_user = config.get("smtp_user")
            smtp_password = config.get("smtp_password")
            to_email = config.get("to_email")

            if not all([smtp_server, smtp_user, smtp_password, to_email]):
                raise HTTPException(status_code=400, detail="Missing email configuration")

            import smtplib
            from email.mime.text import MIMEText

            msg = MIMEText("This is a test notification from Anemone Disaster Recovery system.")
            msg["Subject"] = "Anemone - Test Notification"
            msg["From"] = smtp_user
            msg["To"] = to_email

            with smtplib.SMTP(smtp_server, smtp_port) as server:
                server.starttls()
                server.login(smtp_user, smtp_password)
                server.send_message(msg)

            return JSONResponse(content={"status": "success", "message": "Email sent successfully"})

        elif notification_type == "webhook":
            # Test webhook (optionnel)
            webhook_url = config.get("webhook_url")

            if not webhook_url:
                raise HTTPException(status_code=400, detail="Missing webhook URL")

            import requests

            payload = {
                "event": "test_notification",
                "timestamp": datetime.now().isoformat(),
                "message": "Test notification from Anemone Disaster Recovery system"
            }

            response = requests.post(webhook_url, json=payload, timeout=10)
            response.raise_for_status()

            return JSONResponse(content={
                "status": "success",
                "message": "Webhook delivered successfully",
                "status_code": response.status_code
            })

        else:
            raise HTTPException(status_code=400, detail="Invalid notification type")

    except HTTPException:
        raise
    except Exception as e:
        return JSONResponse(
            content={"status": "error", "message": str(e)},
            status_code=500
        )


@app.post("/api/recovery/force-backup")
async def force_config_backup():
    """
    Force une sauvegarde immédiate de la configuration
    Appelle le script backup-config.sh dans le conteneur core
    """
    try:
        result = subprocess.run(
            ["docker", "exec", "anemone-core", "/scripts/backup-config.sh"],
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode == 0:
            return JSONResponse(content={
                "status": "success",
                "message": "Sauvegarde de configuration créée avec succès",
                "output": result.stdout
            })
        else:
            return JSONResponse(
                content={
                    "status": "error",
                    "message": "Échec de la sauvegarde",
                    "error": result.stderr
                },
                status_code=500
            )

    except subprocess.TimeoutExpired:
        raise HTTPException(status_code=504, detail="Timeout lors de la sauvegarde")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Erreur lors de la sauvegarde : {str(e)}")


@app.post("/api/restic/test-connection/{peer_name}")
async def test_restic_connection(peer_name: str):
    """
    Teste la connexion SSH et Restic avec un peer
    Retourne un diagnostic détaillé des problèmes éventuels
    """
    try:
        # Charger la configuration
        with open(CONFIG_PATH) as f:
            config = yaml.safe_load(f)

        server_name = config.get('server', {}).get('name', 'unknown')
        peers = config.get('peers', [])

        # Trouver le peer
        peer = next((p for p in peers if p.get('name') == peer_name), None)
        if not peer:
            raise HTTPException(status_code=404, detail=f"Peer '{peer_name}' not found")

        peer_ip = peer.get('allowed_ips', '').split('/')[0]
        if not peer_ip:
            raise HTTPException(status_code=400, detail="Peer has no IP configured")

        repo_url = f"sftp:restic@{peer_ip}:/backups/{server_name}"

        results = {
            "peer_name": peer_name,
            "peer_ip": peer_ip,
            "repository_url": repo_url,
            "tests": {}
        }

        # Test 1 : Ping
        try:
            ping_result = subprocess.run(
                ["docker", "exec", "anemone-core", "ping", "-c", "2", "-W", "2", peer_ip],
                capture_output=True,
                text=True,
                timeout=5
            )
            results["tests"]["ping"] = {
                "status": "ok" if ping_result.returncode == 0 else "error",
                "message": "Peer is reachable" if ping_result.returncode == 0 else "Peer is unreachable"
            }
        except Exception as e:
            results["tests"]["ping"] = {
                "status": "error",
                "message": f"Ping failed: {str(e)}"
            }

        # Test 2 : SSH Connection
        try:
            ssh_result = subprocess.run(
                ["docker", "exec", "anemone-core", "ssh", "-o", "StrictHostKeyChecking=no",
                 "-o", "ConnectTimeout=5", f"restic@{peer_ip}", "echo OK"],
                capture_output=True,
                text=True,
                timeout=10
            )

            error_msg = ssh_result.stderr if ssh_result.stderr else ssh_result.stdout

            # Succès si on obtient "OK" OU si le serveur répond qu'il n'accepte que SFTP
            # "This service allows sftp connections only" = SSH fonctionne, juste restreint à SFTP (BIEN pour sécurité)
            if ssh_result.returncode == 0 and "OK" in ssh_result.stdout:
                results["tests"]["ssh"] = {
                    "status": "ok",
                    "message": "SSH connection successful"
                }
            elif "sftp connections only" in error_msg.lower() or "sftp connections only" in ssh_result.stdout.lower():
                results["tests"]["ssh"] = {
                    "status": "ok",
                    "message": "SSH connection OK (SFTP-only mode, secure)"
                }
            else:
                if "Permission denied" in error_msg:
                    results["tests"]["ssh"] = {
                        "status": "error",
                        "message": "SSH key not authorized on peer",
                        "fix": f"Add this server's SSH public key to {peer_name}'s authorized_keys"
                    }
                else:
                    results["tests"]["ssh"] = {
                        "status": "error",
                        "message": f"SSH connection failed: {error_msg}"
                    }
        except Exception as e:
            results["tests"]["ssh"] = {
                "status": "error",
                "message": f"SSH test error: {str(e)}"
            }

        # Test 3 : SFTP Access
        if results["tests"].get("ssh", {}).get("status") == "ok":
            try:
                sftp_result = subprocess.run(
                    ["docker", "exec", "anemone-core", "sh", "-c",
                     f"echo 'ls /backups' | sftp -o StrictHostKeyChecking=no restic@{peer_ip}"],
                    capture_output=True,
                    text=True,
                    timeout=10
                )

                if sftp_result.returncode == 0:
                    results["tests"]["sftp"] = {
                        "status": "ok",
                        "message": "SFTP access successful"
                    }
                else:
                    results["tests"]["sftp"] = {
                        "status": "error",
                        "message": f"SFTP access failed: {sftp_result.stderr}"
                    }
            except Exception as e:
                results["tests"]["sftp"] = {
                    "status": "error",
                    "message": f"SFTP test error: {str(e)}"
                }

        # Test 4 : Restic Repository
        if results["tests"].get("ssh", {}).get("status") == "ok":
            try:
                restic_result = subprocess.run(
                    ["docker", "exec", "anemone-core", "sh", "-c",
                     f"export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py 2>/dev/null) && "
                     f"restic -r {repo_url} snapshots --json --latest 1 2>&1"],
                    capture_output=True,
                    text=True,
                    timeout=15
                )

                if restic_result.returncode == 0 and restic_result.stdout.strip():
                    results["tests"]["restic"] = {
                        "status": "ok",
                        "message": "Restic repository accessible"
                    }
                elif "does not exist" in restic_result.stdout or "does not exist" in restic_result.stderr:
                    results["tests"]["restic"] = {
                        "status": "not_initialized",
                        "message": "Repository not initialized",
                        "fix": "Click 'Initialize Repository' button"
                    }
                else:
                    results["tests"]["restic"] = {
                        "status": "error",
                        "message": f"Repository access failed: {restic_result.stderr or restic_result.stdout}"
                    }
            except Exception as e:
                results["tests"]["restic"] = {
                    "status": "error",
                    "message": f"Restic test error: {str(e)}"
                }

        # Déterminer le statut global
        all_tests = list(results["tests"].values())
        if all(t["status"] == "ok" for t in all_tests):
            results["overall_status"] = "ok"
            results["overall_message"] = "All tests passed. Backup is ready!"
        elif any(t["status"] == "not_initialized" for t in all_tests):
            results["overall_status"] = "not_initialized"
            results["overall_message"] = "Connection OK but repository needs initialization"
        else:
            results["overall_status"] = "error"
            results["overall_message"] = "Some tests failed. Check details below."

        return JSONResponse(content=results)

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Test failed: {str(e)}")


@app.post("/api/restic/init-repository/{peer_name}")
async def init_restic_repository(peer_name: str):
    """
    Initialise le repository Restic sur un peer
    """
    try:
        # Charger la configuration
        with open(CONFIG_PATH) as f:
            config = yaml.safe_load(f)

        server_name = config.get('server', {}).get('name', 'unknown')
        peers = config.get('peers', [])

        # Trouver le peer
        peer = next((p for p in peers if p.get('name') == peer_name), None)
        if not peer:
            raise HTTPException(status_code=404, detail=f"Peer '{peer_name}' not found")

        peer_ip = peer.get('allowed_ips', '').split('/')[0]
        if not peer_ip:
            raise HTTPException(status_code=400, detail="Peer has no IP configured")

        repo_url = f"sftp:restic@{peer_ip}:/backups/{server_name}"

        # Initialiser le repository
        result = subprocess.run(
            ["docker", "exec", "anemone-core", "sh", "-c",
             f"export RESTIC_PASSWORD=$(python3 /scripts/decrypt_key.py 2>/dev/null) && "
             f"restic -r {repo_url} init 2>&1"],
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode == 0:
            return JSONResponse(content={
                "status": "success",
                "message": f"Repository initialized successfully on {peer_name}",
                "repository_url": repo_url,
                "output": result.stdout
            })
        else:
            # Vérifier si c'est juste déjà initialisé
            if "already initialized" in result.stderr or "already exists" in result.stderr:
                return JSONResponse(content={
                    "status": "already_initialized",
                    "message": f"Repository already initialized on {peer_name}",
                    "repository_url": repo_url
                })
            else:
                return JSONResponse(
                    content={
                        "status": "error",
                        "message": "Failed to initialize repository",
                        "error": result.stderr or result.stdout
                    },
                    status_code=500
                )

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Initialization failed: {str(e)}")


@app.get("/api/restic/status")
async def get_restic_status():
    """
    Récupère l'état des snapshots Restic sur tous les targets de backup
    Lit le fichier JSON généré par le service core après chaque backup
    """
    try:
        stats_file = Path("/var/stats/restic-status.json")

        # Vérifier si le fichier de stats existe
        if not stats_file.exists():
            # Si pas encore de stats, retourner un statut par défaut
            with open(CONFIG_PATH) as f:
                config = yaml.safe_load(f)

            targets = config.get('backup', {}).get('targets', [])
            active_targets = [t for t in targets if t.get('enabled', True)]

            return JSONResponse(content={
                "global_status": "waiting",
                "message": "Waiting for first backup to complete",
                "server_name": config.get('node', {}).get('name', 'unknown'),
                "targets": [],
                "total_targets": len(active_targets),
                "checked_at": datetime.now().isoformat()
            })

        # Lire le fichier de stats
        with open(stats_file) as f:
            stats = json.load(f)

        # Ajouter des métadonnées
        stats["total_targets"] = len(stats.get("targets", []))
        stats["checked_at"] = datetime.now().isoformat()

        return JSONResponse(content=stats)

    except json.JSONDecodeError:
        raise HTTPException(status_code=500, detail="Invalid stats file format")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to get Restic status: {str(e)}")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000)
