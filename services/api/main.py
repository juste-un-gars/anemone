#!/usr/bin/env python3
"""
Anemone API - Interface de monitoring et gestion avec setup sécurisé
"""

import os
import yaml
import docker
import psutil
import subprocess
import secrets
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional
from fastapi import FastAPI, HTTPException, Form, Request, UploadFile, File
from fastapi.responses import HTMLResponse, RedirectResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from starlette.middleware.base import BaseHTTPMiddleware

# Configuration
CONFIG_PATH = os.getenv('CONFIG_PATH', '/config/config.yaml')
LOG_PATH = os.getenv('LOG_PATH', '/logs')
SETUP_COMPLETED = Path('/config/.setup-completed')
RESTIC_ENCRYPTED = Path('/config/.restic.encrypted')
RESTIC_SALT = Path('/config/.restic.salt')
RESTIC_PASSWORD_FILE = Path('/config/restic-password')

app = FastAPI(
    title="Anemone API",
    description="API de monitoring pour Anemone",
    version="1.0.0"
)

# Client Docker
try:
    docker_client = docker.from_env()
except:
    docker_client = None

# ===== Utilitaires de sécurité =====

def is_setup_completed() -> bool:
    """Vérifier si le setup initial a été complété"""
    return SETUP_COMPLETED.exists()

def get_system_key() -> str:
    """Obtenir une clé unique au système"""
    try:
        with open('/proc/sys/kernel/random/uuid') as f:
            return f.read().strip()
    except:
        return os.getenv('HOSTNAME', 'anemone') + '-' + str(os.getpid())

def generate_restic_key() -> str:
    """Générer une clé Restic aléatoire sécurisée"""
    return secrets.token_urlsafe(32)

def encrypt_restic_key(key: str) -> bool:
    """Chiffrer et stocker la clé Restic de manière sécurisée"""
    try:
        system_key = get_system_key()
        salt = secrets.token_hex(32)
        
        process = subprocess.Popen(
            [
                'openssl', 'enc', '-aes-256-cbc',
                '-pbkdf2', '-iter', '100000',
                '-pass', f'pass:{system_key}:{salt}',
                '-out', str(RESTIC_ENCRYPTED)
            ],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        stdout, stderr = process.communicate(input=key.encode())
        
        if process.returncode != 0:
            print(f"Encryption error: {stderr.decode()}")
            return False
        
        RESTIC_SALT.write_text(salt)
        SETUP_COMPLETED.touch()
        
        if RESTIC_PASSWORD_FILE.exists():
            try:
                subprocess.run(['shred', '-u', str(RESTIC_PASSWORD_FILE)], 
                             capture_output=True, timeout=5)
            except:
                RESTIC_PASSWORD_FILE.unlink()
        
        return True
        
    except Exception as e:
        print(f"Error encrypting key: {e}")
        return False

def validate_restic_key(key: str) -> bool:
    """Valider le format de la clé Restic"""
    return len(key) >= 20 and all(c.isalnum() or c in '-_+/=' for c in key)

# ===== Middleware de redirection setup =====

class SetupMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        path = request.url.path
        
        if not is_setup_completed() and not path.startswith('/setup'):
            return RedirectResponse('/setup', status_code=302)
        
        if is_setup_completed() and path.startswith('/setup'):
            return RedirectResponse('/', status_code=302)
        
        return await call_next(request)

app.add_middleware(SetupMiddleware)

# ===== Routes de Setup =====

@app.get("/setup", response_class=HTMLResponse)
async def setup_page():
    """Page de choix du mode de setup"""
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 Anemone - Configuration initiale</title>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body {
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                display: flex;
                align-items: center;
                justify-content: center;
                padding: 20px;
            }
            .container {
                background: white;
                border-radius: 16px;
                padding: 40px;
                max-width: 600px;
                width: 100%;
                box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            }
            h1 { color: #333; margin-bottom: 10px; font-size: 2em; }
            .subtitle { color: #666; margin-bottom: 40px; }
            .option {
                border: 2px solid #e0e0e0;
                border-radius: 12px;
                padding: 24px;
                margin-bottom: 16px;
                cursor: pointer;
                transition: all 0.3s;
            }
            .option:hover { border-color: #667eea; background: #f8f9ff; }
            .option.selected { border-color: #667eea; background: #f0f3ff; }
            .option input[type="radio"] { display: none; }
            .option h3 { color: #333; margin-bottom: 8px; }
            .option p { color: #666; font-size: 0.9em; }
            button {
                width: 100%;
                padding: 16px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                border: none;
                border-radius: 8px;
                font-size: 1.1em;
                font-weight: 600;
                cursor: pointer;
                margin-top: 20px;
                transition: transform 0.2s;
            }
            button:hover { transform: translateY(-2px); }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>🪸 Bienvenue sur Anemone</h1>
            <p class="subtitle">Configuration initiale de votre serveur</p>
            
            <div class="option" onclick="selectOption('new')">
                <input type="radio" name="mode" value="new" id="mode-new" checked>
                <h3>🆕 Nouveau serveur</h3>
                <p>Générer une nouvelle clé de chiffrement</p>
            </div>
            
            <div class="option" onclick="selectOption('restore')">
                <input type="radio" name="mode" value="restore" id="mode-restore">
                <h3>♻️ Restauration</h3>
                <p>J'ai déjà une clé de chiffrement (récupération après incident)</p>
            </div>
            
            <button onclick="nextStep()">Continuer</button>
        </div>
        
        <script>
            function selectOption(mode) {
                document.querySelectorAll('.option').forEach(el => {
                    el.classList.remove('selected');
                });
                event.currentTarget.classList.add('selected');
                document.getElementById('mode-' + mode).checked = true;
            }
            
            function nextStep() {
                const mode = document.querySelector('input[name="mode"]:checked').value;
                window.location = '/setup/' + mode;
            }
            
            selectOption('new');
        </script>
    </body>
    </html>
    """)

@app.get("/setup/new", response_class=HTMLResponse)
async def setup_new():
    """Générer une nouvelle clé"""
    key = generate_restic_key()
    
    return HTMLResponse(f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 Nouvelle clé générée</title>
        <meta charset="utf-8">
        <style>
            * {{ margin: 0; padding: 0; box-sizing: border-box; }}
            body {{
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                padding: 20px;
            }}
            .container {{
                background: white;
                border-radius: 16px;
                padding: 40px;
                max-width: 700px;
                margin: 0 auto;
                box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            }}
            h1 {{ color: #333; margin-bottom: 20px; }}
            .warning {{
                background: #fff3cd;
                border-left: 4px solid #ffc107;
                padding: 16px;
                margin: 20px 0;
                border-radius: 4px;
                font-weight: 600;
                color: #856404;
            }}
            .key-display {{
                background: #f8f9fa;
                border: 2px solid #dee2e6;
                border-radius: 8px;
                padding: 20px;
                margin: 20px 0;
                word-break: break-all;
                font-family: 'Courier New', monospace;
            }}
            .actions {{
                display: grid;
                grid-template-columns: repeat(3, 1fr);
                gap: 10px;
                margin: 20px 0;
            }}
            .actions button {{
                padding: 12px;
                background: white;
                border: 2px solid #667eea;
                color: #667eea;
                border-radius: 8px;
                cursor: pointer;
            }}
            .actions button:hover {{ background: #667eea; color: white; }}
            .checkbox-container {{
                margin: 30px 0;
                padding: 20px;
                background: #f8f9fa;
                border-radius: 8px;
            }}
            #continue-btn {{
                width: 100%;
                padding: 16px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                border: none;
                border-radius: 8px;
                font-size: 1.1em;
                cursor: pointer;
            }}
            #continue-btn:disabled {{ background: #ccc; cursor: not-allowed; }}
        </style>
    </head>
    <body>
        <div class="container">
            <h1>✅ Clé générée !</h1>
            
            <div class="warning">
                ⚠️ IMPORTANT : Sauvegardez cette clé MAINTENANT dans un endroit sûr
            </div>
            
            <div class="key-display">
                <code id="key">{key}</code>
            </div>
            
            <div class="actions">
                <button onclick="copyKey()">📋 Copier</button>
                <button onclick="downloadKey()">💾 Télécharger</button>
                <button onclick="alert('Scannez ce code avec Bitwarden')">📱 Enregistrer</button>
            </div>
            
            <div class="checkbox-container">
                <label>
                    <input type="checkbox" id="saved" onchange="toggleContinue()">
                    J'ai sauvegardé ma clé en lieu sûr (Bitwarden, KeePass, clé USB, etc.)
                </label>
            </div>
            
            <form method="POST" action="/setup/complete">
                <input type="hidden" name="key" value="{key}">
                <button id="continue-btn" type="submit" disabled>Continuer la configuration</button>
            </form>
        </div>
        
        <script>
            function copyKey() {{
                navigator.clipboard.writeText(document.getElementById('key').textContent);
                alert('✅ Clé copiée !');
            }}
            
            function downloadKey() {{
                const key = document.getElementById('key').textContent;
                const blob = new Blob([key], {{ type: 'text/plain' }});
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'anemone-restic-key.txt';
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
            }}
            
            function toggleContinue() {{
                document.getElementById('continue-btn').disabled = !document.getElementById('saved').checked;
            }}
        </script>
    </body>
    </html>
    """)

@app.get("/setup/restore", response_class=HTMLResponse)
async def setup_restore():
    """Page de restauration avec clé existante"""
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 Restauration</title>
        <meta charset="utf-8">
        <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body {
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                display: flex;
                align-items: center;
                justify-content: center;
                padding: 20px;
            }
            .container {
                background: white;
                border-radius: 16px;
                padding: 40px;
                max-width: 600px;
                width: 100%;
                box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            }
            h1 { color: #333; margin-bottom: 20px; }
            .info {
                background: #d1ecf1;
                border-left: 4px solid #17a2b8;
                padding: 16px;
                margin: 20px 0;
                border-radius: 4px;
                color: #0c5460;
            }
            label { display: block; margin: 20px 0 8px; font-weight: 600; }
            input[type="password"], input[type="text"] {
                width: 100%;
                padding: 12px;
                border: 2px solid #dee2e6;
                border-radius: 8px;
                font-family: 'Courier New', monospace;
            }
            input:focus { outline: none; border-color: #667eea; }
            .divider { text-align: center; margin: 30px 0; color: #666; }
            input[type="file"] {
                width: 100%;
                padding: 12px;
                border: 2px dashed #dee2e6;
                border-radius: 8px;
                cursor: pointer;
            }
            button {
                width: 100%;
                padding: 16px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                border: none;
                border-radius: 8px;
                font-size: 1.1em;
                cursor: pointer;
                margin-top: 20px;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>♻️ Restauration</h1>
            
            <div class="info">
                Restaurez votre serveur avec une clé existante (depuis Bitwarden, fichier backup, etc.)
            </div>
            
            <form method="POST" action="/setup/complete" onsubmit="return validateForm()">
                <label for="key">Collez votre clé Restic :</label>
                <input type="password" name="key" id="key" required 
                       placeholder="Depuis Bitwarden, KeePass, etc.">
                
                <div class="divider">OU</div>
                
                <label for="keyfile">📁 Importer depuis un fichier :</label>
                <input type="file" id="keyfile" accept=".txt" onchange="loadFile(event)">
                
                <button type="submit">Valider et continuer</button>
            </form>
        </div>
        
        <script>
            function loadFile(event) {
                const file = event.target.files[0];
                if (file) {
                    const reader = new FileReader();
                    reader.onload = (e) => {
                        document.getElementById('key').value = e.target.result.trim();
                    };
                    reader.readAsText(file);
                }
            }
            
            function validateForm() {
                const key = document.getElementById('key').value.trim();
                if (key.length < 20) {
                    alert('⚠️ La clé semble trop courte');
                    return false;
                }
                return true;
            }
        </script>
    </body>
    </html>
    """)

@app.post("/setup/complete")
async def setup_complete(key: str = Form(...)):
    """Finaliser le setup"""
    key = key.strip()
    
    if not validate_restic_key(key):
        raise HTTPException(400, "Format de clé invalide")
    
    success = encrypt_restic_key(key)
    
    if not success:
        raise HTTPException(500, "Erreur lors du chiffrement")
    
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
    <head>
        <title>✅ Configuration terminée</title>
        <meta http-equiv="refresh" content="5;url=/">
        <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body {
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                display: flex;
                align-items: center;
                justify-content: center;
                padding: 20px;
            }
            .container {
                background: white;
                border-radius: 16px;
                padding: 40px;
                max-width: 600px;
                text-align: center;
            }
            h1 { color: #28a745; font-size: 2.5em; }
            .success { background: #d4edda; padding: 16px; margin: 20px 0; border-radius: 4px; color: #155724; }
            .warning { background: #fff3cd; padding: 20px; margin: 20px 0; border-radius: 4px; }
            a {
                display: inline-block;
                margin-top: 20px;
                padding: 12px 24px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                text-decoration: none;
                border-radius: 8px;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>✅ Configuration terminée</h1>
            <div class="success">La clé a été enregistrée de manière sécurisée.</div>
            <div class="warning">
                <h3>⚠️ Important</h3>
                <ul style="text-align: left;">
                    <li>Cette page ne s'affichera plus jamais</li>
                    <li>La clé n'est PAS accessible via l'interface</li>
                    <li>Conservez votre copie en lieu sûr</li>
                </ul>
            </div>
            <a href="/">Accéder au tableau de bord</a>
            <p style="margin-top: 20px; color: #666;">Redirection dans 5 secondes...</p>
        </div>
    </body>
    </html>
    """)

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
    node_name = config.get('node', {}).get('name', 'Anemone')
    
    return HTMLResponse(f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 {node_name}</title>
        <meta charset="utf-8">
        <style>
            * {{ margin: 0; padding: 0; box-sizing: border-box; }}
            body {{
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                padding: 20px;
            }}
            .container {{ max-width: 1200px; margin: 0 auto; }}
            .header {{ text-align: center; color: white; margin-bottom: 40px; }}
            .header h1 {{ font-size: 3em; margin-bottom: 10px; }}
            .card {{
                background: white;
                border-radius: 12px;
                padding: 24px;
                margin-bottom: 20px;
                box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            }}
            .card h2 {{ margin-bottom: 16px; color: #333; }}
            a {{ color: #667eea; text-decoration: none; }}
            a:hover {{ text-decoration: underline; }}
        </style>
    </head>
    <body>
        <div class="container">
            <div class="header">
                <h1>🪸 {node_name}</h1>
                <p>Serveur de fichiers distribué</p>
            </div>
            
            <div class="card">
                <h2>📊 État du système</h2>
                <ul style="line-height: 2;">
                    <li><a href="/api/status">État général</a></li>
                    <li><a href="/api/services">Services</a></li>
                    <li><a href="/docs">Documentation API</a></li>
                </ul>
            </div>
        </div>
    </body>
    </html>
    """)

@app.get("/health")
async def health():
    return {{"status": "healthy", "setup_completed": is_setup_completed()}}

@app.get("/api/status")
async def get_status():
    config = load_config()
    return {{
        "node": {{
            "name": config.get('node', {{}}).get('name', 'unknown'),
            "role": config.get('node', {{}}).get('role', 'unknown'),
        }},
        "timestamp": datetime.now().isoformat(),
        "setup_completed": is_setup_completed()
    }}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000)
