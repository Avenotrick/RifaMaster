# RifaMaster

App web liviana para rifas con pago integrado via **Mercado Pago**.

- Backend en Go (un solo binario)
- SQLite embebido (no requiere base de datos externa)

---

## Hosteo rápido (Hostinger VPS / cualquier VPS)

Elegí una de las dos opciones:

- **Opción A — Directo con systemd** (más simple, menos capas)
- **Opción B — Docker** (más aislado, más fácil de actualizar)

---

### Opcion A: Directo con systemd

#### 1. Compila el binario para Linux

En tu maquina (sin importar si usas Windows, Mac o Linux):

```bash
GOOS=linux GOARCH=amd64 go build -o rifamaster .
```

Esto genera un binario listo para Linux amd64.

#### 2. Subi los archivos al VPS

```bash
scp rifamaster rifa-master.zip
# mejor con rsync:
rsync -avz rifamaster .env static/ usuario@tu-vps:/home/rifamaster/
```

Tambien tienen que estar:
- `.env` — configuracion real (crealo a partir de `.env.example`)
- `static/` — contiene el frontend (`index.html`)

#### 3. Servicio systemd (para que arranque solo)

```ini
# /etc/systemd/system/rifamaster.service
[Unit]
Description=RifaMaster
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/home/rifamaster
ExecStart=/home/rifamaster/rifamaster
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now rifamaster
```

Para ver logs:
```bash
journalctl -u rifamaster -f
```

#### 4. Reverse proxy con Caddy (HTTPS automatico)

Instala Caddy en el VPS y agrega esto en `/etc/caddy/Caddyfile`:

```
tusitio.com {
    reverse_proxy localhost:3000
}
```

Caddy se encarga de HTTPS con Let's Encrypt automaticamente.

---

### Opcion B: Docker

Con Docker no necesitas instalar Go ni compilar nada en tu maquina — todo se hace en el VPS.

#### 1. Instala Docker en el VPS

```bash
curl -fsSL https://get.docker.com | sh
```

#### 2. En el VPS, clona el repo y construi la imagen

```bash
git clone https://github.com/tuusuario/rifamaster.git
cd rifamaster
cp .env.example .env
nano .env                  # completa con tus datos reales
docker build -t rifamaster .
```

El `Dockerfile` incluido descarga Go, compila el binario, y genera una imagen final de ~10 MB.

#### 3. Corre el contenedor

```bash
docker run -d \
  --name rifamaster \
  --restart always \
  -p 3000:3000 \
  -v rifa-data:/app \
  rifamaster
```

**Explicacion de cada flag:**

| Flag | Que hace |
|---|---|
| `-d` | Corre en segundo plano (detached) |
| `--name rifamaster` | Le pone nombre para manejarlo facil |
| `--restart always` | Si el VPS se reinicia, arranca solo |
| `-p 3000:3000` | Expone el puerto 3000 al VPS |
| `-v rifa-data:/app` | Volumen persistente para que SQLite sobreviva aunque borres el contenedor |

#### 4. Comandos utiles para el dia a dia

```bash
# Ver logs en vivo
docker logs -f rifamaster

# Frenar
docker stop rifamaster

# Iniciar de nuevo
docker start rifamaster

# Actualizar a nueva version
docker stop rifamaster && docker rm rifamaster
docker build -t rifamaster .
docker run -d --name rifamaster --restart always -p 3000:3000 -v rifa-data:/app rifamaster
```

#### 5. Reverse proxy con Caddy (HTTPS)

```bash
docker run -d \
  --name caddy \
  --restart always \
  -p 80:80 -p 443:443 \
  -v caddy-data:/data \
  -v $PWD/Caddyfile:/etc/caddy/Caddyfile \
  caddy
```

Crea un `Caddyfile` al lado:

```
tusitio.com {
    reverse_proxy rifamaster:3000
}
```

Como Caddy y rifamaster corren en Docker, se ven por nombre de contenedor (`rifamaster`) en vez de `localhost`.

#### 6. Docker Compose (todo junto, recomendado)

Si preferis levantar todo de un solo comando, crea un `docker-compose.yml`:

```yaml
version: "3"
services:
  rifamaster:
    build: .
    restart: always
    ports:
      - "3000:3000"
    volumes:
      - rifa-data:/app
    env_file: .env

  caddy:
    image: caddy
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - caddy-data:/data
      - ./Caddyfile:/etc/caddy/Caddyfile

volumes:
  rifa-data:
  caddy-data:
```

```bash
docker compose up -d
```

Para actualizar despues:

```bash
git pull
docker compose build rifamaster
docker compose up -d rifamaster
```

#### 7. Backup de la base de datos

```bash
docker run --rm -v rifa-data:/data -v $PWD/backups:/backups alpine \
  cp /data/rifa.db /backups/rifa-$(date +%Y%m%d).db
```

Agregalo a cron para backup diario automatico:

```bash
crontab -e
0 3 * * * docker run --rm -v rifa-data:/data -v /home/usuario/backups:/backups alpine cp /data/rifa.db /backups/rifa-$(date +\%Y\%m\%d).db
```

---

## Variables de entorno (`.env`)

Copia `.env.example` como `.env` y completa solo las que necesites:

| Variable | Para que sirve | Obligatoria |
|---|---|---|
| `MERCADO_PAGO_ACCESS_TOKEN` | **Procesar pagos.** El token de produccion de Mercado Pago (empieza con `APP_USR-`). Lo sacas de tu cuenta MP | **Si**, sino la app no arranca |
| `MERCADO_PAGO_WEBHOOK_SECRET` | **Validar webhooks.** Un secreto que inventas vos para asegurarte que las notificaciones de pago vienen de MP y no de un atacante | Recomendada |
| `FRONTEND_URL` | **URL publica del sitio.** Se usa para redirecciones, ej. `https://tusitio.com` | Recomendada |
| `SMTP_HOST` / `SMTP_USER` / `SMTP_PASS` | **Notificaciones por email.** Si se completa, al comprar un numero llega un mail de confirmacion. Si no, los emails solo se loguean. Para Gmail necesitas una [contrasena de aplicacion](https://myaccount.google.com/apppasswords) | No |

El resto (`HOST`, `PORT`, `DATABASE_URL`) tienen valores por defecto que andan bien, no hace tocarlos.

---

## Webhook de Mercado Pago

En el panel de Mercado Pago > **Configuracion > Webhooks**, agrega:

```
https://tusitio.com/api/webhook/mercadopago
```

Selecciona el evento **Pagos** y guarda. El webhook notifica a tu app cuando un pago se aprueba o rechaza.

---

## Base de datos (SQLite)

- Se crea sola al arrancar la app
- El archivo esta en `rifa.db` (al lado del binario)
- Configura un backup diario con cron:

```bash
crontab -e
# agrega esta linea para backup diario a las 3 AM
0 3 * * * cp /home/rifamaster/rifa.db /home/rifamaster/backups/rifa-$(date +\%Y\%m\%d).db
```

---

> El proyecto incluye un `Dockerfile`. Si no usas Docker, podes borrarlo, no afecta en nada.

## Desarrollo local

### Con Go (requiere Go instalado)

```bash
go run .
```

### Con Docker (no requiere Go)

```bash
docker build -t rifamaster .
docker run -d --name rifamaster -p 3000:3000 -v rifa-data:/app rifamaster
```

Para frenarlo:

```bash
docker stop rifamaster && docker rm rifamaster
```

Despues abri http://localhost:3000.

Para pagos de prueba usa el token de MP en modo Sandbox (`TEST-...`) del `.env` y las tarjetas de prueba de Mercado Pago.
