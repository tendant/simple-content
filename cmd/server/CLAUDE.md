[根目录](../../CLAUDE.md) > [cmd](../) > **server**

# Server 模块 - 主服务入口

## 模块职责

Server模块是Simple Content Management System的主要HTTP服务入口点，负责：
- 初始化所有核心组件（仓储、服务、存储后端）
- 配置HTTP路由和中间件
- 提供优雅关闭机制
- 设置默认存储后端

## 入口与启动

### 主入口文件
- **文件**: `main.go`
- **启动命令**: `./dist/cmd/server` 或 `make run`
- **默认端口**: 8080 (可通过PORT环境变量修改)

### 启动流程
1. 初始化内存仓储层组件
2. 设置存储后端（内存、文件系统）
3. 初始化服务层（Content、Object、StorageBackend）
4. 注册存储后端到ObjectService
5. 创建默认文件系统存储后端
6. 设置API处理器和路由
7. 启动HTTP服务器，监听优雅关闭信号

## 对外接口

### HTTP 端点
| 路径 | 描述 |
|------|------|
| `/health` | 健康检查端点 |
| `/content/*` | 内容管理API |
| `/object/*` | 对象操作API |
| `/storage-backend/*` | 存储后端管理API |

### 中间件配置
- Logger: 请求日志记录
- Recoverer: panic恢复
- RequestID: 请求ID追踪
- RealIP: 真实IP获取
- Timeout: 60秒请求超时

## 关键依赖与配置

### 核心依赖
```go
// 路由框架
"github.com/go-chi/chi/v5"
"github.com/go-chi/chi/v5/middleware"

// 内部组件
"github.com/tendant/simple-content/internal/api"
"github.com/tendant/simple-content/pkg/repository/memory"  
"github.com/tendant/simple-content/pkg/service"
"github.com/tendant/simple-content/pkg/storage/fs"
"github.com/tendant/simple-content/pkg/storage/memory"
```

### 配置项
- **PORT**: HTTP服务端口 (默认: 8080)
- **文件存储目录**: ./data/storage (文件系统后端)

### 默认存储后端
系统自动创建名为"fs-default"的文件系统存储后端，存储路径为`./data/storage`

## 数据模型

Server模块不直接操作数据模型，而是通过以下组件进行协调：

### 初始化的仓储
- ContentRepository: 内容仓储
- ContentMetadataRepository: 内容元数据仓储  
- ObjectRepository: 对象仓储
- ObjectMetadataRepository: 对象元数据仓储
- StorageBackendRepository: 存储后端仓储

### 注册的存储后端
- "memory": 内存存储后端
- "fs": 文件系统存储后端
- "fs-test": 测试用文件系统后端

## 测试与质量

### 运行测试
当前模块主要进行集成测试，通过启动完整服务器进行端到端测试。

### 健康检查
访问 `GET /health` 端点验证服务运行状态，返回HTTP 200和"OK"响应。

### 优雅关闭
- 监听SIGINT和SIGTERM信号
- 10秒超时的优雅关闭机制
- 等待现有连接完成处理

## 常见问题 (FAQ)

**Q: 如何修改服务端口？**
A: 设置PORT环境变量，如 `PORT=9090 ./dist/cmd/server`

**Q: 如何添加新的存储后端？**
A: 在main.go中创建新的后端实例，然后调用`objectService.RegisterBackend(name, backend)`

**Q: 服务启动失败怎么排查？**
A: 检查端口占用、文件权限、依赖服务（如PostgreSQL）是否正常

**Q: 如何配置生产环境？**
A: 建议使用Docker部署，通过环境变量配置数据库连接、存储后端等参数

## 相关文件清单

```
cmd/server/
├── main.go          # 主入口文件，服务启动逻辑
└── CLAUDE.md        # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-04 15:26:32 - 模块文档初始化
- 📝 创建Server模块详细文档
- 🚀 记录启动流程和配置选项
- 🔧 标识关键依赖和存储后端注册
- 💡 提供常见问题解答