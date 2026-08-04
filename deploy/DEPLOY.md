# TFT Copilot 云服务器部署指南

四条路径，按服务器条件选择。**路径 0 最省事**（本地一条命令）；路径 A 手动分步（免 Docker、免 Go 环境）。

---

## 路径 0：本地一条命令发布（`make publish`，最省事）

开发机上打包 + 上传 + 远端安装全自动：

```bash
make publish SERVER=root@<服务器IP>
```

等价于依次执行：`make release` → `scp` 上传 → 远端解压 → `sudo ./install.sh`。

前置条件（一次性）：
1. **ssh 免密**：`ssh-copy-id root@<服务器IP>`
2. **首次部署**：先在服务器配好 LLM Key（`sudo ./install.sh` 首次会生成 `/opt/tft-copilot/.env` 模板并退出，`vi` 填入 key 后再跑一次即可）
3. 之后每次 `make publish` 全自动：`.env` 保留，二进制原子热替换，服务自动重启

---

## 路径 A：release 包 + 一键安装（手动分步）

服务器只需 Linux（x86_64 或 ARM64），无需 Docker、无需 Go。

### 1. 本地打包（开发机）

```bash
make release    # = ./deploy/build-release.sh
# 产出 dist/tft-copilot-<版本>-<日期>.tar.gz（含双架构二进制 + 数据 + 安装脚本）
```

### 2. 上传服务器

```bash
scp dist/tft-copilot-*.tar.gz user@<服务器IP>:~/
```

### 3. 服务器上一键部署

```bash
tar -xzf tft-copilot-*.tar.gz && cd tft-copilot-release
cp .env.example .env && vi .env      # 填入 LLM API Key（必填）
sudo ./install.sh                     # 自动识别架构 → 安装 → systemd 启动 → 健康检查
```

完成。访问 `http://<服务器IP>:8080/` 即为 Web 页面。

> install.sh 细节：`.env` 优先级 = `/opt/tft-copilot/.env`（升级保留）> 发布包内已配置的 `.env` > 模板；
> 二进制原子替换（服务运行中升级不报 Text file busy）；无 curl/wget 时跳过健康检查而非报错。

### 运维命令

```bash
systemctl status tft-copilot          # 状态
systemctl restart tft-copilot         # 重启（改 .env 后）
journalctl -u tft-copilot -f          # 实时日志
curl localhost:8080/v1/tft/health     # 健康/版本检查
```

### 升级

重新打包上传，解压后 `sudo ./install.sh`（会保留原 `.env`，仅替换二进制与数据）。

---

## 路径 B：Docker Compose（服务器已装 Docker）

```bash
git clone https://github.com/sagerlabs/awesome.git && cd awesome
git checkout dev
cp deploy/.env.example .env && vi .env   # 填入 LLM API Key
docker compose up -d --build
docker compose ps                         # 确认 healthy
```

镜像为多阶段构建（静态二进制 + Alpine + 非 root），数据已打进镜像，内置 healthcheck。

## 路径 C：本地预构建镜像直推（本地有 Docker，服务器无 Go/不想构建）

```bash
# 本地
make docker-build
docker save tft-copilot:latest | gzip > tft-copilot-image.tar.gz
scp tft-copilot-image.tar.gz user@<服务器IP>:~/

# 服务器
gunzip -c tft-copilot-image.tar.gz | docker load
cp deploy/.env.example .env && vi .env
docker compose up -d    # compose 里 image 已是 tft-copilot:latest，可跳过 --build
```

---

## 公网访问与安全

1. **安全组**：云控制台放行 `8080`（或只放行 80/443 走反代）。
2. **Nginx 反代（可选，推荐绑定域名 + HTTPS）**：

```nginx
server {
    listen 80;
    server_name tft.example.com;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        # SSE 流式必需
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 300s;
    }
}
```

3. **限流**：公网暴露建议在 `.env` 开 `RATE_LIMIT_RPS=5 RATE_LIMIT_BURST=10`。
4. **验证版本**：`curl http://<域名>/v1/tft/health` 返回的 `version`/`git_commit` 即当前线上版本。

## 常见问题

| 现象 | 排查 |
|---|---|
| 启动即退出 | `journalctl -u tft-copilot -n 50`：多为 `.env` 缺 API key 或 provider 不匹配 |
| health 返回 503 degraded | 数据未加载：确认 `/opt/tft-copilot/metadata` 与 `tft/knowledge` 存在（install.sh 已处理） |
| 页面能开但回答报错 | LLM key 无效/欠费；日志里看 `llm_refine` 节点错误 |
| 外网访问不到 | 安全组未放行；或云厂商另有防火墙（如阿里云 iptables 前置） |
