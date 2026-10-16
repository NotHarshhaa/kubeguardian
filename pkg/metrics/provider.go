package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PrometheusProvider queries a Prometheus server for real cluster metrics
type PrometheusProvider struct {
	baseURL    string
	client     *http.Client
	queryRange time.Duration
}

// QueryResult represents a Prometheus query result
type QueryResult struct {
	Status string                 `json:"status"`
	Data   QueryData              `json:"data"`
}

type QueryData struct {
	ResultType string                  `json:"resultType"`
	Result     []QueryResultItem       `json:"result"`
}

type QueryResultItem struct {
	Metric map[string]string    `json:"metric"`
	Value  []interface{}        `json:"value"`
}

// NewPrometheusProvider creates a new Prometheus metrics provider
func NewPrometheusProvider(baseURL string, timeout time.Duration) *PrometheusProvider {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &PrometheusProvider{
		baseURL:    baseURL,
		client:     &http.Client{Timeout: timeout},
		queryRange: 5 * time.Minute,
	}
}

// QueryCPUUsage queries the average CPU usage percentage for a pod
func (p *PrometheusProvider) QueryCPUUsage(ctx context.Context, namespace, podName string) (float64, error) {
	query := fmt.Sprintf(
		`avg(rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!="POD",container!=""}[5m])) * 100`,
		namespace, podName,
	)

	result, err := p.query(ctx, query)
	if err != nil {
		return 0, err
	}

	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	// Extract the value from the first result
	val := result.Data.Result[0].Value
	if len(val) >= 2 {
		if strVal, ok := val[1].(string); ok {
			var f float64
			if _, err := fmt.Sscanf(strVal, "%f", &f); err == nil {
				return f, nil
			}
		}
	}

	return 0, nil
}

// QueryMemoryUsage queries the memory usage percentage for a pod
func (p *PrometheusProvider) QueryMemoryUsage(ctx context.Context, namespace, podName string) (float64, error) {
	query := fmt.Sprintf(
		`avg(container_memory_usage_bytes{namespace="%s",pod="%s",container!="POD",container!=""}) / avg(kube_pod_container_resource_requests_memory_bytes{namespace="%s",pod="%s",container!="POD",container!=""}) * 100`,
		namespace, podName, namespace, podName,
	)

	result, err := p.query(ctx, query)
	if err != nil {
		return 0, err
	}

	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	val := result.Data.Result[0].Value
	if len(val) >= 2 {
		if strVal, ok := val[1].(string); ok {
			var f float64
			if _, err := fmt.Sscanf(strVal, "%f", &f); err == nil {
				return f, nil
			}
		}
	}

	return 0, nil
}

// QueryNodeCPUUsage queries the CPU usage percentage for a node
func (p *PrometheusProvider) QueryNodeCPUUsage(ctx context.Context, nodeName string) (float64, error) {
	query := fmt.Sprintf(
		`avg(rate(node_cpu_seconds_total{mode!="idle",mode!="iowait",instance=~".*%s.*"}[5m])) * 100`,
		nodeName,
	)

	result, err := p.query(ctx, query)
	if err != nil {
		return 0, err
	}

	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	val := result.Data.Result[0].Value
	if len(val) >= 2 {
		if strVal, ok := val[1].(string); ok {
			var f float64
			if _, err := fmt.Sscanf(strVal, "%f", &f); err == nil {
				return f, nil
			}
		}
	}

	return 0, nil
}

// QueryNodeMemoryUsage queries the memory usage percentage for a node
func (p *PrometheusProvider) QueryNodeMemoryUsage(ctx context.Context, nodeName string) (float64, error) {
	query := fmt.Sprintf(
		`(1 - (node_memory_MemAvailable_bytes{instance=~".*%s.*"} / node_memory_MemTotal_bytes{instance=~".*%s.*"})) * 100`,
		nodeName, nodeName,
	)

	result, err := p.query(ctx, query)
	if err != nil {
		return 0, err
	}

	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	val := result.Data.Result[0].Value
	if len(val) >= 2 {
		if strVal, ok := val[1].(string); ok {
			var f float64
			if _, err := fmt.Sscanf(strVal, "%f", &f); err == nil {
				return f, nil
			}
		}
	}

	return 0, nil
}

// query executes a Prometheus instant query
func (p *PrometheusProvider) query(ctx context.Context, query string) (*QueryResult, error) {
	u, err := url.Parse(p.baseURL + "api/v1/query")
	if err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus URL: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus query failed: unexpected status %s", result.Status)
	}

	return &result, nil
}

// IsAvailable checks if the Prometheus server is reachable
func (p *PrometheusProvider) IsAvailable(ctx context.Context) bool {
	_, err := p.query(ctx, "up")
	return err == nil
}
