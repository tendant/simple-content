[根目录](../../CLAUDE.md) > [cmd](../) > **mcpserver**

# MCP Server 模块 - Model Context Protocol 服务器

## 模块职责

MCP Server模块实现了Model Context Protocol (MCP) 服务器，为AI模型提供内容管理能力：
- 实现MCP协议标准，支持AI工具调用
- 提供内容上传、下载、管理的MCP工具
- 集成PostgreSQL数据库和S3存储
- 支持Base64编码的内容传输
- 为AI助手提供文件操作接口

## 入口与启动

### 主入口文件
- **文件**: `main.go`
- **启动命令**: `./dist/cmd/mcpserver` 或指定参数启动
- **默认端口**: 8000
- **协议**: MCP (Model Context Protocol)

### 启动流程
1. 解析命令行参数和环境变量配置
2. 加载.env文件 (如果存在)
3. 建立PostgreSQL数据库连接
4. 初始化S3存储后端配置
5. 创建仓储层和服务层组件
6. 注册S3存储后端到ObjectService
7. 初始化MCP处理器和服务器
8. 注册MCP工具和资源
9. 启动MCP服务器监听

### 命令行参数
```bash
./mcpserver [flags]
# 主要通过环境变量配置，支持标准flag包参数
```

## 对外接口

### MCP Tools - AI工具接口
MCP协议工具，供AI模型调用：

| 工具名 | 描述 | 参数 | 返回 |
|--------|------|------|------|
| `upload_content` | 上传内容到系统 | content(base64), filename, content_type | content_id, status |
| `download_content` | 下载内容数据 | content_id | base64_content, metadata |
| `list_contents` | 列出内容清单 | filter_params | content_list |
| `get_content_info` | 获取内容详情 | content_id | content_metadata |

### MCP Resources - 资源接口
MCP协议资源，提供系统状态信息：

| 资源名 | 描述 | URI | 内容类型 |
|--------|------|-----|----------|
| `content://list` | 内容列表资源 | content://list | application/json |
| `storage://status` | 存储状态资源 | storage://status | application/json |

## 关键依赖与配置

### 环境变量配置
```bash
# 服务器配置
HOST=localhost
PORT=8000
BASE_URL=http://localhost:8000

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
AWS_S3_BUCKET=mymusic              # 默认桶名
AWS_S3_REGION=us-east-1
AWS_S3_USE_SSL=false
```

### MCP依赖
```go
// MCP协议实现
"github.com/mark3labs/mcp-go/mcp"
"github.com/mark3labs/mcp-go/server"

// 配置管理
"github.com/ilyakaznacheev/cleanenv"
"github.com/joho/godotenv"

// 数据库连接
"github.com/jackc/pgx/v5/pgxpool"

// 内部组件
"github.com/tendant/simple-content/internal/mcp"
"github.com/tendant/simple-content/pkg/repository/psql"
"github.com/tendant/simple-content/pkg/service"
"github.com/tendant/simple-content/pkg/storage/s3"
```

### MCP协议特性
- **工具调用**: 支持AI模型调用内容管理工具
- **资源访问**: 提供系统资源的只读访问
- **Base64编码**: 支持二进制内容的安全传输
- **异步操作**: 支持长时间运行的操作

## 数据模型

### MCP工具参数

#### upload_content工具参数
```go
type UploadContentParams struct {
    Content     string `json:"content"`      // Base64编码的文件内容
    Filename    string `json:"filename"`     // 文件名
    ContentType string `json:"content_type"` // MIME类型
    OwnerID     string `json:"owner_id"`     // 所有者ID
    TenantID    string `json:"tenant_id"`    // 租户ID
}
```

#### download_content工具参数
```go
type DownloadContentParams struct {
    ContentID string `json:"content_id"` // 内容ID
}
```

### MCP响应模型
```go
type MCPToolResponse struct {
    Success bool                   `json:"success"`
    Data    map[string]interface{} `json:"data"`
    Error   *string               `json:"error,omitempty"`
}
```

## 测试与质量

### 开发环境集成
MCP Server可以与AI开发环境集成：
- **Claude Desktop**: 通过MCP配置文件集成
- **其他AI工具**: 支持标准MCP协议的工具
- **开发调试**: 支持本地调试和日志输出

### 配置示例
Claude Desktop配置示例：
```json
{
  "mcpServers": {
    "simple-content": {
      "command": "./dist/cmd/mcpserver",
      "env": {
        "CONTENT_PG_HOST": "localhost",
        "AWS_S3_ENDPOINT": "http://localhost:9000"
      }
    }
  }
}
```

### 错误处理
- **数据库连接错误**: 自动重试和错误日志
- **S3操作错误**: 详细错误信息返回
- **MCP协议错误**: 标准MCP错误响应格式
- **Base64解码错误**: 输入验证和错误提示

## 常见问题 (FAQ)

**Q: 什么是Model Context Protocol (MCP)？**
A: MCP是Anthropic开发的协议，让AI模型可以安全地访问外部工具和资源

**Q: 如何配置Claude Desktop使用MCP Server？**
A: 在Claude Desktop配置文件中添加MCP服务器配置，指定命令和环境变量

**Q: 支持哪些文件类型？**
A: 支持任何可以Base64编码的文件类型，包括图片、文档、音频、视频等

**Q: MCP Server如何保证安全性？**
A: 通过数据库权限控制、租户隔离、Base64编码传输等机制保证安全

**Q: 如何调试MCP工具调用？**
A: 查看服务器日志输出，使用slog记录详细的操作信息和错误

## 相关文件清单

```
cmd/mcpserver/
├── main.go          # MCP服务器主入口，配置和启动逻辑
└── CLAUDE.md       # 本模块文档

internal/mcp/
├── handler.go       # MCP协议处理器，工具和资源实现
└── CLAUDE.md       # MCP处理器文档
```

## 变更记录 (Changelog)

### 2025-09-05 10:41:03 - 模块文档创建
- 📝 创建MCP Server模块详细文档
- 🤖 记录MCP协议工具和资源接口
- 🔧 整理环境变量配置和启动流程
- 🗄️ 记录PostgreSQL和S3集成配置
- 📋 建立MCP数据模型和响应结构
- 💡 提供AI集成和调试指南
- 🔒 强调安全性和错误处理机制