FROM python:3.11-alpine

RUN apk add --no-cache gcc musl-dev libffi-dev openssl-dev curl docker-cli

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir --break-system-packages -r requirements.txt

COPY . .

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s \
    CMD curl -f http://localhost:3000/health || exit 1

CMD ["python", "main.py"]
