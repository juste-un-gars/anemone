from fastapi import FastAPI, HTTPException, Form
from fastapi.responses import HTMLResponse, RedirectResponse
import os
import subprocess
import secrets
from pathlib import Path
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

SETUP_COMPLETED = Path("/config/.setup-completed")
RESTIC_ENCRYPTED = Path("/config/.restic.encrypted")

def is_setup_completed():
    return SETUP_COMPLETED.exists()

def generate_restic_key():
    """Générer une clé Restic aléatoire"""
    return secrets.token_urlsafe(32)

def encrypt_restic_key(key: str):
    """Chiffrer la clé Restic"""
    try:
        # Récupérer l'UUID système
        with open("/proc/sys/kernel/random/uuid") as f:
            system_key = f.read().strip()
    except:
        system_key = os.getenv('HOSTNAME', 'anemone')

    # Générer un salt
    salt = secrets.token_bytes(32)

    # Derive encryption key using PBKDF2
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=salt,
        iterations=100000,
        backend=default_backend()
    )
    derived_key = kdf.derive(f"{system_key}".encode())

    # Generate IV for AES-CBC
    iv = secrets.token_bytes(16)

    # Encrypt using AES-256-CBC
    cipher = Cipher(
        algorithms.AES(derived_key),
        modes.CBC(iv),
        backend=default_backend()
    )
    encryptor = cipher.encryptor()

    # Pad the key to be multiple of 16 bytes (AES block size)
    key_bytes = key.encode()
    padding_length = 16 - (len(key_bytes) % 16)
    padded_key = key_bytes + bytes([padding_length] * padding_length)

    # Encrypt
    encrypted = encryptor.update(padded_key) + encryptor.finalize()

    # Save encrypted data (IV + encrypted data)
    RESTIC_ENCRYPTED.write_bytes(iv + encrypted)

    # Sauvegarder le salt as hex for compatibility
    with open("/config/.restic.salt", "w") as f:
        f.write(salt.hex())

    # Marquer comme terminé
    SETUP_COMPLETED.touch()

    # Nettoyer
    if Path("/config/restic-password").exists():
        subprocess.run(["shred", "-u", "/config/restic-password"])

@app.get("/setup")
async def setup_page():
    if is_setup_completed():
        return RedirectResponse("/")
    
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 Anemone - Configuration initiale</title>
        <style>
            /* CSS moderne ici */
        </style>
    </head>
    <body>
        <div class="container">
            <h1>🪸 Bienvenue sur Anemone</h1>
            <p>Configuration initiale de votre serveur</p>
            
            <div class="option">
                <input type="radio" name="mode" value="new" id="mode-new" checked>
                <label for="mode-new">
                    <h3>🆕 Nouveau serveur</h3>
                    <p>Générer une nouvelle clé de chiffrement</p>
                </label>
            </div>
            
            <div class="option">
                <input type="radio" name="mode" value="restore" id="mode-restore">
                <label for="mode-restore">
                    <h3>♻️ Restauration</h3>
                    <p>J'ai déjà une clé de chiffrement</p>
                </label>
            </div>
            
            <button onclick="nextStep()">Continuer</button>
        </div>
        
        <script>
            function nextStep() {
                const mode = document.querySelector('input[name="mode"]:checked').value;
                window.location = `/setup/${mode}`;
            }
        </script>
    </body>
    </html>
    """)

@app.get("/setup/new")
async def setup_new():
    if is_setup_completed():
        return RedirectResponse("/")
    
    # Générer la clé
    key = generate_restic_key()
    
    return HTMLResponse(f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 Nouvelle clé générée</title>
        <style>/* CSS */</style>
    </head>
    <body>
        <div class="container">
            <h1>✅ Clé générée !</h1>
            
            <div class="warning">
                ⚠️ IMPORTANT : Sauvegardez cette clé MAINTENANT
            </div>
            
            <div class="key-display">
                <code id="key">{key}</code>
            </div>
            
            <div class="actions">
                <button onclick="copyKey()">📋 Copier</button>
                <button onclick="showQR()">📱 QR Code</button>
                <button onclick="downloadKey()">💾 Télécharger</button>
            </div>
            
            <label>
                <input type="checkbox" id="saved" onchange="toggleContinue()">
                J'ai sauvegardé ma clé en lieu sûr (Bitwarden, clé USB, etc.)
            </label>
            
            <form method="POST" action="/setup/complete">
                <input type="hidden" name="key" value="{key}">
                <button id="continue-btn" disabled>Continuer</button>
            </form>
        </div>
        
        <script>
            function copyKey() {{
                navigator.clipboard.writeText(document.getElementById('key').textContent);
                alert('Clé copiée !');
            }}
            
            function toggleContinue() {{
                document.getElementById('continue-btn').disabled = 
                    !document.getElementById('saved').checked;
            }}
        </script>
    </body>
    </html>
    """)

@app.get("/setup/restore")
async def setup_restore():
    if is_setup_completed():
        return RedirectResponse("/")
    
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
    <head>
        <title>🪸 Restauration</title>
    </head>
    <body>
        <div class="container">
            <h1>♻️ Restauration</h1>
            
            <form method="POST" action="/setup/complete">
                <label>
                    Collez votre clé Restic :
                    <input type="password" name="key" required 
                           placeholder="Depuis Bitwarden, fichier, etc.">
                </label>
                
                <p>OU</p>
                
                <label>
                    📁 Importer depuis un fichier :
                    <input type="file" id="keyfile" onchange="loadFile()">
                </label>
                
                <button type="submit">Valider et continuer</button>
            </form>
        </div>
        
        <script>
            function loadFile(event) {
                const file = event.target.files[0];
                const reader = new FileReader();
                reader.onload = (e) => {
                    document.querySelector('input[name="key"]').value = 
                        e.target.result.trim();
                };
                reader.readAsText(file);
            }
        </script>
    </body>
    </html>
    """)

@app.post("/setup/complete")
async def setup_complete(key: str = Form(...)):
    if is_setup_completed():
        raise HTTPException(400, "Setup already completed")
    
    # Valider la clé (format base64, longueur)
    if len(key) < 20:
        raise HTTPException(400, "Invalid key")
    
    # Chiffrer et stocker
    encrypt_restic_key(key)
    
    return HTMLResponse("""
    <!DOCTYPE html>
    <html>
    <head>
        <title>✅ Configuration terminée</title>
        <meta http-equiv="refresh" content="5;url=/">
    </head>
    <body>
        <div class="container">
            <h1>✅ Configuration terminée</h1>
            
            <div class="success">
                La clé a été enregistrée de manière sécurisée.
            </div>
            
            <div class="warning">
                <h3>⚠️ Important</h3>
                <ul>
                    <li>Cette page ne s'affichera plus jamais</li>
                    <li>La clé n'est PAS accessible via l'interface</li>
                    <li>La clé est chiffrée au repos</li>
                    <li>Seul le service de backup peut la lire</li>
                </ul>
            </div>
            
            <p>Redirection automatique dans 5 secondes...</p>
            <a href="/">Accéder au tableau de bord</a>
        </div>
    </body>
    </html>
    """)
