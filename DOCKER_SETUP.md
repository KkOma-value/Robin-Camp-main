# Docker 部署指南

本指南将帮助你使用 Docker 快速启动 Movies API 服务。

## 前置要求

确保已安装以下软件：

- **Docker** (版本 20.10+)
- **Docker Compose** (版本 2.0+)

验证安装：
```bash
docker --version
docker compose version
```

---

## 快速启动（3 步搞定）

### 1. 配置环境变量

复制示例配置文件并填写必要的值：

```bash
cp .env.example .env
```

编辑 `.env` 文件，确保以下变量已设置：

```env
# 服务配置
PORT=8080
BASE_URL=http://127.0.0.1:8080

# 认证密钥（请修改为你的密钥）
AUTH_TOKEN=my-secret-token-12345

# 数据库配置（Docker 环境使用以下配置）
DB_URL=movieuser:moviepass@tcp(mysql:3306)/movies?parseTime=true

# 票房 API 配置
BOXOFFICE_URL=https://m1.apifoxmock.com/m1/7149601-6873494-default
BOXOFFICE_API_KEY=0B4nmUwMPBphsKDr_u9HX
```

### 2. 启动所有服务

使用 Make 命令（推荐）：

```bash
make docker-up
```

或直接使用 Docker Compose：

```bash
docker compose up -d --build
```

**启动过程说明：**
- 🗄️ 启动 MySQL 8.0 数据库
- ⏳ 等待数据库健康检查通过
- 📋 自动执行数据库迁移（创建表结构）
- 🚀 启动 API 服务

### 3. 验证服务状态

**检查容器运行状态：**
```bash
docker compose ps
```

预期输出：
```
NAME                       SERVICE       STATUS         PORTS
robin-camp-main-api-1      api           running        0.0.0.0:8080->8080/tcp
robin-camp-main-mysql-1    mysql         running        0.0.0.0:3307->3306/tcp
robin-camp-main-migrations-1  migrations  exited (0)
```

> **注意**：MySQL 端口映射为 `3307:3306`，避免与本地 MySQL 服务冲突。

**测试健康检查：**
```bash
curl http://localhost:8080/healthz
```

预期返回：`ok`

---

## 查看日志

### 查看所有服务日志
```bash
docker compose logs -f
```

### 查看特定服务日志
```bash
# API 服务日志
docker compose logs -f api

# MySQL 日志
docker compose logs -f mysql

# 数据库迁移日志
docker compose logs migrations
```

---

## 运行端到端测试

在服务启动后，运行自动化测试：

```bash
make test-e2e
```

或直接执行脚本：

```bash
./e2e-test.sh
```

**Windows 用户可使用 PowerShell 测试脚本：**
```powershell
.\test-api.ps1
```

---

## 常用操作

### 停止服务
```bash
make docker-down
```

或：
```bash
docker compose down
```

### 停止并删除数据卷（清空数据库）
```bash
docker compose down -v
```

### 重新构建镜像
```bash
docker compose build --no-cache
```

### 重启服务
```bash
docker compose restart
```

### 进入 API 容器
```bash
docker compose exec api sh
```

### 进入 MySQL 容器
```bash
docker compose exec mysql mysql -u movieuser -pmoviepass movies
```

---

## API 使用示例

### 1. 创建电影

```bash
curl -X POST http://localhost:8080/movies \
  -H "Authorization: Bearer my-secret-token-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Inception",
    "releaseDate": "2010-07-16",
    "genre": "Sci-Fi",
    "distributor": "Warner Bros. Pictures",
    "budget": 160000000,
    "mpaRating": "PG-13"
  }'
```

### 2. 查询电影列表

```bash
# 获取所有电影
curl http://localhost:8080/movies

# 按年份筛选
curl "http://localhost:8080/movies?year=2010"

# 按类型筛选
curl "http://localhost:8080/movies?genre=Sci-Fi"

# 关键词搜索
curl "http://localhost:8080/movies?q=Inception"

# 分页查询
curl "http://localhost:8080/movies?limit=10"
```

### 3. 提交评分

```bash
curl -X POST "http://localhost:8080/movies/Inception/ratings" \
  -H "X-Rater-Id: user123" \
  -H "Content-Type: application/json" \
  -d '{"rating": 4.5}'
```

### 4. 查询评分汇总

```bash
curl "http://localhost:8080/movies/Inception/rating"
```

---

## 故障排查

### 问题 1：端口被占用

**错误信息：**
```
Error starting userland proxy: listen tcp4 0.0.0.0:8080: bind: address already in use
```

**解决方法：**

1. 检查占用端口的进程：
   ```bash
   # Linux/Mac
   lsof -i :8080
   
   # Windows
   netstat -ano | findstr :8080
   ```

2. 修改 `.env` 文件中的 PORT 或停止占用端口的程序

### 问题 2：数据库连接失败

**错误信息：**
```
failed to connect to database
```

**解决方法：**

1. 检查 MySQL 容器是否健康：
   ```bash
   docker compose ps mysql
   ```

2. 查看 MySQL 日志：
   ```bash
   docker compose logs mysql
   ```

3. 确保 `.env` 中的 `DB_URL` 配置正确

### 问题 3：数据库迁移失败

**解决方法：**

1. 查看迁移日志：
   ```bash
   docker compose logs migrations
   ```

2. 手动重新执行迁移：
   ```bash
   docker compose up migrations
   ```

3. 如需重置数据库：
   ```bash
   docker compose down -v
   docker compose up -d
   ```

### 问题 4：API 服务无响应

**解决方法：**

1. 检查 API 容器状态：
   ```bash
   docker compose ps api
   ```

2. 查看 API 日志：
   ```bash
   docker compose logs api
   ```

3. 确认环境变量配置：
   ```bash
   docker compose exec api env | grep -E 'PORT|AUTH_TOKEN|DB_URL|BOXOFFICE'
   ```

### 问题 5：健康检查失败

**解决方法：**

1. 手动测试健康检查：
   ```bash
   docker compose exec api wget -O- http://localhost:8080/healthz
   ```

2. 检查数据库连接：
   ```bash
   docker compose exec mysql mysqladmin ping -h localhost
   ```

---

## 容器架构说明

### 服务组成

1. **mysql**
   - 镜像：`mysql:8.0`
   - 端口：`3306`
   - 数据持久化：`mysql_data` volume
   - 健康检查：每 5 秒 ping 一次

2. **migrations**
   - 镜像：`gomicro/goose:3.7.0`
   - 作用：自动执行数据库迁移
   - 依赖：等待 mysql 健康后执行
   - 执行完成后自动退出

3. **api**
   - 镜像：本地构建（基于 Dockerfile）
   - 端口：`8080`
   - 依赖：等待 migrations 完成后启动
   - 健康检查：每 10 秒检查 `/healthz`

### 网络拓扑

```
┌─────────────────────────────────────┐
│         Docker Network              │
│                                     │
│  ┌─────────┐      ┌──────────┐    │
│  │  MySQL  │◄─────┤   API    │    │
│  │  :3306  │      │  :8080   │    │
│  └─────────┘      └──────────┘    │
│       ▲                             │
│       │                             │
│  ┌────┴─────┐                      │
│  │Migration │                      │
│  │ (one-off)│                      │
│  └──────────┘                      │
└─────────────────────────────────────┘
            │
            ▼
    宿主机 localhost:8080
```

---

## 生产环境建议

### 1. 安全配置

- ✅ 修改默认数据库密码
- ✅ 使用强随机 AUTH_TOKEN
- ✅ 配置防火墙规则，限制端口访问
- ✅ 启用 TLS/HTTPS

### 2. 性能优化

- 调整 MySQL 配置（my.cnf）：
  ```ini
  [mysqld]
  max_connections = 200
  innodb_buffer_pool_size = 2G
  ```

- 调整 API 服务资源限制：
  ```yaml
  api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
  ```

### 3. 监控与日志

- 使用 `docker stats` 监控资源使用
- 配置日志驱动（如 json-file 或 syslog）
- 集成 Prometheus + Grafana 监控

### 4. 备份策略

```bash
# 备份数据库
docker compose exec mysql mysqldump -u movieuser -pmoviepass movies > backup.sql

# 恢复数据库
docker compose exec -T mysql mysql -u movieuser -pmoviepass movies < backup.sql
```

---

## 开发模式

如需热重载等开发功能，可修改 `docker-compose.yml`：

```yaml
api:
  build: .
  command: ["go", "run", "cmd/server/main.go"]
  volumes:
    - .:/app
  environment:
    - CGO_ENABLED=0
```

---

## 清理资源

### 删除所有容器和镜像
```bash
docker compose down --rmi all -v
```

### 清理 Docker 系统
```bash
docker system prune -a --volumes
```

---

## 支持

遇到问题？

1. 查看 [README.md](./README.md) 了解架构设计
2. 查看 [ASSIGNMENT.md](./ASSIGNMENT.md) 了解项目需求
3. 检查日志：`docker compose logs -f`
4. 提交 Issue 或联系开发团队

---

**祝你使用愉快！🚀**
