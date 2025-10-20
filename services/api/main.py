#!/usr/bin/env python3
"""
Anemone API - Interface de monitoring et gestion avec setup sécurisé
"""

import os
import yaml
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

// Charger au démarrage
loadStats();

// Rafraîchir toutes les 30 secondes
setInterval(loadStats, 30000);
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

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000)
