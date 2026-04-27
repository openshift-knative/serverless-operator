# Knative OpenTelemetry Metrics Migration Guide for Custom Dashboards

This guide helps you migrate custom Grafana dashboards and monitoring queries from OpenCensus metrics to OpenTelemetry metrics after the Knative 1.21 migration.

## Overview

In Knative 1.21, both Serving and Eventing migrated from OpenCensus to OpenTelemetry (OTel) for metrics and tracing. This change affects metric names, label names, and metric units. If you have custom dashboards or alerting rules, you'll need to update them to use the new metric names.

**Important**: Not all components migrated to seconds-based metrics. Some Kafka-based components still use millisecond-based metrics. Pay careful attention to the metric names and units for each component.

## General Changes

### Metric Naming Conventions

OpenTelemetry follows [semantic conventions](https://opentelemetry.io/docs/specs/semconv/general/metrics/#units) for metric naming:
- **Units in metric names vary by component** - some use `_seconds`, others still use `_ms`
- **Units are provided via metric metadata** using `metric.WithUnit`
- **Metric names use semantic prefixes** like `kn.serving.*`, `kn.eventing.*`, `http.server.*`, `http.client.*`

### Unit Changes

**Important**: Unit migration is inconsistent across components:
- **MT Broker (Ingress/Filter)** and **InMemoryChannel**: Changed from milliseconds to **seconds**
- **Kafka Broker**, **KafkaChannel**, **KafkaSource**, **KafkaSink**: Still use **milliseconds** (`_ms`)
- **Serving** (Queue Proxy, Activator, Autoscaler): Changed to **seconds**
- **ApiServerSource** and **PingSource**: Use standard HTTP client metrics in **seconds**

## Eventing Metrics Migration

### Broker Metrics (MT Broker Ingress)

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `mt_broker_ingress_event_count` | `kn_eventing_dispatch_duration_seconds_count` | Add `job="mt-broker-ingress-sm-service"`, **changed to seconds** |
| `mt_broker_ingress_event_dispatch_latencies_bucket` | `kn_eventing_dispatch_duration_seconds_bucket` | **Changed from ms to seconds** |

**Example Migration:**
```promql
# Old query
sum(rate(mt_broker_ingress_event_count{namespace_name="default"}[1m]))

# New query
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m]))
```

```promql
# Old latency query (milliseconds)
histogram_quantile(0.99, sum(rate(mt_broker_ingress_event_dispatch_latencies_bucket{namespace_name="default"}[1m])) by (le))

# New latency query (SECONDS - unit changed!)
histogram_quantile(0.99, sum(rate(kn_eventing_dispatch_duration_seconds_bucket{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m])) by (le))
```

### Broker Metrics (MT Broker Filter)

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `mt_broker_filter_event_count` | `kn_eventing_dispatch_duration_seconds_count` | Add `job="mt-broker-filter-sm-service"`, **changed to seconds** |
| `mt_broker_filter_event_dispatch_latencies_bucket` | `kn_eventing_dispatch_duration_seconds_bucket` | **Changed from ms to seconds** |
| `mt_broker_filter_event_processing_latencies_bucket` | `kn_eventing_process_duration_seconds_bucket` | Event processing time, **changed to seconds** |

**Example Migration:**
```promql
# Old query
sum(rate(mt_broker_filter_event_count{namespace_name="default"}[1m]))

# New query
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="mt-broker-filter-sm-service", kn_broker_namespace="default"}[1m]))
```

### InMemoryChannel Dispatcher

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `inmemorychannel_dispatcher_event_count` | `kn_eventing_dispatch_duration_seconds_count` | Add `job="imc-dispatcher-sm-service"`, **changed to seconds** |
| `inmemorychannel_dispatcher_event_dispatch_latencies_bucket` | `kn_eventing_dispatch_duration_seconds_bucket` | **Changed from ms to seconds** |

**Example Migration:**
```promql
# Old query
sum(rate(inmemorychannel_dispatcher_event_count{namespace_name="default"}[1m]))

# New query
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="imc-dispatcher-sm-service", kn_channel_namespace="default"}[1m]))
```

### Kafka Channel Receiver

**Note: KafkaChannel still uses millisecond-based metrics!**

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `event_count_1_total` | `kn_eventing_dispatch_latency_ms_count` | Add `job="kafka-channel-receiver-sm-service"`, **STILL milliseconds!** |
| `event_dispatch_latencies_ms_bucket` | `kn_eventing_dispatch_latency_ms_bucket` | **STILL milliseconds!** |

**Example Migration:**
```promql
# Old query
sum(rate(event_count_1_total{job="kafka-channel-receiver-sm-service", namespace_name="default"}[1m]))

# New query - NOTE: still uses _ms metric and kn_kafkachannel_namespace label
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-channel-receiver-sm-service", kn_kafkachannel_namespace="default"}[1m]))
```

### Kafka Broker (Receiver & Dispatcher)

**Note: Kafka Broker still uses millisecond-based metrics and has DIFFERENT namespace labels for receiver vs dispatcher!**

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `event_count_1_total` (receiver) | `kn_eventing_dispatch_latency_ms_count` | Add `job="kafka-broker-receiver-sm-service"`, **STILL milliseconds!** |
| `event_dispatch_latencies_ms_bucket` (receiver) | `kn_eventing_dispatch_latency_ms_bucket` | **STILL milliseconds!** |
| `event_count_1_total` (dispatcher) | `kn_eventing_dispatch_latency_ms_count` | Add `job="kafka-broker-dispatcher-sm-service"`, **STILL milliseconds!** |
| `event_dispatch_latencies_ms_bucket` (dispatcher) | `kn_eventing_dispatch_latency_ms_bucket` | **STILL milliseconds!** |
| `event_processing_latencies_ms_bucket` (dispatcher) | `kn_eventing_process_latency_ms_bucket` | Processing time metric, **STILL milliseconds!** |

**Important: Kafka Broker Receiver uses `kn_broker_namespace` but Dispatcher uses `kn_trigger_namespace`!**

**Example Migration (Receiver):**
```promql
# Old query
sum(rate(event_count_1_total{job="kafka-broker-receiver-sm-service", namespace_name="default"}[1m]))

# New query - uses kn_broker_namespace
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-broker-receiver-sm-service", kn_broker_namespace="default"}[1m]))
```

**Example Migration (Dispatcher):**
```promql
# Old query
sum(rate(event_count_1_total{job="kafka-broker-dispatcher-sm-service", namespace_name="default"}[1m]))

# New query - uses kn_trigger_namespace (DIFFERENT from receiver!)
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-broker-dispatcher-sm-service", kn_trigger_namespace="default"}[1m]))
```

### Source Metrics

#### ApiServerSource

**Note: ApiServerSource uses HTTP client metrics, not eventing dispatch metrics!**

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `apiserversource_event_count` | `http_client_request_duration_seconds_count` | Uses HTTP client metric with regex job pattern, **changed to seconds** |

**Example Migration:**
```promql
# Old query
sum(rate(apiserversource_event_count{namespace_name="default"}[1m]))

# New query - uses http_client metric and kn_source_namespace with regex job pattern
sum(rate(http_client_request_duration_seconds_count{job=~"apiserversource-.*", kn_source_namespace="default"}[1m]))
```

#### PingSource

**Note: PingSource uses HTTP client metrics, not eventing dispatch metrics!**

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `pingsource_event_count` | `http_client_request_duration_seconds_count` | Uses HTTP client metric, **changed to seconds** |

**Example Migration:**
```promql
# Old query
sum(rate(pingsource_event_count{namespace_name="default"}[1m]))

# New query - uses http_client metric and kn_source_namespace
sum(rate(http_client_request_duration_seconds_count{job="pingsource-mt-adapter-sm-service", kn_source_namespace="default"}[1m]))
```

#### KafkaSource

**Note: KafkaSource still uses millisecond-based metrics!**

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `event_count_1_total` | `kn_eventing_dispatch_latency_ms_count` | Add `job="kafka-source-dispatcher-sm-service"`, **STILL milliseconds!** |

**Example Migration:**
```promql
# Old query
sum(rate(event_count_1_total{job="kafka-source-dispatcher-sm-service", namespace_name="default"}[1m]))

# New query - still uses _ms metric
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-source-dispatcher-sm-service", namespace="knative-eventing", kn_kafkasource_namespace="default"}[1m]))
```

### Kafka Sink

**Note: Kafka Sink still uses millisecond-based metrics!**

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `event_count_1_total` | `kn_eventing_dispatch_latency_ms_count` | Add `job="kafka-sink-receiver-sm-service"`, **STILL milliseconds!** |
| `event_dispatch_latencies_ms_bucket` | `kn_eventing_dispatch_latency_ms_bucket` | **STILL milliseconds!** |

**Example Migration:**
```promql
# Old query
sum(rate(event_count_1_total{job="kafka-sink-receiver-sm-service", namespace_name="default"}[1m]))

# New query - still uses _ms metric
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-sink-receiver-sm-service", kn_kafkasink_namespace="default"}[1m]))
```

## Serving Metrics Migration

### Queue Proxy / Revision Metrics

| Old Metric Name | New Metric Name | Notes |
|----------------|-----------------|-------|
| `revision_app_request_count` | `kn_serving_invocation_duration_seconds_count` | Application request count, **changed to seconds** |
| `revision_app_request_latencies_bucket` | `kn_serving_invocation_duration_seconds_bucket` | **Changed from ms to seconds** |

**Example Migration:**
```promql
# Old query
sum(rate(revision_app_request_count{namespace="default", revision_name=~"hello.*"}[1m]))

# New query
sum(rate(kn_serving_invocation_duration_seconds_count{k8s_namespace_name="default", kn_revision_name=~"hello.*"}[1m]))
```

```promql
# Old latency query
histogram_quantile(0.99, sum(rate(revision_app_request_latencies_bucket{namespace="default"}[1m])) by (le))

# New latency query (SECONDS - unit changed!)
histogram_quantile(0.99, sum(rate(kn_serving_invocation_duration_seconds_bucket{k8s_namespace_name="default"}[1m])) by (le))
```

### Activator Metrics

| Old Metric Name | New Metric Name | Notes                                                                             |
|----------------|-----------------|-----------------------------------------------------------------------------------|
| `activator_request_count` | `http_server_request_duration_seconds_count` | Add `job="activator-sm-service"`, Standard OTel HTTP metric, **changed to seconds** |
| `activator_request_latencies_bucket` | `http_server_request_duration_seconds_bucket` | Add `job="activator-sm-service"`, **Changed from ms to seconds**                    |
| `request_concurrency` | `kn_revision_request_concurrency` | Concurrent requests                               |

**Example Migration:**
```promql
# Old query
sum(rate(activator_request_count{namespace="default"}[1m]))

# New query - uses standard HTTP server metric
sum(rate(http_server_request_duration_seconds_count{job="activator-sm-service", k8s_namespace_name="default"}[1m]))
```

### Autoscaler Metrics

All autoscaler metrics have been renamed with the `kn_revision_*` or `kn_autoscaler_*` prefix:

| Old Metric Name | New Metric Name |
|----------------|-----------------|
| `autoscaler_actual_pods` | `kn_revision_pods_count` |
| `autoscaler_desired_pods` | `kn_revision_pods_desired` |
| `autoscaler_requested_pods` | `kn_revision_pods_requested` |
| `autoscaler_pending_pods` | `kn_revision_pods_pending_count` |
| `autoscaler_not_ready_pods` | `kn_revision_pods_not_ready_count` |
| `autoscaler_terminating_pods` | `kn_revision_pods_terminating_count` |
| `autoscaler_stable_request_concurrency` | `kn_revision_concurrency_stable` |
| `autoscaler_panic_request_concurrency` | `kn_revision_concurrency_panic` |
| `autoscaler_target_concurrency_per_pod` | `kn_revision_concurrency_target` |
| `autoscaler_excess_burst_capacity` | `kn_revision_capacity_excess` |
| `autoscaler_panic_mode` | `kn_revision_panic_mode` |
| `autoscaler_stable_requests_per_second` | `kn_revision_rps_stable_per_second` |
| `autoscaler_panic_requests_per_second` | `kn_revision_rps_panic` |
| `autoscaler_target_requests_per_second` | `kn_revision_rps_target` |
| `autoscaler_scrape_time_bucket` | `kn_autoscaler_scrape_duration_bucket` |

**Example Migration:**
```promql
# Old query
sum(autoscaler_actual_pods{namespace_name="default", revision_name="hello-00001"})

# New query
sum(kn_revision_pods_count{k8s_namespace_name="default", kn_revision_name="hello-00001"})
```

```promql
# Old scrape time query
histogram_quantile(0.99, sum(rate(autoscaler_scrape_time_bucket{namespace_name="default"}[1m])) by (le))

# New scrape time query (SECONDS - unit changed!)
histogram_quantile(0.99, sum(rate(kn_autoscaler_scrape_duration_bucket{k8s_namespace_name="default"}[1m])) by (le))
```

## Label Name Changes

Label names have been updated to follow OpenTelemetry semantic conventions. **Note that different components use different namespace label names:**

### Serving Components

| Old Label Name | New Label Name |
|---------------|---------------|
| `namespace` | `k8s_namespace_name` |
| `namespace_name` | `k8s_namespace_name` |
| `revision_name` | `kn_revision_name` |
| `configuration_name` | `kn_configuration_name` |
| `response_code_class` | `http_response_status_code` |
| `response_code` | `http_response_status_code` |

### Eventing Components - Namespace Labels

**Critical: Different Eventing components use different namespace labels:**

| Component | Old Label | New Label |
|-----------|-----------|-----------|
| MT Broker Ingress | `namespace_name` | `kn_broker_namespace` |
| MT Broker Filter | `namespace_name` | `kn_broker_namespace` |
| InMemoryChannel | `namespace_name` | `kn_channel_namespace` |
| **KafkaChannel** | `namespace_name` | **`kn_kafkachannel_namespace`** |
| Kafka Broker Receiver | `namespace_name` | `kn_broker_namespace` |
| **Kafka Broker Dispatcher** | `namespace_name` | **`kn_trigger_namespace`** ⚠️ |
| ApiServerSource | `namespace_name` | `kn_source_namespace` (generic) |
| PingSource | `namespace_name` | `kn_source_namespace` (generic) |
| KafkaSource | `namespace_name` | `kn_kafkasource_namespace` |
| Kafka Sink | `namespace_name` | `kn_kafkasink_namespace` |

### Other Common Label Changes

| Old Label Name | New Label Name | Components |
|---------------|---------------|------------|
| `response_code_class` | `http_response_status_code` | All components |
| `response_code` | `http_response_status_code` | All components |
| `event_type` | `cloudevents_type` | MT Broker, Kafka Broker, Kafka Sink |

### Response Code Pattern Changes

**Important**: Response code regex patterns differ between Serving and Eventing:

**Eventing** (MT Broker, Kafka Broker, Channels, Sources):
```promql
# Success: 2xx
http_response_status_code=~"2.*"

# Error: non-2xx
http_response_status_code!~"2.*"
```

**Serving** (Queue Proxy, Activator):
```promql
# Success: 2xx
http_response_status_code=~"2.."

# Error: non-2xx
http_response_status_code!~"2.."
```

**Example:**
```promql
# Old query (success rate) - Eventing
sum(rate(mt_broker_ingress_event_count{response_code_class="2xx"}[1m])) /
sum(rate(mt_broker_ingress_event_count[1m]))

# New query (success rate) - Eventing uses "2.*"
sum(rate(kn_eventing_dispatch_duration_seconds_count{http_response_status_code=~"2.*"}[1m])) /
sum(rate(kn_eventing_dispatch_duration_seconds_count[1m]))
```

```promql
# Old query (success rate) - Serving
sum(rate(revision_app_request_count{response_code_class="2xx"}[1m]))

# New query (success rate) - Serving uses "2.."
sum(rate(kn_serving_invocation_duration_seconds_count{http_response_status_code=~"2.."}[1m]))
```

## Migration Checklist

When migrating your custom dashboards:

- [ ] Update metric names following the tables above
- [ ] **Check if metric uses seconds or milliseconds** - Kafka components still use `_ms`!
- [ ] Update label names - **pay special attention to namespace labels** which vary by component type:
  - [ ] Serving metrics: use `k8s_namespace_name`
  - [ ] MT Broker: use `kn_broker_namespace`
  - [ ] **Kafka Broker Receiver: use `kn_broker_namespace`**
  - [ ] **Kafka Broker Dispatcher: use `kn_trigger_namespace`** ⚠️ Different!
  - [ ] InMemoryChannel: use `kn_channel_namespace`
  - [ ] **KafkaChannel: use `kn_kafkachannel_namespace`** (component-specific)
  - [ ] **ApiServerSource/PingSource: use `kn_source_namespace`** (generic, not component-specific)
  - [ ] KafkaSource: use `kn_kafkasource_namespace`
  - [ ] Kafka Sink: use `kn_kafkasink_namespace`
- [ ] Update resource-specific labels (`revision_name` → `kn_revision_name`, `configuration_name` → `kn_configuration_name`)
- [ ] Add appropriate `job` label filters for Eventing metrics
  - [ ] **ApiServerSource uses regex pattern `job=~"apiserversource-.*"`**
- [ ] Change `response_code_class` filters:
  - [ ] Eventing: `="2xx"` → `=~"2.*"`
  - [ ] Serving: `="2xx"` → `=~"2.."`
- [ ] Change `response_code_class` error filters:
  - [ ] Eventing: `!="2xx"` → `!~"2.*"`
  - [ ] Serving: `!="2xx"` → `!~"2.."`
- [ ] Update legend formats to use new label names
- [ ] **Check event type labels**: `event_type` → `cloudevents_type`
- [ ] Verify histogram buckets:
  - [ ] MT Broker/Filter, InMemoryChannel, Serving: use `_seconds_bucket`
  - [ ] Kafka components: use `_ms_bucket` or `_latency_ms_bucket`
- [ ] Adjust dashboard panel display units if needed (milliseconds → seconds where applicable)
- [ ] Update alert thresholds considering unit changes
- [ ] Test queries against your metrics endpoint to verify data availability

## Job Labels for Eventing Components

When querying Eventing metrics, you'll need to include the appropriate `job` label to distinguish between components:

| Component | Job Label Value | Job Pattern |
|-----------|----------------|-------------|
| MT Broker Ingress | `mt-broker-ingress-sm-service` | Exact match |
| MT Broker Filter | `mt-broker-filter-sm-service` | Exact match |
| InMemoryChannel Dispatcher | `imc-dispatcher-sm-service` | Exact match |
| Kafka Broker Receiver | `kafka-broker-receiver-sm-service` | Exact match |
| Kafka Broker Dispatcher | `kafka-broker-dispatcher-sm-service` | Exact match |
| Kafka Channel Receiver | `kafka-channel-receiver-sm-service` | Exact match |
| Kafka Sink Receiver | `kafka-sink-receiver-sm-service` | Exact match |
| **ApiServerSource** | `apiserversource-.*` | **Regex pattern** `job=~"apiserversource-.*"` |
| PingSource | `pingsource-mt-adapter-sm-service` | Exact match |
| KafkaSource | `kafka-source-dispatcher-sm-service` | Exact match |

## Common Migration Patterns

### Pattern 1: Event Rate (MT Broker)

```promql
# OLD
sum(rate(mt_broker_ingress_event_count{namespace_name="default"}[1m]))

# NEW - uses seconds-based metric
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m]))
```

### Pattern 2: Event Rate (Kafka Broker Receiver)

```promql
# OLD
sum(rate(event_count_1_total{job="kafka-broker-receiver-sm-service", namespace_name="default"}[1m]))

# NEW - still uses milliseconds-based metric!
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-broker-receiver-sm-service", kn_broker_namespace="default"}[1m]))
```

### Pattern 3: Event Rate (Kafka Broker Dispatcher)

```promql
# OLD
sum(rate(event_count_1_total{job="kafka-broker-dispatcher-sm-service", namespace_name="default"}[1m]))

# NEW - uses kn_trigger_namespace (different from receiver!)
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-broker-dispatcher-sm-service", kn_trigger_namespace="default"}[1m]))
```

### Pattern 4: Success Rate (Eventing)

```promql
# OLD
sum(rate(mt_broker_ingress_event_count{response_code_class="2xx", namespace_name="default"}[1m])) /
sum(rate(mt_broker_ingress_event_count{namespace_name="default"}[1m]))

# NEW - uses "2.*" pattern for eventing
sum(rate(kn_eventing_dispatch_duration_seconds_count{http_response_status_code=~"2.*", job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m])) /
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m]))
```

### Pattern 5: Success Rate (Serving)

```promql
# OLD
sum(rate(revision_app_request_count{response_code_class="2xx", namespace="default"}[1m]))

# NEW - uses "2.." pattern for serving (different from eventing!)
sum(rate(kn_serving_invocation_duration_seconds_count{http_response_status_code=~"2..", k8s_namespace_name="default"}[1m]))
```

### Pattern 6: P99 Latency (MT Broker - Seconds)

```promql
# OLD (milliseconds)
histogram_quantile(0.99,
  sum(rate(mt_broker_ingress_event_dispatch_latencies_bucket{namespace_name="default"}[1m])) by (le)
)

# NEW (seconds - unit changed!)
histogram_quantile(0.99,
  sum(rate(kn_eventing_dispatch_duration_seconds_bucket{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m])) by (le)
)
```

### Pattern 7: P99 Latency (Kafka Broker - Still Milliseconds!)

```promql
# OLD (milliseconds)
histogram_quantile(0.99,
  sum(rate(event_dispatch_latencies_ms_bucket{job="kafka-broker-receiver-sm-service", namespace_name="default"}[1m])) by (le)
)

# NEW (still milliseconds - unit NOT changed!)
histogram_quantile(0.99,
  sum(rate(kn_eventing_dispatch_latency_ms_bucket{job="kafka-broker-receiver-sm-service", kn_broker_namespace="default"}[1m])) by (le)
)
```

### Pattern 8: Grouping by Response Code

```promql
# OLD
sum(rate(mt_broker_ingress_event_count{namespace_name="default"}[1m])) by (response_code_class)

# NEW
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m])) by (http_response_status_code)
```

Don't forget to update the legend format:
```
# OLD
{{response_code_class}}

# NEW
{{http_response_status_code}}
```

### Pattern 9: Grouping by Event Type (MT Broker)

```promql
# OLD
sum(rate(mt_broker_ingress_event_count{namespace_name="default"}[1m])) by (event_type)

# NEW
sum(rate(kn_eventing_dispatch_duration_seconds_count{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}[1m])) by (cloudevents_type)
```

Legend format:
```
# OLD
{{event_type}}

# NEW
{{cloudevents_type}}
```

### Pattern 10: Grouping by Event Type (Kafka Sink)

```promql
# OLD
sum(rate(event_count_1_total{job="kafka-sink-receiver-sm-service", namespace_name="default"}[1m])) by (event_type)

# NEW - uses cloudevents_type
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-sink-receiver-sm-service", kn_kafkasink_namespace="default"}[1m])) by (cloudevents_type)
```

### Pattern 11: ApiServerSource (HTTP Client Metrics)

```promql
# OLD
sum(rate(apiserversource_event_count{namespace_name="default"}[1m]))

# NEW - uses http_client metric with regex job pattern and generic kn_source_namespace
sum(rate(http_client_request_duration_seconds_count{job=~"apiserversource-.*", kn_source_namespace="default"}[1m]))
```

### Pattern 12: KafkaChannel-Specific Query

**Note: KafkaChannel uses unique namespace label `kn_kafkachannel_namespace` and millisecond metrics:**

```promql
# OLD
sum(rate(event_count_1_total{job="kafka-channel-receiver-sm-service", namespace_name="default"}[1m]))

# NEW - Note the kn_kafkachannel_namespace label and _ms metric
sum(rate(kn_eventing_dispatch_latency_ms_count{job="kafka-channel-receiver-sm-service", kn_kafkachannel_namespace="default"}[1m]))
```

## Component-Specific Label Reference

To avoid confusion, here's a quick reference showing the exact namespace label to use for each component:

```promql
# Serving - Queue Proxy (seconds)
kn_serving_invocation_duration_seconds_count{k8s_namespace_name="default", kn_revision_name="..."}

# Serving - Activator (seconds)
http_server_request_duration_seconds_count{k8s_namespace_name="default", job="activator-sm-service", kn_revision_name="..."}

# Serving - Autoscaler
kn_revision_pods_count{k8s_namespace_name="default", kn_revision_name="..."}

# MT Broker Ingress (seconds)
kn_eventing_dispatch_duration_seconds_count{job="mt-broker-ingress-sm-service", kn_broker_namespace="default"}

# MT Broker Filter (seconds)
kn_eventing_dispatch_duration_seconds_count{job="mt-broker-filter-sm-service", kn_broker_namespace="default"}

# InMemoryChannel (seconds)
kn_eventing_dispatch_duration_seconds_count{job="imc-dispatcher-sm-service", kn_channel_namespace="default"}

# KafkaChannel (milliseconds!) - DIFFERENT NAMESPACE LABEL
kn_eventing_dispatch_latency_ms_count{job="kafka-channel-receiver-sm-service", kn_kafkachannel_namespace="default"}

# Kafka Broker Receiver (milliseconds!)
kn_eventing_dispatch_latency_ms_count{job="kafka-broker-receiver-sm-service", kn_broker_namespace="default"}

# Kafka Broker Dispatcher (milliseconds!) - DIFFERENT NAMESPACE LABEL
kn_eventing_dispatch_latency_ms_count{job="kafka-broker-dispatcher-sm-service", kn_trigger_namespace="default"}

# ApiServerSource (seconds) - HTTP CLIENT metric with REGEX job pattern
http_client_request_duration_seconds_count{job=~"apiserversource-.*", kn_source_namespace="default"}

# PingSource (seconds) - HTTP CLIENT metric
http_client_request_duration_seconds_count{job="pingsource-mt-adapter-sm-service", kn_source_namespace="default"}

# KafkaSource (milliseconds!)
kn_eventing_dispatch_latency_ms_count{job="kafka-source-dispatcher-sm-service", namespace="knative-eventing", kn_kafkasource_namespace="default"}

# Kafka Sink (milliseconds!)
kn_eventing_dispatch_latency_ms_count{job="kafka-sink-receiver-sm-service", kn_kafkasink_namespace="default"}
```

## Quick Reference: Metric Units by Component

To quickly identify whether a component uses seconds or milliseconds:

**Seconds (`_seconds_count`, `_seconds_bucket`):**
- MT Broker Ingress
- MT Broker Filter
- InMemoryChannel
- All Serving components (Queue Proxy, Activator, Autoscaler)
- ApiServerSource (HTTP client metric)
- PingSource (HTTP client metric)

**Milliseconds (`_ms_count`, `_latency_ms_bucket`, `_ms_bucket`):**
- Kafka Broker Receiver
- Kafka Broker Dispatcher
- KafkaChannel
- KafkaSource
- Kafka Sink

## Resources

- [Knative OpenTelemetry Migration Proposal](https://docs.google.com/document/d/1QQ_ubc0RjeZbRHdN4rQR85Z7RZfTSjz4GoKsE0dZ2Z0/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [OpenTelemetry HTTP Metrics](https://opentelemetry.io/docs/specs/semconv/http/http-metrics/)
- [Reference Dashboards](knative-operator/deploy/resources/dashboards/) in this repository

## Getting Help

If you encounter issues during migration:
1. Check that metrics are being exported by curling the `/metrics` endpoint of the relevant pods
2. Verify the `job` label in your ServiceMonitor matches what you're querying
3. **Pay special attention to the namespace label** - each component type uses a different one
4. **Verify metric units** - Kafka components still use milliseconds while others use seconds
5. Compare your queries with the [reference dashboards](knative-operator/deploy/resources/dashboards/) in this repository
