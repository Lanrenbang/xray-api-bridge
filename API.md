## API 端点

以下是此桥接服务提供的 REST API 端点的完整列表，按其对应的 Xray gRPC 服务分类。

<details>
<summary>自定义端点</summary>

### 自定义端点

*   **GET /status**
    *   **描述:** 检查 API 桥接服务是否正在运行。
    *   **`curl` 示例:** 
        ```bash
        curl http://localhost:8081/status
        ```
    *   **响应 (成功):** 
        ```json
        {
            "success": true,
            "message": "Xray API Bridge is running!"
        }
        ```
*   **GET /generateSubLinks**
    *   **描述:** 校验用户 UUID 并生成加密的订阅 URL。此端点作为 `/subscription` 的入口，将 UUID 加密为 token 嵌入返回的 URL 中。
    *   **环境变量依赖:**
        *   `XRAY_API_BRIDGE_SUBS_AUTHSECRET` (必须): Token 加密密钥。
        *   `XRAY_API_BRIDGE_SUBS_URL` (必须): 订阅服务的基础 URL。
        *   `XRAY_API_BRIDGE_SUBS_SUPERKEY` (可选): 超级密钥，用于获取所有用户的订阅。
    *   **查询参数:**
        *   `uuid` (必须): 一个或多个用户的 ID，以逗号分隔。如果提供的值与 `XRAY_API_BRIDGE_SUBS_SUPERKEY` 环境变量匹配，则返回包含所有用户链接的订阅 URL。
    *   **`curl` 示例:** 
        ```bash
        # 获取单个用户的订阅 URL
        curl "http://localhost:8081/generateSubLinks?uuid=c6a5b2a0-5a5a-4a5a-a5a5-a5a5a5a5a5a5"

        # 获取多个用户的订阅 URL
        curl "http://localhost:8081/generateSubLinks?uuid=user-id-1,user-id-2"

        # 使用超级密钥获取所有链接
        curl "http://localhost:8081/generateSubLinks?uuid=YOUR_SUPER_KEY"
        ```
    *   **响应 (成功):** 
        ```json
        {
            "success": true,
            "data": "https://your-domain.com/subscription?token=kX9vR2mN8pQ1wZ5t..."
        }
        ```
    *   **响应 (错误 - UUID 无效):** 
        ```json
        {
            "success": false,
            "error": "The provided uuid does not match any active inbound client."
        }
        ```
*   **GET /subscription**
    *   **描述:** 根据加密 token 中携带的用户 UUID 生成订阅链接。返回 Base64 编码的通用 URI 格式订阅内容（多条链接以换行符分隔）。
    *   **查询参数:**
        *   `token` (必须): 由 `/generateSubLinks` 端点生成的加密 token，内含用户 UUID 信息。
    *   **`curl` 示例:** 
        ```bash
        # 直接使用 /generateSubLinks 返回的完整 URL
        curl "https://your-domain.com/subscription?token=kX9vR2mN8pQ1wZ5t..."
        ```
    *   **响应 (成功):** 
        ```
        dmxlc3M6Ly91dWlkQGFkZHJlc3M6cG9ydD9zZWN1cml0eT1yZWFsaXR5...
        ```
        > 响应为 `text/plain` 格式的 Base64 编码文本。解码后为多行 vless/vmess URI。
</details>
<details>
<summary>StatsService (统计服务)</summary>

### StatsService (统计服务)

*   **GET /stats/sys**
    *   **描述:** 检索 Xray 系统运行时统计信息。
    *   **`curl` 示例:** 
        ```bash
        curl -i http://localhost:8081/stats/sys
        ```
    *   **响应:** 
        ```json
        {"success":true,"data":{"NumGoroutine":13,"NumGC":75,"Alloc":96263480,"TotalAlloc":844702808,"Sys":372836696,"Mallocs":7920246,"Frees":5959066,"LiveObjects":1961180,"PauseTotalNs":643993573,"Uptime":6906}}
        ```

*   **GET /stats**
    *   **描述:** 检索指定名称的统计计数器的值。
    *   **`curl` 示例:** 
        ```bash
        curl -i 'http://localhost:8081/stats?name=inbound>>>in_raw_reality>>>traffic>>>uplink'
        ```
    *   **响应 (成功, 值为 0):** 
        ```json
        {"success":true,"data":{"name":"inbound>>>in_raw_reality>>>traffic>>>uplink"}}
        ```

*   **GET /stats/query**
    *   **描述:** 查询与给定模式匹配的统计计数器。
    *   **`curl` 示例:** 
        ```bash
        curl -i 'http://localhost:8081/stats/query?pattern=inbound>>>in_raw_reality'
        ```
    *   **响应 (成功, 值为 0):** 
        ```json
        {"success":true,"data":[{"name":"inbound>>>in_raw_reality>>>traffic>>>uplink"},{"name":"inbound>>>in_raw_reality>>>traffic>>>downlink"}]}
        ```

*   **GET /stats/online**
    *   **描述:** 检索指定用户的在线会话数。
    *   **`curl` 示例:** 
        ```bash
        curl -i 'http://localhost:8081/stats/online?name=raw_pc@xray.com'
        ```
    *   **响应 (错误 - 用户不在线):** 
        ```json
        {"success":false,"error":"Failed to get online stat: rpc error: code = NotFound desc = raw_pc@xray.com not found."}
        ```

*   **GET /stats/online/iplist**
    *   **描述:** 检索指定用户的在线 IP 地址和访问时间。
    *   **`curl` 示例:** 
        ```bash
        curl -i 'http://localhost:8081/stats/online/iplist?name=raw_pc@xray.com'
        ```
    *   **响应 (错误 - 用户不在线):** 
        ```json
        {"success":false,"error":"Failed to get online IP list: rpc error: code = NotFound desc = raw_pc@xray.com not found."}
        ```
</details>
<details>
<summary>HandlerService (代理处理器服务)</summary>

### HandlerService (代理处理器服务)

*   **GET /inbound**
    *   **描述:** 列出所有入站代理配置，并提供解码后的人类可读设置。
    *   **`curl` 示例:** 
        ```bash
        curl -i http://localhost:8081/inbound
        ```

*   **POST /inbound**
    *   **描述:** 使用用户友好的 JSON 格式添 加新的入站代理配置。
    *   **`curl` 示例:** 
        ```bash
        curl -X POST -H "Content-Type: application/json" -d 
        '{'
            '"tag": "test_inbound_from_api",'
            '"listen": "/dev/shm/test_inbound.sock,0666",'
            '"protocol": "vless",'
            '"settings": {
                "clients": [
                    {
                        "id": "c6a5b2a0-5a5a-4a5a-a5a5-a5a5a5a5a5a5",
                        "email": "test_user@example.com",
                        "flow": "xtls-rprx-vision"
                    }
                ],
                "decryption": "none"
            },
            "streamSettings": {
                "network": "tcp",
                "security": "none"
            }
        }' http://localhost:8081/inbound
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Inbound 'test_inbound_from_api' added successfully"}
        ```

*   **DELETE /inbound/{tag}**
    *   **描述:** 按标签删除现有入站代理。
    *   **`curl` 示例:** 
        ```bash
        curl -X DELETE http://localhost:8081/inbound/test_inbound_from_api
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Inbound 'test_inbound_from_api' removed successfully"}
        ```

*   **PUT /inbound/{tag}**
    *   **描述:** 通过添加用户来更改入站代理的配置。
    *   **`curl` 示例:** 
        ```bash
        curl -i -X PUT -H "Content-Type: application/json" \
        -d '{"id": "c6a5b2a0-5a5a-4a5a-a5a5-a5a5a5a5a5a5", "email": "test_user@example.com", "level": 0, "flow": "xtls-rprx-vision"}' \
        http://localhost:8081/inbound/in_raw_reality
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Inbound altered successfully"}
        ```

*   **GET /inbound/{tag}/users**
    *   **描述:** 检索指定入站代理下的用户列表。
    *   **`curl` 示例:** 
        ```bash
        curl http://localhost:8081/inbound/test_inbound_from_api/users
        ```
    *   **响应:** 
        ```json
        {"success":true,"data":[{"flow":"xtls-rprx-vision","id":"c6a5b2a0-5a5a-4a5a-a5a5-a5a5a5a5a5a4","email":"test_user1@example.com","level":0},{"flow":"xtls-rprx-vision","id":"c6a5b2a0-5a5a-4a5a-a5a5-a5a5a5a5a5a5","email":"test_user2@example.com","level":0}]}
        ```

*   **GET /inbound/{tag}/users/count**
    *   **描述:** 检索指定入站代理下的用户数量。
    *   **`curl` 示例:** 
        ```bash
        curl http://localhost:8081/inbound/in_raw_reality/users/count
        ```
    *   **响应:** 
        ```json
        {"success":true,"data":{"count":4}}
        ```

*   **POST /inbound/{tag}/users**
    *   **描述:** 向指定的入站代理添加一个或多个用户。
    *   **`curl` 示例:** 
        ```bash
        curl -X POST -H "Content-Type: application/json" \
        -d '[{"id": "c6a5b2a0-5a5a-4a5a-a5a5-a5a5a5a5a5a5", "email": "multi_user1@example.com", "level": 0, "flow": "xtls-rprx-vision"}]' \
        http://localhost:8081/inbound/in_raw_reality/users
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"1 users added to inbound 'in_raw_reality'"}
        ```

*   **DELETE /inbound/{tag}/users**
    *   **描述:** 从指定的入站代理中删除一个或多个用户。
    *   **`curl` 示例:** 
        ```bash
        curl -X DELETE -H "Content-Type: application/json" \
        -d '{"emails": ["multi_user1@example.com"]}' \
        http://localhost:8081/inbound/in_raw_reality/users
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"1 users removed from inbound 'in_raw_reality'"}
        ```

*   **GET /outbound**
    *   **描述:** 列出所有出站代理配置。
    *   **`curl` 示例:** 
        ```bash
        curl http://localhost:8081/outbound
        ```
    *   **响应:** 
        ```json
        {"success":true,"data":[{"protocol":"freedom","sendThrough":null,"tag":"direct","settings":{},"streamSettings":null,"proxySettings":null,"mux":null,"targetStrategy":""},{"protocol":"blackhole","sendThrough":null,"tag":"block","settings":{},"streamSettings":null,"proxySettings":null,"mux":null,"targetStrategy":""}]}
        ```

*   **POST /outbound**
    *   **描述:** 添加新的出站代理配置。
    *   **`curl` 示例:** 
        ```bash
        curl -X POST -H "Content-Type: application/json" -d '{"tag": "test_outbound", "protocol": "freedom", "settings": {}}' http://localhost:8081/outbound
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Outbound 'test_outbound' added successfully"}
        ```

*   **DELETE /outbound/{tag}**
    *   **描述:** 按标签删除现有出站代理。
    *   **`curl` 示例:** 
        ```bash
        curl -X DELETE http://localhost:8081/outbound/test_outbound
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Outbound removed successfully"}
        ```
</details>
<details>
<summary>RoutingService (路由服务)</summary>

### RoutingService (路由服务)

*   **POST /routing/rule**
    *   **描述:** 使用用户友好的 JSON 格式添加新的路由规则。
    *   **`curl` 示例:** 
        ```bash
        curl -X POST -H "Content-Type: application/json" -d 
        '{'
            '"ruleTag": "test_block_google",'
            '"domain": ["google.com"],
            '"outboundTag": "block"
        }' http://localhost:8081/routing/rule
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Routing rule added successfully"}
        ```

*   **DELETE /routing/rule/{tag}**
    *   **描述:** 按标签删除路由规则。
    *   **`curl` 示例:** 
        ```bash
        curl -X DELETE http://localhost:8081/routing/rule/test_block_google
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Routing rule removed successfully"}
        ```

*   **GET /routing/balancer/{tag}**
    *   **描述:** 检索指定负载均衡器的统计信息。
    *   **`curl` 示例:** 
        ```bash
        curl -i http://localhost:8081/routing/balancer/mybalancer
        ```
    *   **响应 (错误):** 
        ```json
        {"success":false,"error":"Failed to get balancer stats: rpc error: code = Unknown desc = app/router: cannot find tag"}
        ```

*   **POST /routing/balancer/{tag}/choose**
    *   **描述:** 强制负载均衡器选择指定的出站标签。

*   **POST /routing/blockip**
    *   **描述:** 添加源 IP 阻塞路由规则。
</details>
<details>
<summary>LoggerService (日志服务)</summary>

### LoggerService (日志服务)

*   **POST /logger/restart**
    *   **描述:** 重启 Xray 内置的日志记录器。
    *   **`curl` 示例:** 
        ```bash
        curl -X POST http://localhost:8081/logger/restart
        ```
    *   **响应:** 
        ```json
        {"success":true,"message":"Logger restarted successfully"}
        ```
</details>

