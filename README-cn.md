# xray-api-bridge
[英文](README.md) | [简体中文](README-cn.md)

本项目为 Xray-core 实例的 gRPC 管理 API 提供了完整的 RESTful HTTP API 接口；
它充当一个桥梁，将 HTTP 请求转换为 gRPC 调用，允许像 Caddy 这样的服务与 Xray 进行交互，而无需原生的 gRPC 客户端。

## 功能
- **完整的 REST API**：包含当前 Xray gRPC 管理 API 的所有端点；
- **实时订阅**：通过当前运行的 Xray 服务端以及必要的订阅配置，生成完整的订阅链接；
- **完整的后端**：你可以将本项目作为后端，使用任何语言开发自己的前端面板/仪表盘/桌面小部件等；
- 环境变量支持机密，如 Docker/Podman secrets 等；
- 自动跟踪上游更新并构建镜像；
- 提供 Docker/Podman build/compose 配置；
- 提供 compose 前置 caddy 嵌套配置；

## 项目结构
```shell
.
├── apiserver
├── bridge
├── xrayapi                           # 以上为项目源码
├── templates
│   └── subscription.jsonc.template   # 订阅配置模板
├── .env.warp.example                 # 模板替换定义
├── compose.yaml                      # 容器编排配置 - 子服务运行
├── compose.override.yaml             # 容器覆盖配置 - 独立运行
├── Dockerfile                        # 镜像生成配置
├── API.md                            # REAT API 端点文档
├── go.mod
├── go.sum
├── main.go
├── README-cn.md
└── README.md
```

## 用法
1. 自行安装 Docker 或者 Podman（推荐），也可以选择本机运行；
2. 推荐 [caddy-services](https://github.com/Lanrenbang/caddy-services)，与本项目完美搭配；
自行安装其他版本的 Caddy 或者 Ningx 也可，但相应配置需自己实现。
3. 推荐 [xray-services](https://github.com/Lanrenbang/xray-services)，与本项目完美搭配；
你可以使用现有的 xray 参考本项目相应改造，主要是配置模板的编写；
本项目的 [订阅端点配置](templates/subscription.jsonc.template) 依赖多个 `xray-services` 中的变量定义，如果你使用的不是该版本，需要将其自行配置到当前 `.env.warp` 中；
4. 克隆本仓库：
```shell
git clone https://github.com/Lanrenbang/xray-api-bridge.git
```
> 也可下载 [Releases](https://github.com/Lanrenbang/xray-api-bridge.git/releases) 档案
5. 复制或更名 [.env.warp.example](.env.warp.example) 为 `.env.warp`，按需修改必要内容；
6. 参考内部注释按需修改 [compose.yaml](compose.yaml)/[compose.override.yaml](compose.override.yaml)；
7. 将上一步配置为机密的信息加入密钥：
```shell
echo -n "some secret" | docker secret create <secret_name> - 
echo -n "some secret" | podman secret create <secret_name> - 
```
> 或者直接运行 `docker/podman secret create <secret_name>`，然后根据提示输入密钥；

> **注意：**`<secret_name>` 必须在 `.env.warp`、`compose.yaml` 相关文件中保持一致！
8. 进入根目录后，启动容器服务：
```shell
docker compose up -d
podman compose up -d
```
> 提示：
>   - 如果前置 caddy，本服务将作为子服务启动，这里无需操作，具体查看 [caddy-services](https://github.com/Lanrenbang/caddy-services)

## 其他
- 完整的 [API 文档参考](API.md)
- 如果不希望使用容器，在本机环境运行本项目，请参考 [本机运行指南](https://github.com/Lanrenbang/xray-services/blob/main/systemd/README.md)；
- 关于容器健康检查，请参考 [HEALTHCHECK 说明](https://github.com/Lanrenbang/caddy-services/blob/main/HEALTHCHECK.md)

## 相关项目
- [caddy-services](https://github.com/Lanrenbang/caddy-services)
- [xray-services](https://github.com/Lanrenbang/xray-services)

## 鸣谢
- [go-chi](https://github.com/go-chi/chi/v5)，用于路由和中间件的 Web 框架；
- [grpc](https://google.golang.org/grpc)，用于与 Xray-core 通信的 gRPC 客户端；
- [Xray-core](https://github.com/xtls/xray-core)，用于配置结构和 gRPC 定义；
- [VMessAEAD/VLESS 分享链接标准提案](https://github.com/XTLS/Xray-core/discussions/716)，订阅生成标准；
- [五合一配置](https://github.com/XTLS/Xray-core/discussions/4118)，订阅分享配置参考。

## 通过捐赠支持我
[![BuyMeACoffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-ffdd00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/bobbynona) [![Ko-Fi](https://img.shields.io/badge/Ko--fi-F16061?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/bobbynona) [![USDT(TRC20)/Tether](https://img.shields.io/badge/Tether-168363?style=for-the-badge&logo=tether&logoColor=white)](https://github.com/bobbynona/bobbynona/blob/c9f5b7482b4a951bd40a5f4284df41c0627724b8/USDT-TRC20.md) [![Litecoin](https://img.shields.io/badge/Litecoin-A6A9AA?style=for-the-badge&logo=litecoin&logoColor=white)](https://github.com/bobbynona/bobbynona/blob/c9f5b7482b4a951bd40a5f4284df41c0627724b8/Litecoin.md)

## 许可
本项目按照 `LICENSE` 文件中的条款进行分发。
