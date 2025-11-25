# Deploy AI-KMS to Kubernetes

## Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- Docker registry access
- NGINX Ingress Controller (for Ingress)
- cert-manager (optional, for TLS)

---

## Quick Start

### 1. Build and Push Docker Image

```bash
# Build the image
docker build -t your-registry/ai-kms:latest .

# Push to registry
docker push your-registry/ai-kms:latest
```

### 2. Update Configuration

Edit `k8s/configmap.yaml` and `k8s/ai-kms.yaml`:

```yaml
# In ai-kms.yaml
image: your-registry/ai-kms:latest  # Your actual registry

# In configmap.yaml (secrets)
OPENAI_API_KEY: "your-actual-key"
```

### 3. Deploy to Kubernetes

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create ConfigMap and Secrets
kubectl apply -f k8s/configmap.yaml

# Deploy PostgreSQL
kubectl apply -f k8s/postgres.yaml

# Deploy Jaeger
kubectl apply -f k8s/jaeger.yaml

# Deploy AI-KMS
kubectl apply -f k8s/ai-kms.yaml

#Apply Ingress (optional)
kubectl apply -f k8s/ingress.yaml
```

### 4. Verify Deployment

```bash
# Check all pods
kubectl get pods -n ai-kms

# Check services
kubectl get svc -n ai-kms

# Check logs
kubectl logs -n ai-kms -l app=ai-kms --tail=100 -f
```

---

## Architecture

```
┌─────────────────────────────────────────┐
│          Ingress Controller             │
│     (ai-kms.example.com)                │
└─────────────┬───────────────────────────┘
              │
      ┌───────▼────────┐
      │  LoadBalancer  │
      │  Service       │
      └───────┬────────┘
              │
    ┌─────────▼─────────┐
    │   AI-KMS Pods     │
    │   (3 replicas)    │
    │   + HPA           │
    └─────┬─────┬───────┘
          │     │
    ┌─────▼─┐ ┌▼────────┐
    │ Redis │ │Postgres │
    │ (opt) │ │+pgvector│
    └───────┘ └─────────┘
```

---

## Components

### 1. Namespace
- Isolates AI-KMS resources
- File: `namespace.yaml`

### 2. PostgreSQL
- Persistent storage with PVC
- pgvector extension
- Health checks
- File: `postgres.yaml`

### 3. Jaeger
- Distributed tracing
- Multiple services (collector, query, agent)
- File: `jaeger.yaml`

### 4. AI-KMS Application
- 3 replicas (horizontal scaling)
- Health checks (liveness + readiness)
- HPA (auto-scaling)
- WebSocket-friendly service (sticky sessions)
- File: `ai-kms.yaml`

### 5. Ingress
- External traffic routing
- WebSocket support
- TLS termination (optional)
- File: `ingress.yaml`

---

## Scaling

### Manual Scaling

```bash
# Scale up
kubectl scale deployment ai-kms -n ai-kms --replicas=5

# Scale down
kubectl scale deployment ai-kms -n ai-kms --replicas=2
```

### Auto-Scaling (HPA)

Already configured in `ai-kms.yaml`:
- Min replicas: 3
- Max replicas: 10
- CPU threshold: 70%
- Memory threshold: 80%

```bash
# Check HPA status
kubectl get hpa -n ai-kms

# Describe HPA
kubectl describe hpa ai-kms-hpa -n ai-kms
```

---

## Monitoring

### Pod Status

```bash
# Watch pods
kubectl get pods -n ai-kms --watch

# Describe pod
kubectl describe pod <pod-name> -n ai-kms
```

### Logs

```bash
# Stream logs
kubectl logs -n ai-kms -l app=ai-kms -f

# Previous container logs (if crashed)
kubectl logs -n ai-kms <pod-name> --previous
```

### Jaeger UI

```bash
# Port-forward Jaeger UI
kubectl port-forward -n ai-kms svc/jaeger-query 16686:16686

# Access at http://localhost:16686
```

### Metrics

```bash
# Pod resource usage
kubectl top pods -n ai-kms

# Node resource usage
kubectl top nodes
```

---

## Security Best Practices

### 1. Use Secrets Management

**Option A: Kubernetes Secrets (Basic)**
```bash
kubectl create secret generic ai-kms-secrets \
  --from-literal=OPENAI_API_KEY=your-key \
  --from-literal=DB_PASSWORD=strong-password \
  -n ai-kms
```

**Option B: External Secrets Operator (Recommended)**
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: ai-kms-secrets
spec:
  secretStoreRef:
    name: aws-secrets-manager
  target:
    name: ai-kms-secrets
  data:
  - secretKey: OPENAI_API_KEY
    remoteRef:
      key: ai-kms/openai-key
```

### 2. Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ai-kms-netpol
  namespace: ai-kms
spec:
  podSelector:
    matchLabels:
      app: ai-kms
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 8080
```

### 3. Resource Quotas

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: ai-kms-quota
  namespace: ai-kms
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
```

---

## Troubleshooting

### Pods Not Starting

```bash
# Check events
kubectl get events -n ai-kms --sort-by='.lastTimestamp'

# Describe deployment
kubectl describe deployment ai-kms -n ai-kms

# Check pod logs
kubectl logs -n ai-kms <pod-name>
```

### Database Connection Issues

```bash
# Test postgres connectivity
kubectl run -it --rm debug --image=postgres:latest --restart=Never -n ai-kms -- \
  psql -h postgres-service -U postgres -d ai_kms

# Check postgres logs
kubectl logs -n ai-kms -l app=postgres
```

### WebSocket Not Working

1. Check sticky sessions in Service
2. Verify Ingress WebSocket annotations
3. Check timeout settings
4. Test with: `websocat ws://your-domain/ws/document/test`

---

## Updates & Rollouts

### Rolling Update

```bash
# Update image
kubectl set image deployment/ai-kms ai-kms=your-registry/ai-kms:v2 -n ai-kms

# Watch rollout
kubectl rollout status deployment/ai-kms -n ai-kms

# Check history
kubectl rollout history deployment/ai-kms -n ai-kms
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/ai-kms -n ai-kms

# Rollback to specific revision
kubectl rollout undo deployment/ai-kms --to-revision=2 -n ai-kms
```

---

## Cleanup

```bash
# Delete all resources
kubectl delete namespace ai-kms

# Or delete individual components
kubectl delete -f k8s/
```

---

## Production Checklist

- [ ] Set proper resource limits
- [ ] Configure persistent volumes for data
- [ ] Enable TLS/SSL
- [ ] Set up external secrets management
- [ ] Configure backup strategy for PostgreSQL
- [ ] Set up monitoring (Prometheus/Grafana)
- [ ] Configure log aggregation (ELK/Loki)
- [ ] Implement network policies
- [ ] Set resource quotas
- [ ] Configure pod disruption budgets
- [ ] Test disaster recovery
- [ ] Document runbook

---

## Next Steps

1. **CI/CD**: Automate deployments with GitOps (ArgoCD/Flux)
2. **Monitoring**: Add Prometheus + Grafana
3. **Backup**: Velero for cluster backups
4. **Security**: Pod security policies, OPA/Gatekeeper
5. **Service Mesh**: Istio for advanced traffic management

🚀 **Your AI-KMS is now running in Kubernetes!**
