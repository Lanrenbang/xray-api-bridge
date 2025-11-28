# xray-api-bridge
[English](README.md) | [Simplified Chinese](README-cn.md)

This project provides a complete RESTful HTTP API interface for the Xray-core instance's gRPC management API.
It acts as a bridge, converting HTTP requests into gRPC calls, allowing services like Caddy to interact with Xray without needing a native gRPC client.

## Features
- **Complete REST API**: Includes all endpoints of the current Xray gRPC management API.
- **Real-time Subscription**: Generates complete subscription links based on the currently running Xray server and necessary subscription configurations.
- **Complete Backend**: You can use this project as a backend to develop your own frontend panel, dashboard, desktop widget, etc.
- Support for secrets in environment variables (e.g., Docker/Podman secrets).
- Automatically tracks upstream updates and rebuilds images.
- Includes Docker/Podman build/compose configurations.
- Provides pre-configured Caddy nesting for Compose.

## Project Structure
```shell
.
├── apiserver
├── bridge
├── xrayapi                           # Source code above
├── templates
│   └── subscription.jsonc.template   # Subscription configuration template
├── .env.warp.example                 # Template substitution definitions
├── compose.yaml                      # Container orchestration - Sub-service execution
├── compose.override.yaml             # Container override - Standalone execution
├── Dockerfile                        # Image generation config
├── API.md                            # REST API endpoint documentation
├── go.mod
├── go.sum
├── main.go
├── README-cn.md
└── README.md
```

## Usage
1. Install Docker or Podman (recommended), or choose to run natively.
2. [caddy-services](https://github.com/Lanrenbang/caddy-services) is recommended as the perfect companion for this project.
   You can also install other versions of Caddy or Nginx yourself, but you will need to implement the corresponding configurations manually.
3. [xray-services](https://github.com/Lanrenbang/xray-services) is recommended as the perfect companion for this project.
   You can also use an existing Xray setup and modify it by referring to this project, mainly regarding the configuration templates.
   The [Subscription Endpoint Configuration](templates/subscription.jsonc.template) in this project depends on multiple variable definitions from `xray-services`. If you are not using that version, you will need to manually configure them in the current `.env.warp`.
4. Clone this repository:
```shell
git clone https://github.com/Lanrenbang/xray-api-bridge.git
```
> You can also download the [Releases](https://github.com/Lanrenbang/xray-api-bridge.git/releases) archive.
5. Copy or rename [.env.warp.example](.env.warp.example) to `.env.warp` and modify the necessary content as needed.
6. Refer to the internal comments to modify [compose.yaml](compose.yaml)/[compose.override.yaml](compose.override.yaml) as required.
7. Add the information configured as secrets in the previous step to the keystore:
```shell
echo -n "some secret" | docker secret create <secret_name> - 
echo -n "some secret" | podman secret create <secret_name> - 
```
> Or run `docker/podman secret create <secret_name>` directly and enter the secret when prompted.

> **Note:** `<secret_name>` must match the entries in `.env.warp` and `compose.yaml`.
8. Enter the root directory and start the container service:
```shell
docker compose up -d
podman compose up -d
```
> **Tip:**
> - If Caddy is used as a frontend, this service will start as a sub-service. No action is needed here; please refer to [caddy-services](https://github.com/Lanrenbang/caddy-services) for details.

## Others
- Full [API Documentation Reference](API.md).
- If you do not wish to use containers and prefer to run this project natively, please refer to the [Native Execution Guide](https://github.com/Lanrenbang/xray-services/blob/main/systemd/README.md).
- For container health checks, please refer to the [HEALTHCHECK Guide](https://github.com/Lanrenbang/caddy-services/blob/main/HEALTHCHECK.md).

## Related Projects
- [caddy-services](https://github.com/Lanrenbang/caddy-services)
- [xray-services](https://github.com/Lanrenbang/xray-services)

## Credits
- [go-chi](https://github.com/go-chi/chi/v5), Web framework used for routing and middlewares.
- [grpc](https://google.golang.org/grpc), gRPC client used to communicate with Xray-core.
- [Xray-core](https://github.com/xtls/xray-core), Used for configuration structures and gRPC definitions.
- [VMessAEAD/VLESS Sharing Standard Proposal](https://github.com/XTLS/Xray-core/discussions/716), Standard for subscription generation.
- [5-in-1 Configuration](https://github.com/XTLS/Xray-core/discussions/4118), Subscription sharing configuration reference.

## Support Me
[![BuyMeACoffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-ffdd00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/bobbynona) [![Ko-Fi](https://img.shields.io/badge/Ko--fi-F16061?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/bobbynona) [![USDT(TRC20)/Tether](https://img.shields.io/badge/Tether-168363?style=for-the-badge&logo=tether&logoColor=white)](https://github.com/Lanrenbang/.github/blob/5b06b0b2d0b8e4ce532c1c37c72115dd98d7d849/custom/USDT-TRC20.md) [![Litecoin](https://img.shields.io/badge/Litecoin-A6A9AA?style=for-the-badge&logo=litecoin&logoColor=white)](https://github.com/Lanrenbang/.github/blob/5b06b0b2d0b8e4ce532c1c37c72115dd98d7d849/custom/Litecoin.md)

## License
This project is distributed under the terms of the `LICENSE` file.
```

