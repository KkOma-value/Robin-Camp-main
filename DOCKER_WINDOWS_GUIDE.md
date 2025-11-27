# Windows Docker 启动指南

本指南专为 **Windows 用户** 编写，帮助你在 Windows 环境下使用 Docker 快速启动 Movies API 服务。

---

## 📋 前置要求

### 1. 安装 Docker Desktop for Windows

1. 下载 [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop/)
2. 安装并启动 Docker Desktop
3. 确保 Docker 服务正在运行（系统托盘图标为绿色）

### 2. 验证安装

打开 PowerShell，运行以下命令：

```powershell
docker --version
docker compose version
```

预期输出类似：

```
Docker version 24.0.x, build xxxxxxx
Docker Compose version v2.x.x
```

---

## 🚀 快速启动（4 步搞定）

### 第 1 步：配置环境变量

```powershell
# 进入项目目录
cd D:\Robin-Camp-main

# 复制环境变量配置文件
Copy-Item .env.example .env
```

编辑 `.env` 文件，确保以下配置：

```env
# 服务配置
PORT=8080
BASE_URL=http://127.0.0.1:8080

# 认证密钥（请修改为你的密钥）
AUTH_TOKEN=my-secret-token-12345

# 数据库配置（Docker 环境使用以下配置，无需修改）
DB_URL=movieuser:moviepass@tcp(mysql:3306)/movies?parseTime=true

# 票房 API 配置
BOXOFFICE_URL=https://m1.apifoxmock.com/m1/7149601-6873494-default
BOXOFFICE_API_KEY=0B4nmUwMPBphsKDr_u9HX
```

### 第 2 步：启动所有服务

```powershell
docker compose up -d --build
```

**启动过程说明：**

| 阶段 | 说明 |
|------|------|
| 🗄️ MySQL 启动 | 启动 MySQL 8.0 数据库容器 |
| ⏳ 健康检查 | 等待数据库完全就绪 |
| 📋 数据库迁移 | 自动创建数据表结构 |
| 🚀 API 启动 | 启动 Movies API 服务 |

### 第 3 步：验证服务状态

```powershell
# 查看容器状态
docker compose ps
```

预期输出：

```
NAME                           SERVICE      STATUS                 PORTS
robin-camp-main-api-1          api          running (healthy)      0.0.0.0:8080->8080/tcp
robin-camp-main-mysql-1        mysql        running (healthy)      0.0.0.0:3307->3306/tcp
robin-camp-main-migrations-1   migrations   exited (0)
```

> **说明**：`migrations` 服务执行完迁移后会自动退出，状态为 `exited (0)` 是正常的。

### 第 4 步：测试 API

```powershell
# 健康检查
Invoke-RestMethod http://localhost:8080/healthz
```

预期返回：`ok`

**或者运行测试脚本：**

```powershell
.\test-api.ps1
```

---

## 📝 常用 PowerShell 命令

### 服务管理

```powershell
# 启动服务
docker compose up -d

# 启动并重新构建
docker compose up -d --build

# 停止服务
docker compose down

# 停止服务并删除数据卷（清空数据库）
docker compose down -v

# 重启服务
docker compose restart

# 重启单个服务
docker compose restart api
```

### 查看日志

```powershell
# 查看所有日志
docker compose logs -f

# 查看 API 服务日志
docker compose logs -f api

# 查看 MySQL 日志
docker compose logs -f mysql

# 查看数据库迁移日志
docker compose logs migrations
```

### 进入容器

```powershell
# 进入 API 容器
docker compose exec api sh

# 进入 MySQL 容器并连接数据库
docker compose exec mysql mysql -u movieuser -pmoviepass movies
```

---

## 🔧 API 使用示例（PowerShell）

### 创建电影

```powershell
$headers = @{
    "Authorization" = "Bearer my-secret-token-12345"
    "Content-Type" = "application/json"
}

$body = @{
    title = "Inception"
    releaseDate = "2010-07-16"
    genre = "Sci-Fi"
    distributor = "Warner Bros. Pictures"
    budget = 160000000
    mpaRating = "PG-13"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/movies" -Method POST -Headers $headers -Body $body
```

### 查询电影列表

```powershell
# 获取所有电影
Invoke-RestMethod http://localhost:8080/movies

# 按年份筛选
Invoke-RestMethod "http://localhost:8080/movies?year=2010"

# 按类型筛选
Invoke-RestMethod "http://localhost:8080/movies?genre=Sci-Fi"
```

### 提交评分

```powershell
$headers = @{
    "X-Rater-Id" = "user123"
    "Content-Type" = "application/json"
}

$body = @{ rating = 4.5 } | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/movies/Inception/ratings" -Method POST -Headers $headers -Body $body
```

### 查询评分汇总

```powershell
Invoke-RestMethod "http://localhost:8080/movies/Inception/rating"
```

---

## ⚠️ 常见问题排查

### 问题 1：Docker Hub 连接超时

**错误信息：**

```
failed to fetch oauth token: dial tcp: connection timed out
```

**解决方法：**

项目已配置国内镜像源 `docker.1ms.run`，如果仍然超时，可以在 Docker Desktop 中添加镜像加速器：

1. 打开 Docker Desktop → Settings → Docker Engine
2. 添加镜像源：

```json
{
  "registry-mirrors": [
    "https://docker.1ms.run",
    "https://docker.xuanyuan.me"
  ]
}
```

3. 点击 Apply & Restart

### 问题 2：端口被占用

**错误信息：**

```
bind: Only one usage of each socket address is normally permitted
```

**解决方法：**

1. 检查占用端口的进程：

```powershell
# 检查 8080 端口
netstat -ano | findstr :8080

# 检查 3307 端口
netstat -ano | findstr :3307
```

2. 终止占用端口的进程，或修改 `docker-compose.yml` 中的端口映射

### 问题 3：数据库连接失败

**错误信息：**

```
failed to connect to database
```

**解决方法：**

1. 检查 MySQL 容器是否健康：

```powershell
docker compose ps mysql
```

2. 查看 MySQL 日志：

```powershell
docker compose logs mysql
```

3. 确保 `.env` 中的 `DB_URL` 配置正确

### 问题 4：API 服务无响应

**解决方法：**

1. 检查 API 容器状态：

```powershell
docker compose ps api
```

2. 查看 API 日志：

```powershell
docker compose logs api
```

3. 确认环境变量配置：

```powershell
docker compose exec api env
```

---

## 🔄 重置环境

如果遇到问题需要重新开始：

```powershell
# 1. 停止并删除所有容器和数据卷
docker compose down -v

# 2. 删除构建的镜像
docker compose down --rmi all -v

# 3. 清理 Docker 系统（谨慎使用）
docker system prune -a --volumes

# 4. 重新启动
docker compose up -d --build
```

---

## 📊 服务架构

```
┌─────────────────────────────────────────────────────┐
│                  Docker Network                      │
│                                                      │
│   ┌───────────┐          ┌───────────────────┐     │
│   │   MySQL   │◄─────────┤   Movies API      │     │
│   │  :3306    │          │   :8080           │     │
│   └───────────┘          └───────────────────┘     │
│        ▲                                            │
│        │                                            │
│   ┌────┴──────┐                                    │
│   │ Migration │ (执行后自动退出)                    │
│   │  (goose)  │                                    │
│   └───────────┘                                    │
└─────────────────────────────────────────────────────┘
              │                    │
              ▼                    ▼
     宿主机 localhost:3307   宿主机 localhost:8080
```

---

## 📚 相关文档

- [README.md](./README.md) - 项目技术设计文档
- [DOCKER_SETUP.md](./DOCKER_SETUP.md) - 通用 Docker 部署指南
- [ASSIGNMENT.md](./ASSIGNMENT.md) - 项目需求说明

---

**祝你使用愉快！** 🎬🍿
