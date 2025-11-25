# Movies API - 技术设计文档

## 项目概述

这是一个电影评分 API 服务，支持电影的创建、查询、评分提交和评分聚合功能。服务在创建电影时会同步调用外部票房 API 获取票房数据，并支持灵活的搜索和分页功能。

---

## 1. 数据库选型与设计

### 1.1 数据库选型：MySQL 8.0

**选择理由：**
- **成熟稳定**：MySQL 是业界广泛使用的关系型数据库，文档完善，社区活跃
- **ACID 保证**：评分和电影数据需要强一致性，关系型数据库天然支持事务
- **查询性能**：支持复杂的条件查询和索引优化，适合本项目的多维度搜索需求
- **JSON 支持**：MySQL 8.0 对 JSON 字段的支持良好，虽然本项目未使用，但为未来扩展预留空间
- **成本考虑**：开源免费，部署简单，适合中小规模应用

### 1.2 数据库 Schema 设计

#### 表结构设计

**1. movies 表（电影主表）**
```sql
CREATE TABLE movies (
    id CHAR(26) PRIMARY KEY,              -- ULID 格式，全局唯一且有序
    title VARCHAR(255) NOT NULL UNIQUE,   -- 电影标题，唯一索引
    release_date DATE NOT NULL,           -- 发行日期
    genre VARCHAR(64) NOT NULL,           -- 类型
    distributor VARCHAR(255),             -- 发行商（可选）
    budget BIGINT,                        -- 预算（可选）
    mpa_rating VARCHAR(16),               -- 分级（可选）
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    -- 索引设计
    INDEX idx_title (title),              -- 标题查询
    INDEX idx_release_date (release_date),-- 年份筛选
    INDEX idx_genre (genre),              -- 类型筛选
    INDEX idx_distributor (distributor),  -- 发行商筛选
    INDEX idx_budget (budget),            -- 预算筛选
    INDEX idx_mpa_rating (mpa_rating)     -- 分级筛选
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**设计要点：**
- 使用 ULID 而非自增 ID，避免分布式环境下的 ID 冲突，且保持时间有序性
- title 设置 UNIQUE 约束，因为业务逻辑中通过标题查询电影
- 微秒级时间戳（TIMESTAMP(6)）提供更精确的时间记录
- 多个索引支持各种查询场景，但需平衡写入性能

**2. movie_box_office 表（票房数据）**
```sql
CREATE TABLE movie_box_office (
    movie_id CHAR(26) PRIMARY KEY,
    gross_usd BIGINT NOT NULL,            -- 全球总票房
    opening_weekend_usa BIGINT,           -- 美国首周末票房（可选）
    currency VARCHAR(8) NOT NULL,         -- 货币单位
    source VARCHAR(64) NOT NULL,          -- 数据来源
    last_reported TIMESTAMP(6) NOT NULL,  -- 上游数据更新时间
    fetched_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_movie_box_office_movie
        FOREIGN KEY (movie_id) REFERENCES movies(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**设计要点：**
- 与 movies 表 1:1 关系，movie_id 作为主键
- 级联删除确保数据一致性
- 分离票房数据便于独立更新和查询优化

**3. movie_ratings 表（评分数据）**
```sql
CREATE TABLE movie_ratings (
    movie_id CHAR(26) NOT NULL,
    rater_id VARCHAR(128) NOT NULL,
    rating DECIMAL(2,1) NOT NULL,         -- 评分值，范围 0.5-5.0，步长 0.5
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (movie_id, rater_id),     -- 复合主键实现 Upsert
    CONSTRAINT fk_movie_ratings_movie
        FOREIGN KEY (movie_id) REFERENCES movies(id)
        ON DELETE CASCADE,
    CONSTRAINT chk_rating_values
        CHECK (rating IN (0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**设计要点：**
- 复合主键 (movie_id, rater_id) 天然支持 Upsert 语义
- CHECK 约束确保评分值的有效性（MySQL 8.0+ 支持）
- 级联删除保证电影删除时相关评分也被清理

### 1.3 数据库迁移策略

使用 goose 工具管理数据库迁移：
- 版本化的 SQL 文件，易于追踪变更历史
- 支持 Up/Down 迁移，便于回滚
- docker-compose 中通过 migrations 服务自动执行

---

## 2. 后端服务选型与设计

### 2.1 技术栈选择：Go 语言

**选择理由：**
- **性能优异**：编译型语言，并发性能强，适合 I/O 密集型 API 服务
- **简洁高效**：语法简单，工程化能力强，代码可读性和可维护性好
- **标准库丰富**：net/http、context、database/sql 等标准库功能完善
- **部署简单**：编译为单一二进制文件，容器化部署无依赖
- **并发模型**：goroutine 和 channel 使并发编程更安全和直观

### 2.2 架构设计

#### 2.2.1 项目结构
```
.
├── cmd/server/          # 应用入口
├── internal/
│   ├── api/
│   │   ├── handlers/    # HTTP 处理器
│   │   └── middleware/  # 中间件（认证、日志）
│   ├── clients/
│   │   └── boxoffice/   # 票房 API 客户端
│   ├── config/          # 配置管理
│   ├── logging/         # 日志封装
│   ├── server/          # HTTP 服务器
│   └── store/           # 数据访问层
├── db/migrations/       # 数据库迁移文件
└── docker-compose.yml
```

**设计原则：**
- **清晰的分层**：handlers -> store -> database，职责明确
- **依赖注入**：通过构造函数注入依赖，便于测试和解耦
- **internal 包**：防止外部包导入，封装实现细节

#### 2.2.2 核心组件设计

**1. HTTP 路由层（chi router）**
- 选择 chi：轻量级、标准库兼容、中间件支持好
- 路由设计：
  ```go
  GET  /healthz                     -> HealthCheck
  GET  /movies                      -> List (公开)
  POST /movies                      -> Create (需 Bearer Token)
  POST /movies/{title}/ratings      -> SubmitRating (需 X-Rater-Id)
  GET  /movies/{title}/rating       -> GetAggregate (公开)
  ```

**2. 中间件设计**
- **Logger**：记录所有请求的方法、路径、状态码、耗时
- **BearerAuth**：验证 Authorization 头中的 Bearer Token
- **RequireRaterID**：验证并提取 X-Rater-Id 头

**3. 数据访问层（Store）**
- **MovieStore**：电影和票房数据的 CRUD
- **RatingStore**：评分的 Upsert 和聚合查询
- 使用 sqlx 简化数据库操作，支持 struct tag 映射

**4. 外部客户端（Box Office Client）**
- 使用 hashicorp/go-retryablehttp 实现自动重试
- 超时控制：30 秒请求超时
- 错误降级：上游失败不阻塞电影创建

#### 2.2.3 关键业务逻辑

**电影创建流程：**
1. 解析请求体，验证必填字段（title, genre, releaseDate）
2. 生成 ULID 作为电影 ID
3. 插入电影基本信息到数据库
4. **异步调用票房 API**（实际实现为同步，但设计为可异步）
   - 成功：合并票房数据，用户提供的字段优先级更高
   - 失败：设置 boxOffice = null，不影响创建成功
5. 返回 201 Created + Location 头

**评分提交流程：**
1. 从 URL 路径提取电影标题
2. 从请求头提取 rater_id
3. 验证评分值是否在 [0.5, 1.0, ..., 5.0] 范围内
4. 查询电影是否存在（不存在返回 404）
5. 检查该用户是否已评分
6. 执行 Upsert 操作（INSERT ... ON DUPLICATE KEY UPDATE）
7. 返回 201（新增）或 200（更新）

**评分聚合查询：**
```sql
SELECT COALESCE(ROUND(AVG(rating), 1), 0) as average, COUNT(*) as count
FROM movie_ratings WHERE movie_id = ?
```
- 使用 ROUND(AVG(rating), 1) 确保平均值保留 1 位小数
- COALESCE 处理无评分时的默认值

**分页实现（Cursor-based）：**
- 使用 (created_at, id) 作为游标
- 优点：稳定性好，不受数据插入影响
- 查询：`WHERE (created_at > ? OR (created_at = ? AND id > ?)) ORDER BY created_at, id LIMIT ?`

### 2.3 配置管理

**环境变量驱动：**
- 所有配置通过环境变量注入，无硬编码
- 启动时验证必需配置，缺失立即失败（Fail Fast）
- 使用 `.env` 文件在本地开发，docker-compose 中通过 environment 传递

**必需配置：**
```
PORT              服务端口
AUTH_TOKEN        Bearer Token
DB_URL            数据库连接字符串
BOXOFFICE_URL     票房 API 基础 URL
BOXOFFICE_API_KEY 票房 API 密钥
```

### 2.4 错误处理

**统一错误响应格式：**
```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable error message"
}
```

**HTTP 状态码规范：**
- 200：成功（评分更新）
- 201：创建成功（电影、新评分）
- 400：请求格式错误（无效 JSON、缺少必填字段）
- 401：未认证（缺少或无效 Token/Rater-Id）
- 404：资源不存在（电影未找到）
- 422：语义错误（评分值不在允许范围内）
- 500：服务器内部错误

### 2.5 容器化部署

**Dockerfile 设计（多阶段构建）：**
1. **Builder 阶段**：使用 golang:1.22-alpine 编译
   - 下载依赖
   - 静态编译（CGO_ENABLED=0）
2. **Runtime 阶段**：使用 alpine:latest
   - 非 root 用户运行（appuser）
   - 仅复制编译后的二进制文件
   - 暴露 8080 端口

**docker-compose.yml 设计：**
1. **mysql 服务**：
   - 健康检查：`mysqladmin ping`
   - 数据持久化：volume 挂载
2. **migrations 服务**：
   - 依赖 mysql 健康
   - 使用 goose 执行迁移
3. **api 服务**：
   - 依赖 migrations 完成
   - 环境变量注入配置
   - 健康检查：`GET /healthz`

---

## 3. 项目优化建议

在完成基本功能后，以下是可以进一步优化的方向：

### 3.1 性能优化

**1. 数据库层面**
- **读写分离**：引入主从复制，读请求路由到从库
- **连接池调优**：根据实际负载调整 MaxOpenConns 和 MaxIdleConns
- **查询优化**：
  - 使用 EXPLAIN 分析慢查询
  - 考虑分区表（按 release_date 分区）
  - 添加复合索引优化多条件查询

**2. 缓存策略**
- **Redis 缓存**：
  - 缓存热门电影数据（TTL 1 小时）
  - 缓存评分聚合结果（TTL 5 分钟）
  - 使用 Cache-Aside 模式
- **本地缓存**：使用 go-cache 缓存配置和静态数据

**3. 并发处理**
- 票房 API 调用改为真正的异步（goroutine + 消息队列）
- 批量查询优化：使用 IN 查询代替循环单次查询

### 3.2 可靠性增强

**1. 可观测性**
- **结构化日志**：已使用 slog，可增加 trace_id
- **Metrics 监控**：
  - 集成 Prometheus，暴露 /metrics 端点
  - 监控指标：请求QPS、延迟分布、错误率、数据库连接数
- **分布式追踪**：接入 OpenTelemetry，追踪请求全链路

**2. 熔断与降级**
- 使用 hystrix-go 对票房 API 调用实现熔断
- 票房服务异常时，快速失败避免雪崩
- 提供降级策略：票房数据缓存或直接返回 null

**3. 限流保护**
- 使用 rate limiter 中间件限制单 IP 请求频率
- Token bucket 算法实现
- 区分读写请求的限流策略

### 3.3 安全加固

**1. 认证授权**
- 引入 JWT 替代静态 Bearer Token
- 实现基于角色的访问控制（RBAC）
- 评分接口增加防刷机制（同一用户对同一电影限制评分频率）

**2. 输入验证**
- 增强参数校验：使用 validator 库
- SQL 注入防护：已使用参数化查询，但需定期审计
- XSS 防护：对输出内容进行转义

**3. HTTPS**
- 生产环境强制 HTTPS
- 配置 TLS 证书
- 添加 HSTS 头

### 3.4 数据一致性

**1. 事务管理**
- 电影创建 + 票房数据插入使用事务
- 评分更新 + 聚合计数使用乐观锁

**2. 数据备份**
- 定期全量备份 + 增量备份
- 异地灾备
- 备份恢复演练

### 3.5 开发体验

**1. 自动化测试**
- 单元测试覆盖率 > 80%
- 集成测试：使用 testcontainers 启动真实数据库
- 契约测试：基于 OpenAPI 规范生成测试用例

**2. CI/CD**
- GitHub Actions / GitLab CI 自动化流水线
- 代码质量检查：golangci-lint
- 自动部署到测试环境

**3. 文档**
- API 文档：基于 OpenAPI 生成 Swagger UI
- 架构文档：使用 C4 模型或 PlantUML
- 开发文档：详细的 README 和 CONTRIBUTING

### 3.6 扩展性设计

**1. 微服务拆分**
- 电影服务、评分服务、票房服务独立部署
- 使用消息队列解耦
- API Gateway 统一入口

**2. 多租户支持**
- 数据隔离：schema 隔离或 tenant_id 字段
- 配置隔离：每个租户独立配置

**3. 国际化**
- 支持多语言错误消息
- 时区处理：统一使用 UTC 存储

---

## 总结

本项目采用 **Go + MySQL + Docker** 的技术栈，实现了一个完整的电影评分 API 服务。核心设计思路：

1. **数据库设计**：三表分离（电影、票房、评分），复合主键实现 Upsert，多索引支持查询
2. **后端架构**：清晰分层，依赖注入，中间件模式，错误降级
3. **容器化**：多阶段构建，非 root 运行，健康检查，一键启动

项目满足了所有功能需求，并在设计上预留了优化空间。通过持续的性能优化、可靠性增强和安全加固，可以将其发展为一个生产级的服务。
