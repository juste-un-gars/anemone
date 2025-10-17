#!/usr/bin/env python3
"""
Anemone API - Interface de monitoring et gestion avec setup sécurisé
"""

import os
import yaml
import subprocess
import secrets
from datetime import datetime
from pathlib import Path
from typing import Dict
from fastapi import FastAPI, HTTPException, Form, Request
from fastapi.responses import HTMLResponse, RedirectResponse
from pydantic import BaseModel
from starlette.middleware.base import BaseHTTPMiddleware
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# Configuration
CONFIG_PATH = os.getenv('CONFIG_PATH', '/config/config.yaml')
SETUP_COMPLETED = Path('/config/.setup-completed')
RESTIC_ENCRYPTED = Path('/config/.restic.encrypted')
RESTIC_SALT = Path('/config/.restic.salt')

app = FastAPI(title="Anemone API", version="1.0.0")

# ===== Utilitaires =====

def is_setup_completed() -> bool:
    return SETUP_COMPLETED.exists()

def get_system_key() -> str:
    # IMPORTANT : Utiliser le HOSTNAME (fixe et persistant) au lieu de UUID
    # L'UUID change à chaque redémarrage du conteneur, rendant le déchiffrement impossible
    return os.getenv('HOSTNAME', 'anemone')

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

class SetupMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        path = request.url.path
        
        if not is_setup_completed() and not path.startswith('/setup'):
            return RedirectResponse('/setup', status_code=302)
        
        if is_setup_completed() and path.startswith('/setup'):
            return RedirectResponse('/', status_code=302)
        
        return await call_next(request)

app.add_middleware(SetupMiddleware)

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
<button onclick="navigator.clipboard.writeText(document.getElementById('key').textContent);alert('Copié!')">📋 Copier</button>
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
    html = f"""<!DOCTYPE html>
<html><head><title>🪸 {name}</title><meta charset="utf-8">
<style>
body {{ font-family: -apple-system, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        min-height: 100vh; padding: 20px; }}
.container {{ max-width: 1200px; margin: 0 auto; }}
.header {{ text-align: center; color: white; margin-bottom: 40px; }}
.card {{ background: white; border-radius: 12px; padding: 24px; margin-bottom: 20px; }}
a {{ color: #667eea; text-decoration: none; }}
</style></head><body>
<div class="container">
<div class="header"><h1>🪸 {name}</h1><p>Serveur de fichiers distribué</p></div>
<div class="card">
<h2>📊 État du système</h2>
<ul style="line-height:2">
<li><a href="/api/status">État général</a></li>
<li><a href="/docs">Documentation API</a></li>
</ul></div></div></body></html>"""
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

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000)
