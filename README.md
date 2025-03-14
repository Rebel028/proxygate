# ProxyGate 
ProxyPivot is a simple yet powerful Golang-based HTTP proxy server that routes requests through a pool of provided proxy servers. It supports both HTTP and SOCKS5 proxies, offering flexible and secure request forwarding.

## Features

- **HTTP Proxy Server**: Offers a simple HTTP proxy interface.
- **Proxy Pool**: Reads and parses a list of proxies from a file, supporting various formats.
- **Random Proxy Selection**: Randomly selects a proxy from the pool for each request.
- **Basic Authentication**: Secures the proxy server with a username and password.
- **Logging**: Logs each request and the selected proxy for easy debugging.

## Getting Started

### Prerequisites

- Go 1.23 or newer
- A list of proxy servers in `proxy_list.txt` format

### Installation

1. **Clone the Repository**

   ```bash
   git clone https://github.com/yourusername/proxypivot.git
   cd proxypivot
   ```

2. **Build the Project**

   ```bash
   go build -o proxypivot
   ```

3. **Run the Server**

   Supply basic authentication credentials via command-line flags or environment variables:

   ```bash
   ./proxypivot -user yourUsername -pass yourPassword
   ```

   Or set environment variables:

   ```bash
   export PROXY_USER=yourUsername
   export PROXY_PASS=yourPassword
   ./proxypivot
   ```

### Usage

- **Proxy File Format**

  The `proxy_list.txt` file should include proxies in the following formats:

  ```
  ip:port
  ip:port:username:password
  http://ip:port
  http://username:password@ip:port
  https://ip:port
  https://username:password@ip:port
  socks://ip:port
  socks://username:password@ip:port
  socks4://ip:port
  socks4://username:password@ip:port
  socks5://ip:port
  socks5://username:password@ip:port
  ```

- **Access the Proxy**

  Use any HTTP client to send requests through the proxy server running on `localhost:8080`, e.g., with `curl`:

  ```bash
  curl -x http://yourUsername:yourPassword@localhost:8080 http://example.com
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

