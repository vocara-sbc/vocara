# Vocara Call Center System - Kubernetes Deployment Guide

## Directory Structure

```
deploy/k8s/
├── namespace.yaml          # Namespace
├── configmap.yaml          # Configuration files for all services
├── mysql.yaml              # MySQL 8.0 (stateful)
├── redis.yaml              # Redis 7 (stateful, AOF persistence)
├── nats.yaml               # NATS JetStream (stateful)
├── vocara-admin.yaml       # Admin API
├── vocara-agentcenter.yaml # Agent center
├── vocara-acd.yaml         # ACD (intelligent queueing)
├── vocara-callctrl.yaml    # Call control
├── vocara-push.yaml        # WebSocket real-time push
├── vocara-web.yaml         # Frontend entry point (Nginx: static hosting + five reverse proxy groups)
├── freeswitch-sbc.yaml     # FreeSWITCH access-side media node (hostNetwork)
├── freeswitch-agent.yaml   # FreeSWITCH agent-side media node (hostNetwork, includes port 5066 Service)
├── ingress.yaml            # NGINX Ingress (routes all traffic to vocara-web)
└── README.md               # This document
```

## Architecture Overview

```
                  ┌─────────────────────────┐
                  │    Ingress (NGINX)      │
                  │   vocara.example.com    │
                  └────────────┬────────────┘
                               │
                  ┌────────────▼────────────┐
                  │ vocara-web (Nginx :80)  │
                  │ Static hosting + proxy  │
                  └──┬─────┬─────┬────┬─────┘
                     │     │     │    │
        ┌────────────▼┐ ┌───▼───┐ │ ┌──▼──────────┐
        │ vocara-admin│ │vocara-│ │ │vocara-push  │
        │    (8080)   │ │agent- │ │ │  (8084/WS)  │
        └─────────────┘ │center │ │ └─────────────┘
                        │(8081) │ │
        ┌────────────┐  └───────┘ │
        │ vocara-    │            │
        │ callctrl   │◄───────────┤
        │  (8083)    │  /callapi  │
        └─────┬──────┘  /agentapi │
              │ ESL              │
   ┌──────────▼─────────┐        │
   │ vocara-fs-agent    │◄───────┘ /sip-ws
   │ (hostNetwork:5066) │
   └────────────────────┘

   MySQL 8.0   Redis 7   NATS JetStream (shared infrastructure for all microservices)
```

## Deployment Order

Deploy in the following order to ensure that dependent services start first:

### Step 1: Create the Namespace

```bash
kubectl apply -f namespace.yaml
```

### Step 2: Deploy the Configuration Files

```bash
kubectl apply -f configmap.yaml
```

### Step 3: Deploy the Infrastructure (Stateful Services)

```bash
# Deploy MySQL
kubectl apply -f mysql.yaml

# Deploy Redis
kubectl apply -f redis.yaml

# Deploy NATS
kubectl apply -f nats.yaml
```

Wait until all infrastructure services are ready:

```bash
kubectl -n vocara get pods
# Confirm that the mysql/redis/nats Pods are Running and READY is 1/1
```

### Step 4: Deploy the Microservices

```bash
# Deploy all application services
kubectl apply -f vocara-admin.yaml
kubectl apply -f vocara-agentcenter.yaml
kubectl apply -f vocara-acd.yaml
kubectl apply -f vocara-callctrl.yaml
kubectl apply -f vocara-push.yaml
```

### Step 5: Deploy the FreeSWITCH Media Nodes

Prepare the hostPath directories and configuration under `/data/vocara/freeswitch/` on the nodes in advance (the header of each manifest includes directory and configuration placement instructions), and allow privileged containers and hostNetwork.

```bash
kubectl apply -f freeswitch-sbc.yaml
kubectl apply -f freeswitch-agent.yaml
```

### Step 6: Deploy the Frontend Entry Point

```bash
kubectl apply -f vocara-web.yaml
```

### Step 7: Deploy Ingress

```bash
# Ensure that the NGINX Ingress Controller is installed
kubectl apply -f ingress.yaml
```

### One-Click Deployment (All Resources)

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

## Verify Deployment

### View All Resources

```bash
# View all resources in the namespace
kubectl -n vocara get all

# View the status of all Pods
kubectl -n vocara get pods -o wide

# View Services
kubectl -n vocara get svc

# View ConfigMaps
kubectl -n vocara get configmap
```

### View Logs

```bash
# View logs for a service
kubectl -n vocara logs -f deployment/vocara-admin

# View MySQL logs
kubectl -n vocara logs -f statefulset/vocara-mysql
```

### Test Service Connectivity

```bash
# Start a temporary Pod for testing
kubectl run -it --rm debug --image=busybox:1.36 --namespace=vocara --restart=Never

# Test each service from within the debug Pod
# wget -qO- http://vocara-admin:8080/health
# wget -qO- http://vocara-redis:6379  # redis-cli requires the redis image
# exit
```

## How to Edit Configuration

### Method 1: Edit the ConfigMap (Recommended)

Configuration is centrally managed in `configmap.yaml`. Edit it and apply the changes:

```bash
# Edit the configuration
kubectl -n vocara edit configmap vocara-admin-config

# Or edit the file and apply it again
kubectl apply -f configmap.yaml

# Restart the Pods for configuration changes to take effect (rolling restart)
kubectl -n vocara rollout restart deployment/vocara-admin
```

> **Tip**: In production, use a `config-hash` annotation to automatically trigger a rolling update when the configuration changes. This works with Kustomize or Helm.

### Method 2: Environment Variable Overrides

Inject sensitive configuration (passwords, secrets, etc.) through environment variables by editing the `env` field in each service's Deployment. All environment variables use the `VOCARA_` prefix. `pkg/config` uses viper AutomaticEnv to recognize `VOCARA_<CONFIG_SECTION>_<KEY>` and override the corresponding configuration item.

```yaml
env:
  - name: VOCARA_MYSQL_PASSWORD
    value: "your_strong_password"
```

Common environment variables:

| Environment Variable | Overridden Configuration Item |
|---|---|
| VOCARA_APP_SECRET_KEY | app.secret_key (JWT signing and internal service tokens) |
| VOCARA_MYSQL_PASSWORD | mysql.password |
| VOCARA_REDIS_PASSWORD | redis.password |
| VOCARA_NATS_PASSWORD | nats.password |
| VOCARA_ESL_HOST | esl.host (agent-side media node IP) |
| VOCARA_ESL_PASSWORD | esl.password |
| VOCARA_FS_DIALPLAN_SBC_PEER | fs_dialplan.sbc_peer |
| VOCARA_FS_DIALPLAN_AGENT_EXTERNAL | fs_dialplan.agent_external |
| VOCARA_ACD_AGENT_CENTER_TOKEN | acd.agent_center_token |

### Method 3: Use Secrets (Recommended for Production)

Store sensitive data in Kubernetes Secrets and reference it using `secretKeyRef`:

```bash
# Create a Secret
kubectl -n vocara create secret generic vocara-secrets \
  --from-literal=app-secret-key='your_strong_secret_key' \
  --from-literal=db-password='your_db_password' \
  --from-literal=redis-password='your_redis_password' \
  --from-literal=nats-password='your_nats_password' \
  --from-literal=esl-password='your_esl_password'
```

Then reference it in the Deployment:

```yaml
env:
  - name: VOCARA_MYSQL_PASSWORD
    valueFrom:
      secretKeyRef:
        name: vocara-secrets
        key: db-password
```

### Change the Image Version

Edit the `image` field in each service's Deployment:

```yaml
image: registry.example.com/vocara/vocara-admin:v1.0.0
```

Or update it quickly with a command:

```bash
kubectl -n vocara set image deployment/vocara-admin \
  vocara-admin=registry.example.com/vocara/vocara-admin:v1.0.0
```

## How to Scale Services

### Manual Scaling

```bash
# Scale vocara-admin to 5 replicas
kubectl -n vocara scale deployment vocara-admin --replicas=5

# Scale vocara-push down to 2 replicas (fewer than 2 is not recommended)
kubectl -n vocara scale deployment vocara-push --replicas=2
```

### Autoscaling (HPA)

Automatically scale based on CPU/memory utilization (requires metrics-server):

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: vocara-admin-hpa
  namespace: vocara
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: vocara-admin
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Notes on Scaling WebSocket Services

- `vocara-push` is a long-lived WebSocket connection service. Scaling out is easy, but scaling in is difficult.
- Scaling in disconnects connections. Set a longer `terminationGracePeriodSeconds`.
- In production, use custom metrics based on connection counts for autoscaling.
- Avoid frequent scaling.

## Resource Configuration Reference

The following are the default resource configurations. Adjust them based on your actual workload:

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit | Replica Count |
|------|---------|---------|---------|---------|-------|
| vocara-admin | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-agentcenter | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-acd | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-callctrl | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-push | 100m | 500m | 256Mi | 512Mi | 3 |
| MySQL | 250m | 1000m | 512Mi | 1Gi | 1 |
| Redis | 100m | 500m | 128Mi | 512Mi | 1 |
| NATS | 100m | 500m | 128Mi | 512Mi | 1 |

> **Note**: Go services typically have low memory usage. 128Mi-512Mi is sufficient for small and medium-scale deployments.
> Adjust CPU and memory limits appropriately for high-concurrency scenarios.

## Notes

### 1. Production Security

- **Change all default passwords and secrets**: MySQL root password, application secret (app.secret_key, used for JWT signing), etc.
- Use Kubernetes **Secrets** to manage sensitive data. Do not write it in plaintext in YAML.
- Enable **HTTPS**: configure a TLS certificate for Ingress.
- Configure **NetworkPolicy** to restrict access between services.
- Do not connect to MySQL as the root user. Create a dedicated database user.

### 2. Data Persistence

- MySQL, Redis, and NATS all use PVCs for persistent storage.
- Deleting a StatefulSet does not automatically delete its PVCs. The data is retained.
- To permanently delete the data, manually delete the PVCs.
- In production, use a highly available StorageClass.

### 3. High Availability Recommendations

- The current templates provide single-node MySQL/Redis/NATS and are suitable only for development and testing.
- Recommendations for production:
  - MySQL: primary-replica replication or a cloud database (such as RDS)
  - Redis: Redis Cluster or Sentinel mode
  - NATS: 3-node cluster
  - All microservices: start with at least 2 replicas

### 4. WebSocket Considerations

- WebSocket protocol upgrades are configured in Ingress.
- Ensure that proxy-read-timeout is long enough (3600s by default).
- WebSocket services use long-lived connections. Drain connections during rolling updates.
- If session persistence is required, enable `sessionAffinity: ClientIP` in the Service.

### 5. Database Initialization and Schema

- The `vocara-mysql-initdb` ConfigMap is mounted at `/docker-entrypoint-initdb.d/`, and MySQL executes its database creation statements on first startup.
- The baseline database and table DDL is located in `db/migrations/001_init_schema.sql` in the repository. For delivery, add this SQL to the initdb ConfigMap so that it runs on first startup.
- Each service maintains its own schema using GORM AutoMigrate, controlled by the `mysql.auto_migrate` configuration item (enabled by default for vocara-admin; vocara-callctrl maintains the call detail record table during startup).

### 6. FreeSWITCH Media Nodes

- FreeSWITCH handles media and protocol processing and includes two nodes: FS-SBC (carrier access side) and FS-Agent (agent side).
- The deployment manifests are `freeswitch-sbc.yaml` and `freeswitch-agent.yaml`. They run with hostNetwork and privileged containers, and mount configuration and recordings through hostPath.
- vocara-callctrl connects to FS-Agent through ESL (8021). The node IP is injected through `VOCARA_ESL_HOST`, and the ESL password through `VOCARA_ESL_PASSWORD`.
- SIP ports: FS-SBC internal 5060 / external 5080; FS-Agent 5060/5080 and WebRTC 5066/7443.
- Node configuration directory content comes from the conf directory for the corresponding node under `freeswitch/` in the core repository.

### 7. Monitoring and Logs

- All services write logs to stdout. View them with `kubectl logs`.
- Recommended production deployments:
  - **Prometheus + Grafana**: metrics monitoring
  - **ELK / Loki**: log collection
  - **Jaeger / Zipkin**: distributed tracing

### 8. Image Registry

- The default value is the placeholder `registry.example.com/vocara/`.
- Replace it with your actual image registry address.
- For a private registry, configure `imagePullSecrets`.

### 9. Version Compatibility

- Kubernetes version requirement: 1.20+ (uses stable APIs such as `apps/v1` and `networking.k8s.io/v1`)
- NGINX Ingress Controller version: 1.0+

### 10. Service Environment Variable Injection

Service environment variable injection is disabled for the Pods of all workloads (`enableServiceLinks: false`). Service discovery uses CoreDNS resolution of Service names (such as `vocara-mysql:3306`, `vocara-redis:6379`, and `vocara-nats:4222` in the configuration) and does not depend on the `*_SERVICE_HOST`/`*_PORT` variables injected by kubelet. This prevents the `VOCARA_MYSQL_PORT=tcp://<host>:3306` variable (legacy combined format) injected by kubelet for the `vocara-mysql` Service from being mapped back to `mysql.port` (int) by viper `AutomaticEnv` and causing configuration parsing to fail. Explicitly declared `VOCARA_*` environment variables are unaffected.

## FAQ

### Q: Why does a Pod remain Pending?

```bash
# View events
kubectl -n vocara describe pod <pod-name>
```

Common causes:
- Insufficient node resources
- PVC cannot bind (no available PV / StorageClass)
- Image pull failure (private registry credential issue)

### Q: Why does a service fail to start and cannot connect to MySQL?

1. Confirm that the MySQL Pod is Running.
2. View the MySQL logs to confirm that initialization is complete.
3. Check that the database password configuration is consistent.
4. Confirm that the Service name `vocara-mysql` resolves within the vocara namespace.

### Q: Why do WebSocket connections disconnect?

1. Check that the WebSocket configuration in Ingress is correct.
2. Confirm that `proxy-read-timeout` is long enough.
3. Check that the push service heartbeat configuration is correct.
4. Check the service logs for errors.

### Q: How do I delete all resources?

```bash
# Delete the namespace (deletes all resources in the namespace)
kubectl delete namespace vocara

# Note: PVCs may not be deleted automatically and may require manual cleanup
kubectl -n vocara delete pvc --all
```

## Further Optimization Recommendations

1. Use a **Helm Chart** to manage deployments (template and parameterize the manifests).
2. Introduce **Kustomize** to manage multi-environment configurations (dev/staging/prod).
3. Configure **PodDisruptionBudget** to ensure service availability.
4. Configure **Pod Security Admission** / **SecurityContext** to strengthen security.
5. Implement **blue-green deployments** / **canary releases**.
6. Integrate a **Service Mesh** (such as Istio) for traffic management.
