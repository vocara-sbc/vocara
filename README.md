# Vocara Call Center System

> Vocal + Atelier — a professional voice workshop

Vocara is an enterprise-grade intelligent call center system that provides an
integrated solution from call access, intelligent queueing, and agent handling
to call management, real-time monitoring, and analytics. It ships with a
built-in WebRTC softphone and an agent phone bar, so you can run inbound and
outbound call operations without dedicated handsets and deploy to a single
machine or a Kubernetes cluster in one step.

## Features

### Core Calling

- **Agent management**: create, edit, batch import/export, enable/disable, and bind skills, extensions, and passwords for agents.
- **Team organization**: agent groups and skill groups with flexible membership and skill levels.
- **Extension management**: extension registration and status checks, batch import, and agent binding.
- **Queue management**: queue configuration and real-time monitoring of queueing and answering.
- **ACD intelligent queueing**: skill-based agent matching with priority queueing that routes calls to available agents automatically.
- **Inbound reception**: inbound routing rules and holiday scheduling that direct calls to the right queue.
- **Call control**: unified inbound/outbound handling with call recording, monitoring, and whisper.
- **Call records**: inbound/outbound CDR query, recording playback and export, and today's traffic overview.

### Operations & Monitoring

- **Real-time agent status**: agent states and call events are synced to managers in real time.
- **Line resource management**: line configuration and channel status statistics.
- **Reports**: multi-dimensional traffic statistics by agent, queue, and line, with trend analysis and real-time dashboards.
- **Multi-tenancy**: tenant-isolated data, with events and CDRs partitioned by tenant.

### Endpoints & Security

- **WebRTC softphone**: take and place calls from the browser, no hardware handset required.
- **Phone bar**: a global agent phone bar that keeps calls active across page navigation.
- **License authorization**: offline license files bound to the machine fingerprint, with expiry alerts and policy control.

## Prerequisites

| Dependency | Version | Purpose |
|------|------|------|
| MySQL | 8.0 | Business data and CDR storage |
| Redis | 7 | Cache and real-time agent state |
| NATS JetStream | 2.10 | Event messaging |
| FreeSWITCH | 1.10.x | SIP and media handling |

> Under Kubernetes deployment, MySQL, Redis, and NATS are provided by the
> manifests; for single-machine deployment, prepare these dependencies
> yourself, with FreeSWITCH deployed and integrated by your implementation team.

## Installation

Vocara supports three deployment forms: **Windows single-machine**, **Linux
single-machine**, and **Kubernetes cluster**.

### Option 1: Windows single-machine

1. Obtain the `dist/` package (backend services, web frontend, Nginx, and one-click start/stop scripts);
2. Edit the MySQL / Redis / NATS / ESL addresses for each service under `configs/` as needed;
3. Place the vendor-provided `vocara.lic` in `dist/license/`;
4. Double-click `start.cmd` to start all services, then open `http://localhost`;
5. Double-click `stop.cmd` to stop.

### Option 2: Linux single-machine

1. Run `db/migrations/001_init_schema.sql` to initialize the schema;
2. Edit the infrastructure addresses under `configs/` as needed;
3. Place the vendor-provided `vocara.lic` in the package's `license/` directory;
4. Run `./start.sh` to start the backend services (`./stop.sh` to stop);
5. Serve `web/` as the site root and include the `nginx/nginx.conf` site snippet in your system Nginx;
6. Open the Nginx site in a browser and sign in.

### Option 3: Kubernetes cluster

Apply the manifests under `deploy/k8s/` in order:

```bash
cd deploy/k8s
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f mysql.yaml
kubectl apply -f redis.yaml
kubectl apply -f nats.yaml
kubectl apply -f vocara-admin.yaml
kubectl apply -f vocara-agentcenter.yaml
kubectl apply -f vocara-acd.yaml
kubectl apply -f vocara-callctrl.yaml
kubectl apply -f vocara-push.yaml
kubectl apply -f freeswitch-sbc.yaml
kubectl apply -f freeswitch-agent.yaml
kubectl apply -f vocara-web.yaml
kubectl apply -f ingress.yaml
```

Access the system through the Ingress hostname (default `vocara.example.com`).
See [deploy/k8s/README.md](deploy/k8s/README.md) for full steps, configuration, and scaling.

## Documentation

| Document | Description |
|------|------|
| [Package guide](deploy/README.md) | `dist/` layout and single-machine usage |
| [Kubernetes deployment](deploy/k8s/README.md) | Cluster deployment steps, configuration, scaling, and FAQ |