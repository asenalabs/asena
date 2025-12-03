<p align="center">
  <img src="assets/logo.svg" alt="asena logo" width="350"/>
</p>

<p align="center">
    asena is a lightweight web server with minimal configuration.
</p>


##  🔹 About

Asena is a lightweight reverse proxy with minimal configuration. It provides basic host-based routing and load balancing out of the box.

## ✨ Features

* **Reverse Proxy** with `Host(...)` based route matching
* **Load Balancing** using round-robin algorithm
* **TLS Support** with hot-reload on certificate changes (SIGHUP)
* **HTTP Fallback** when TLS is invalid
* **Structured Logging** with Zap and log rotation via Lumberjack
* **Configuration** from YAML:
    * `asena.yaml` → **static** (read once at startup, no hot-reload)
    * `dynamic.yaml` → **dynamic** (supports hot-reload at runtime)


## 📦 Example Configuration
`dynamic.yaml`:
```yaml
http:
  routers:
    api-router:
      rule: "Host(`localhost`)"
      service: api-service

  services:
    api-service:
      load_balancer:
        algorithm: round-robin
        servers:
          - url: "http://localhost:9000"
          - url: "http://localhost:9001"
```

## 🚀 Quick Start

```bash
git clone https://github.com/asenalabs/asena
cd asena
go build -o ./bin/asena
chmod +x ./scripts/install.sh
sudo ./scripts/install.sh 
```
Copy and paste the above dynamic test configuration into the `/etc/asena/dynamic.yaml` file
```bash
sudo -u asena ./bin/asena -http-port :80
```
By default, Asena loads configuration from `asena.yaml` and `dynamic.yaml`.

## 🧪 Tests

```bash
go test ./...
```

## 📖 Roadmap

* Support for advanced routing rules (`PathPrefix`, `Method`, `Header`)
* Middleware support (auth, rate-limit, etc.)
* Metrics & observability integration
* Health checks for upstream services
