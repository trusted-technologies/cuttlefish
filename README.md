# Cuttlefish Looking Glass

An open-source **Looking Glass** with a **master/slave** architecture, written in Go. The master provides a web UI for selecting a slave, viewing its IPv4/IPv6 addresses, running network tools (`ping`, `mtr`, `traceroute`, `iperf3`), and downloading test files.

## Features

- Web UI in the style of shadcn (Tailwind CSS).
- Master and Slave written in Go, packaged in separate Docker images.
- Streaming command output via Server-Sent Events.
- Real-time network interface traffic charts (RX/TX area charts).
- Test files: 1M, 10M, 100M, 1G, 10G, 100G (configurable).
- Ready-to-use images built via GitHub Actions and published to GHCR.

## Demo

A live Looking Glass demo is available as the ApexNodes hosting page: https://cuttlefish.apexnodes.xyz/

### Screenshots

![Slaves list](assets/slaves.png)

![Slave details](assets/slave.png)

## Quick install

The installer is interactive. It installs Docker, generates tokens, optionally sets up Nginx with SSL, and configures the master or slave for you. It reads prompts from `/dev/tty`, so it works through `curl | bash`.

```bash
sudo bash -c "$(curl -sSL https://raw.githubusercontent.com/trusted-technologies/cuttlefish/main/scripts/install.sh)"
```

## Update or uninstall

Run the same installer again and choose **Update existing installation** or **Uninstall** from the menu.

## Full installation guide

For manual Docker commands, Docker Compose examples, environment variables and building from source, see [INSTALL.md](INSTALL.md).

## Architecture

- **Master** — web server and slave registry.
- **Slave** — agent that registers with the master and executes network commands.
- Communication between master and slave uses HTTP/REST.

## License

MIT — see [LICENSE](LICENSE).
