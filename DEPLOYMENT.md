# Deployment to HiveVibe Infrastructure

This document describes the deployment configuration for `simple-go-api-server` to the HiveVibe EKS cluster.

## Overview

This application is configured to deploy to:
- **Cluster**: `hive-vibe` (EKS, Kubernetes 1.34)
- **AWS Account**: `189768267137`
- **Region**: `us-east-1`
- **Domain**: `simple-go-api-server.arch.beescloud.com`

## Helm Chart

### Location
```
charts/simple-go-api-server/
├── Chart.yaml          # Chart metadata
├── values.yaml         # Default values
└── templates/          # Kubernetes manifests
    ├── _helpers.tpl    # Template helpers
    ├── deployment.yaml # Application deployment
    ├── service.yaml    # ClusterIP service (port 8080)
    ├── ingress.yaml    # NGINX ingress with TLS
    └── serviceaccount.yaml
```

### Key Values

**Image**:
```yaml
image:
  repository: 189768267137.dkr.ecr.us-east-1.amazonaws.com/simple-go-api-server-image
  pullPolicy: Always
  tag: ""  # Set by CI/CD
```

**Ingress** (with automatic TLS and DNS):
```yaml
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    external-dns.alpha.kubernetes.io/hostname: "simple-go-api-server.arch.beescloud.com"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
```

**Resources**:
```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

## CloudBees Workflow

The workflow (`.cloudbees/workflows/workflow.yml`) performs:

### Build Job
1. Checkout code
2. Build Go binary
3. Build and push Docker image to ECR
4. Package Helm chart with image tag
5. Push Helm chart to ECR (OCI format)

### Deploy Job
1. Configure AWS and EKS credentials
2. Install/upgrade Helm release to `simple-go-api-server` namespace
3. Override ingress values with hostname and TLS configuration

## Deployment Process

### Automatic Steps

When you push code or manually trigger the workflow:

1. **CI/CD builds and deploys** the application
2. **cert-manager** automatically requests a TLS certificate from Let's Encrypt
3. **external-dns** automatically creates/updates the Route53 DNS record
4. **NGINX Ingress** serves the application with TLS termination

### Timeline

- **0-5 min**: Build, test, push image
- **5-7 min**: Deploy to Kubernetes
- **7-10 min**: TLS certificate issued (first time only)
- **10-12 min**: DNS record propagated (first time only)

Subsequent deployments are faster (~5-7 min total) as TLS cert and DNS already exist.

## Verification

### Check Deployment
```bash
kubectl get pods -n simple-go-api-server
kubectl get svc -n simple-go-api-server
kubectl get ingress -n simple-go-api-server
```

### Check TLS Certificate
```bash
kubectl get certificate -n simple-go-api-server
kubectl describe certificate -n simple-go-api-server simple-go-api-server-tls
```

### Check DNS Record
```bash
dig simple-go-api-server.arch.beescloud.com
```

### Test Application
```bash
# From Tailscale or within VPC (NLB is internal)
curl https://simple-go-api-server.arch.beescloud.com
```

## Local Development

### Build Docker Image Locally
```bash
docker build -t simple-go-api-server:local .
docker run -p 8080:8080 simple-go-api-server:local
```

### Test Helm Chart Locally
```bash
# Render templates
helm template simple-go-api-server ./charts/simple-go-api-server

# Lint chart
helm lint ./charts/simple-go-api-server

# Dry-run install
helm install simple-go-api-server ./charts/simple-go-api-server \
  --dry-run --debug \
  --set image.tag=test
```

## Architecture

```
User (via Tailscale)
    ↓
Route53: simple-go-api-server.arch.beescloud.com
    ↓
Internal NLB (AWS Load Balancer Controller)
    ↓
NGINX Ingress Controller (TLS termination with Let's Encrypt)
    ↓
Service: simple-go-api-server (ClusterIP:8080)
    ↓
Pod: simple-go-api-server
```

## Troubleshooting

### Pods Not Starting
```bash
kubectl logs -n simple-go-api-server deployment/simple-go-api-server
kubectl describe pod -n simple-go-api-server <pod-name>
```

### Certificate Issues
```bash
# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager

# Check certificate status
kubectl get certificaterequest -n simple-go-api-server
kubectl get challenge -n simple-go-api-server
```

### DNS Issues
```bash
# Check external-dns logs
kubectl logs -n kube-system deployment/external-dns

# Verify Route53 record
aws route53 list-resource-record-sets \
  --hosted-zone-id Z02190362SPX32M9Q7S3O \
  --query "ResourceRecordSets[?Name=='simple-go-api-server.arch.beescloud.com.']"
```

### Ingress Issues
```bash
# Check NGINX ingress logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller

# Verify NLB
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

## References

- [HiveVibe Infrastructure Repo](https://github.com/YourOrg/arch-infra)
- [App Deployment Guide](/Users/swashburn/dev/arch-infra/docs/app-deployment-guide.md)
- [Rebuild Plan](/Users/swashburn/dev/arch-infra/docs/rebuild-plan.md)
