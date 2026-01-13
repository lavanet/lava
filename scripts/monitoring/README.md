# Local Prometheus & Grafana Monitoring Setup

This directory contains Docker Compose configuration to run Prometheus and Grafana locally for monitoring Lava QoS metrics.

## Quick Start

### 1. Start the Monitoring Stack

```bash
cd scripts/monitoring
docker-compose up -d
```

### 2. Start Lava with Metrics Enabled

Run your Lava setup (the metrics endpoint on port 7779 is already configured):

```bash
./scripts/pre_setups/init_lava_only_with_node_three_providers.sh
```

### 3. Access the Dashboards

- **Grafana**: http://localhost:3000
  - Username: `admin`
  - Password: `admin`
  
- **Prometheus**: http://localhost:9090

### 4. View the Pre-configured Dashboard

1. Open Grafana at http://localhost:3000
2. Login with admin/admin
3. Go to **Dashboards** → **Lava** → **Lava Provider Selection Metrics**

## Dashboard Features

The pre-configured dashboard includes:

| Panel | Description |
|-------|-------------|
| **Provider Composite Scores** | Time series of combined QoS scores for all providers |
| **Provider Selection Rate** | How often each provider is being selected (per second) |
| **Selection Distribution** | Pie chart showing selection distribution over last 5 minutes |
| **RNG Value** | The random values used for provider selection |
| **Total Selections** | Counter of total selections made |
| **Availability Scores** | Provider availability scores over time |
| **Latency Scores** | Provider latency scores over time |
| **Sync Scores** | Provider sync scores over time |
| **Stake Scores** | Provider stake scores over time |

## Useful PromQL Queries

### Check if metrics are being scraped
```promql
up{job="lava-consumer"}
```

### View all provider metrics
```promql
{__name__=~"lava_consumer_provider.*"}
```

### Selection rate comparison
```promql
rate(lava_consumer_provider_selections[1m])
```

### Score vs Selection correlation
```promql
# Composite scores
lava_consumer_provider_composite_score

# Selection rate
rate(lava_consumer_provider_selections[5m])
```

## Troubleshooting

### Prometheus can't reach metrics endpoint

1. Check that your Lava consumer is running with metrics enabled:
   ```bash
   curl http://localhost:7779/metrics
   ```

2. On macOS/Windows, Docker uses `host.docker.internal` to reach the host machine. This is configured in `prometheus.yml`.

3. On Linux, you may need to use `--network=host` or configure the actual IP address. Edit `prometheus.yml`:
   ```yaml
   static_configs:
     - targets: ['172.17.0.1:7779']  # Docker bridge IP on Linux
   ```

### No data in Grafana

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify the consumer is exposing metrics: `curl http://localhost:7779/metrics | grep lava_consumer_provider`
3. Make sure relays are being processed (metrics update on selection events)

### Reset everything

```bash
docker-compose down -v
docker-compose up -d
```

## Configuration

### Add more scrape targets

Edit `prometheus.yml` to add more endpoints:

```yaml
scrape_configs:
  - job_name: 'lava-consumer'
    static_configs:
      - targets: ['host.docker.internal:7779']
      
  - job_name: 'lava-consumer-2'
    static_configs:
      - targets: ['host.docker.internal:7780']
```

Then reload Prometheus:
```bash
curl -X POST http://localhost:9090/-/reload
```

### Change scrape interval

Edit `prometheus.yml` to adjust the scrape interval:

```yaml
global:
  scrape_interval: 5s  # Default is 5 seconds
```

## Stop Monitoring

```bash
cd scripts/monitoring
docker-compose down
```

To also remove stored data:
```bash
docker-compose down -v
```
