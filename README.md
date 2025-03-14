# ProxyGate 
ProxyPivot is a simple yet powerful Golang-based HTTP proxy server that routes requests through a pool of provided proxy servers. It supports both HTTP and SOCKS5 proxies, offering flexible and secure request forwarding.

## Features

- **HTTP Proxy Server**: Offers a simple HTTP proxy interface.
- **Proxy Pool**: Reads and parses a list of proxies from a file, supporting various formats.
- **Random Proxy Selection**: Randomly selects a proxy from the pool for each request.
- **Basic Authentication**: Secures the proxy server with a username and password.
- **Logging**: Logs each request and the selected proxy for easy debugging.

## ToDo:
- fix proxy formats
- handle bad proxies
- add blacklists
- more flexible proxy rotation logic

## Getting Started

### Prerequisites

- Go 1.23 or newer
- A list of proxy servers in `proxy_list.txt` format

### Installation

#### Build from source

1. **Clone the Repository**

   ```bash
   git clone https://github.com/Rebel028/proxygate.git
   cd proxygate
   ```

2. **Build the Project**

   ```bash
   go build -o proxygate
   ```

3. **Run the Server**

   Supply basic authentication credentials via command-line flags or environment variables:

   ```bash
   ./proxygate -user yourUsername -pass yourPassword
   ```

   Or set environment variables:

   ```bash
   export PROXY_USER=yourUsername
   export PROXY_PASS=yourPassword
   ./proxygate
   ```

#### Docker

```bash
docker run -d -p 8080:8080 -e PROXY_USER=yourUsername -e PROXY_PASS=yourPassword -v $(pwd)/proxy_list.txt:/app/proxy_list.txt --name proxygate ghcr.io/rebel028/proxygate:latest
```

### Usage

- **Proxy File Format**

  The `proxy_list.txt` file should include proxies in the following formats:

  ```
  http://ip:port
  http://username:password@ip:port
  https://ip:port
  https://username:password@ip:port
  socks://ip:port
  socks://username:password@ip:port
  socks5://ip:port
  socks5://username:password@ip:port
  ```

- **Access the Proxy**

  Use any HTTP client to send requests through the proxy server running on `localhost:8080`, e.g., with `curl`:

  ```bash
  curl -x http://yourUsername:yourPassword@localhost:8080 http://ipinfo.io
  ```
---
**Note**: Ensure that your proxy servers are correctly listed in `proxy_list.txt` and reachable from your network.


## Configuration

- **Command-line Flags**:
    - `-user`: Username for basic authentication
    - `-pass`: Password for basic authentication

- **Environment Variables**:
    - `PROXY_USER`: Alternative way to set the username
    - `PROXY_PASS`: Alternative way to set the password

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

