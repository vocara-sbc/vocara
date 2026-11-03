# Vocara Distribution Package Guide

`dist/` is the distribution package. It contains five backend service binaries, frontend build artifacts, configuration templates, the Windows Nginx runtime, and one-click start and stop scripts.
The package does not contain Go source code. Call control (callctrl) is delivered as a binary.

## Directory Structure

```
dist/
├── bin/          # vocara-admin / agentcenter / acd / callctrl / push
├── configs/      # Five service configurations (edit for your environment)
├── web/          # Admin frontend build artifacts
├── nginx/        # Windows Nginx runtime (single entry point with five reverse proxy groups)
├── license/      # License directory (place the separately delivered vocara.lic here)
├── records/      # Call recording directory
├── logs/         # Runtime log directory
├── start.cmd     # Windows one-click start (five services + Nginx)
└── stop.cmd      # Windows one-click stop
```

## 1. One-Click Start (Windows)

1. Edit the MySQL / Redis / NATS / ESL addresses for each service in `configs/` as needed;
2. Double-click `start.cmd`. The script uses the distribution root as its working directory and starts the five backend services and Nginx in sequence;
3. Open http://localhost in a browser;
4. Double-click `stop.cmd` to stop the services.

If a port is already in use, the corresponding service is skipped automatically to prevent duplicate startup.

## 2. License Placement

The vendor delivers `vocara.lic` separately, matched to the target machine fingerprint. Place the received file at `dist/license/vocara.lic`.
Services load it at startup according to the `license.file` configuration.

## 3. Bare-Metal Deployment on Non-Windows Systems

1. Edit the MySQL / Redis / NATS / ESL addresses for each service in `configs/`;
2. Run `./start.sh` to start the five backend services (`./stop.sh` stops them);
3. For the frontend entry point, use `web/` as the site root and include the site snippet from `nginx/nginx.conf` in the http block of the system Nginx configuration.
   Adjust the upstream addresses for your environment (or provide `VOCARA_*_ADDR` environment variables when rendering the configuration);
4. Open the Nginx site in a browser (port 80 by default). TLS is terminated by the site or an upstream load balancer.

## 4. Infrastructure Requirements

- MySQL 8.0: initialize the database and tables using `db/migrations/001_init_schema.sql`;
- Redis 7, NATS JetStream, and FreeSWITCH (WebRTC WebSocket port 5066).

Image build files are located in `deploy/docker/`, and Kubernetes deployment manifests are located in `deploy/k8s/`.
