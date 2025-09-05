[根目录](../../CLAUDE.md) > [cmd](../) > **files**

# Files 模块 - 文件上传服务入口

## 模块职责

Files模块是专门用于文件上传和管理的HTTP服务入口，提供产品级的文件处理能力：
- 集成PostgreSQL数据库持久化存储
- 配置S3存储后端用于文件存储
- 提供API密钥身份认证机制
- 支持RESTful文件操作API
- 集成事件审计通知功能

## 入口与启动

### 主入口文件
- **文件**: `main.go`
- **启动命令**: `./dist/cmd/files` 
- **默认端口**: 8080 (通过tendant/chi-demo框架控制)

### 启动流程
1. 读取环境变量配置（数据库、S3、API密钥等）
2. 建立PostgreSQL数据库连接池
3. 初始化PostgreSQL仓储层组件
4. 配置S3存储后端（支持MinIO）
5. 初始化业务服务层（Content、Object服务）
6. 注册S3后端到ObjectService
7. 配置API密钥认证中间件
8. 设置路由和API处理器
9. 启动HTTP服务器

## 对外接口

### HTTP 端点
| 路径 | 方法 | 描述 | 认证 |
|------|------|------|------|
| `/healthz` | GET | 健康检查端点 | 无 |
| `/healthz/ready` | GET | 就绪检查端点 | 无 |
| `/api/v5/files` | POST | 创建文件上传任务 | API Key |
| `/api/v5/files/{content_id}/complete` | POST | 完成文件上传 | API Key |
| `/api/v5/files/{content_id}` | PATCH | 更新文件元数据 | API Key |
| `/api/v5/files/{content_id}` | GET | 获取文件信息 | API Key |
| `/api/v5/files/bulk` | GET | 批量获取文件信息 | API Key |
| `/api/v5/contents/*` | * | 内容管理API | API Key |

### API认证
- 使用SHA256哈希的API密钥验证
- 默认密钥哈希: `ba7816bf...` (对应明文"hello")
- 通过HTTP Header: `X-API-KEY` 传递

## 关键依赖与配置

### 环境变量配置
```bash
# PostgreSQL数据库配置
CONTENT_PG_HOST=localhost
CONTENT_PG_PORT=5432  
CONTENT_PG_NAME=powercard_db
CONTENT_PG_USER=content
CONTENT_PG_PASSWORD=pwd

# S3存储配置
AWS_S3_ENDPOINT=http://localhost:9000  # MinIO端点
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin
AWS_S3_BUCKET=content-bucket
AWS_S3_REGION=us-east-1
AWS_S3_USE_SSL=false

# API认证配置
API_KEY_SHA256=ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad

# 事件审计配置
EVENT_AUDIT_URL=http://localhost:14000/events
```

### 核心依赖
```go
// Web框架和中间件
"github.com/go-chi/chi/v5"
"github.com/tendant/chi-demo/app"
"github.com/tendant/chi-demo/middleware"

// 数据库连接
"github.com/jackc/pgx/v5/pgxpool"

// 内部组件
"github.com/tendant/simple-content/internal/api"
"github.com/tendant/simple-content/pkg/repository/psql"
"github.com/tendant/simple-content/pkg/service"
"github.com/tendant/simple-content/pkg/storage/s3"
```

### S3预签名URL配置
- **有效期**: 6小时 (21600秒)
- **存储后端名称**: "s3-default"
- **支持MinIO**: 是，通过自定义端点

## 数据模型

### 数据持久化
Files服务使用PostgreSQL作为主要数据存储，包含：
- **content表**: 内容实体存储
- **content_metadata表**: 内容元数据存储
- **object表**: 对象实体存储  
- **object_metadata表**: 对象元数据存储

### 存储分离架构
- **元数据**: 存储在PostgreSQL数据库中
- **文件数据**: 存储在S3兼容服务中
- **关系映射**: Content -> Object -> S3存储键

## 测试与质量

### 测试文件
- `main_test.go`: 主程序测试文件

### 依赖服务
Files服务依赖以下外部服务：
- **PostgreSQL**: 元数据持久化
- **MinIO/S3**: 文件存储后端  
- **事件审计服务** (可选): 操作审计日志

### Docker开发环境
```yaml
# docker-compose.yml中的相关服务
services:
  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: powercard_db
      POSTGRES_USER: content
      POSTGRES_PASSWORD: pwd
      
  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
```

## 常见问题 (FAQ)

**Q: 如何生成新的API密钥？**
A: 生成明文API密钥，然后使用SHA256计算哈希值，更新API_KEY_SHA256环境变量

**Q: 文件上传流程是什么？**
A: 1) POST创建文件任务获得预签名URL 2) 客户端直接上传到S3 3) POST complete通知上传完成

**Q: 如何配置自定义S3服务？**
A: 设置AWS_S3_ENDPOINT指向你的S3兼容服务端点，如MinIO或阿里云OSS

**Q: 数据库连接失败怎么排查？**
A: 检查PostgreSQL服务状态、网络连通性、用户权限、数据库名称是否正确

**Q: S3存储配置错误怎么排查？**
A: 检查端点地址、访问密钥、存储桶名称、区域配置，确认服务可达性

## 相关文件清单

```
cmd/files/
├── main.go          # 主入口文件，服务启动和配置
├── main_test.go     # 主程序测试
├── README.md        # 模块说明文档  
└── CLAUDE.md       # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-05 10:41:03 - 模块文档创建
- 📝 创建Files模块详细文档
- 🗄️ 记录PostgreSQL数据库集成
- 🗄️ 记录S3存储后端配置
- 🔐 记录API密钥认证机制
- 📋 整理环境变量配置清单
- 🚀 记录完整的启动流程
- 🔗 建立与其他模块的关系映射