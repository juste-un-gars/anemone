#!/usr/bin/env python3
from fastapi import FastAPI
from fastapi.responses import HTMLResponse
import os

app = FastAPI(title="Anemone API")

@app.get("/", response_class=HTMLResponse)
async def root():
    return """
    <html>
        <head><title>🪸 Anemone</title></head>
        <body style="font-family: sans-serif; text-align: center; padding: 50px;">
            <h1>🪸 Anemone API</h1>
            <p>Service actif</p>
            <p><a href="/docs">Documentation</a></p>
        </body>
    </html>
    """

@app.get("/health")
async def health():
    return {"status": "healthy"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000)
