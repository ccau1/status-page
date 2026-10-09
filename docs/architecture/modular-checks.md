# Modular Check System

The platform features an extensible health-check engine where every check method is defined by a check type. All check types implement a common interface and register into a central thread-safe registry.

---

## 🧩 The `Checker` Interface

Located in `packages/core/checks/checker.go`:

```go
type Checker interface {
    // Type returns the unique identifier for this check method (e.g. "http", "tcp", "dns")
    Type() string

    // Execute performs the health check and returns a normalized CheckResult
    Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error)
}
```

---

## 🗃 Built-in Modular Checkers

### 1. `http` (`packages/core/checks/http.go`)
* **Purpose:** Validates HTTP/HTTPS REST or GraphQL endpoints.
* **Supported Parameters:**
  * `target`: URL (e.g. `https://api.acme.com/v1/health`)
  * `expected_status`: Expected HTTP response status code (defaults to `200`)
  * `expected_body`: Optional substring required in response body
  * `parameters["method"]`: HTTP method (`GET`, `POST`, `HEAD`, default `GET`)
  * `parameters["header:<name>"]`: Custom HTTP request headers (e.g. `header:Authorization`)
  * `timeout_ms`: Maximum execution duration before timing out

### 2. `tcp` (`packages/core/checks/tcp.go`)
* **Purpose:** Validates raw TCP socket handshakes (databases, caches, internal daemons).
* **Supported Parameters:**
  * `target`: Host and port string (e.g. `redis.internal:6379`, `postgres.db:5432`)
  * `timeout_ms`: Socket dial timeout

### 3. `dns` (`packages/core/checks/dns.go`)
* **Purpose:** Tests DNS resolution and record presence.
* **Supported Parameters:**
  * `target`: Domain hostname (e.g. `edge.company.com`)
  * Returns resolved IP addresses and resolution latency.

### 4. `synthetic` (`packages/core/checks/synthetic.go`)
* **Purpose:** Simulates scenarios for preview environments, staging tests, and demo dashboards without external dependencies.
* **Supported Parameters:**
  * `parameters["simulated_state"]`: `operational`, `degraded`, `outage`, or `maintenance`
  * `parameters["simulated_latency_ms"]`: Artificial delay to simulate network latency
  * `parameters["message"]`: Custom status message

### 5. `playwright` (`packages/core/checks/playwright.go`)
* **Purpose:** Executes headless browser synthetic user journeys via the stateless Playwright runner service (`packages/playwright-runner`).
* **Supported Parameters:**
  * `target`: URL to navigate to (e.g. `https://app.acme.com/login`)
  * `timeout_ms`: Navigation and scenario timeout in milliseconds
  * `parameters["scenario"]`: Scenario name (`page-load`, `api-ping`, or custom scenario, defaults to `page-load`)
  * `parameters["selector"]`: CSS/XPath selector to wait for on the page
  * `parameters["expected_text"]`: Required text content expected on the page
  * `parameters["degraded_latency_ms"]`: Latency threshold above which the check reports degraded state
  * `parameters["runner_url"]`: Optional override for the Playwright runner service endpoint

---

## 🛠 Adding a Custom Checker

To add a new modular check type (e.g., `grpc`, `ping`, `cert-expiry`):

1. **Implement the `Checker` interface:**
   ```go
   package mycheckers

   import (
       "context"
       "status-page/packages/core/checks"
       "status-page/packages/core/domain"
   )

   type PingChecker struct{}

   func (p *PingChecker) Type() string {
       return "icmp-ping"
   }

   func (p *PingChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
       // Custom ICMP ping logic...
       return domain.CheckResult{
           Success: true,
           State:   domain.StateOperational,
           Message: "Ping RTT 14ms",
       }, nil
   }
   ```

2. **Register the Checker:**
   ```go
   func init() {
       checks.Register(&PingChecker{})
   }
   ```

3. **Use in `STATUS_DEF_JSON`:**
   ```json
   {
     "id": "edge-ping",
     "feature": "edge-network",
     "type": "icmp-ping",
     "target": "10.0.0.1"
   }
   ```
