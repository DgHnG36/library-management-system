# Monitoring

This directory contains the observability stack for the Library Management System, powered by **Prometheus** (metrics collection) and **Grafana** (visualization).

---

## Components

| Component  | Default Port | Description                            |
| ---------- | ------------ | -------------------------------------- |
| Prometheus | `9090`       | Scrapes and stores time-series metrics |
| Grafana    | `3000`       | Dashboards and alerting UI             |

---

## Directory Layout

```
monitoring/
├── prometheus/
│   ├── prometheus.yml        # Scrape configurations
│   └── data/                 # Prometheus persistent data (git-ignored)
└── grafana/
    ├── provisioning/         # Auto-provisioned datasources and dashboards
    └── data/                 # Grafana persistent data (git-ignored)
```

---

## Metrics Endpoint

The **Gateway Service** exposes Prometheus metrics at:

```
GET http://localhost:8080/metrics
```

Prometheus is configured to scrape this endpoint via `prometheus/prometheus.yml`.

---

## Running the Stack

The monitoring services are included in `docker-compose.yaml`. Start them alongside the application:

```bash
docker-compose up -d grafana prometheus
```

Then open:

- Prometheus: [http://localhost:9090](http://localhost:9090)
- Grafana: [http://localhost:3000](http://localhost:3000) (default credentials: `admin` / `admin`)

---

## Grafana Dashboards

Dashboards are provisioned automatically from `grafana/provisioning/`. On first start, Grafana will load the datasources and dashboard definitions without any manual setup.

---

## See Also

- [services/gateway-service/](../services/gateway-service/) — Exposes `/metrics`
