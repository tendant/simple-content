[根目录](../../CLAUDE.md) > [internal](../) > **api**

# API 模块 - HTTP处理层

## 模块职责

API模块是系统的HTTP处理层，负责处理所有REST API请求并协调业务服务：
- 提供内容管理的RESTful接口
- 处理对象存储操作的HTTP端点
- 实现文件上传下载的API逻辑
- 提供存储后端管理接口
- 统一HTTP请求响应处理和错误管理

## 入口与启动

API模块通过各个服务的main.go文件引用，无独立启动入口：

```go
import "github.com/tendant/simple-content/internal/api"

// 在服务启动中初始化
contentHandler := api.NewContentHandler(contentService, objectService)
filesHandler := api.NewFilesHandler(contentService, objectService) 
objectHandler := api.NewObjectHandler(objectService)
```

## 对外接口

### ContentHandler - 内容管理接口
| 路径 | 方法 | 描述 | 请求体 | 响应 |
|------|------|------|--------|------|
| `/` | POST | 创建内容 | CreateContentRequest | Content |
| `/{id}` | GET | 获取内容 | - | Content |
| `/{id}` | DELETE | 删除内容 | - | Status |
| `/list` | GET | 列出内容 | Query参数 | Content[] |
| `/bulk` | GET | 批量获取内容 | id[]参数 | Content[] |
| `/{id}/metadata` | PUT | 更新元数据 | Metadata | Status |
| `/{id}/metadata` | GET | 获取元数据 | - | ContentMetadata |
| `/{id}/objects` | POST | 创建关联对象 | CreateObjectRequest | Object |
| `/{id}/objects` | GET | 列出关联对象 | - | Object[] |
| `/{id}/download` | GET | 获取下载链接 | - | DownloadURL |

### FilesHandler - 文件操作接口
| 路径 | 方法 | 描述 | 请求体 | 响应 |
|------|------|------|--------|------|
| `/` | POST | 创建文件上传 | CreateFileRequest | CreateFileResponse |
| `/{content_id}/complete` | POST | 完成上传 | - | Status |
| `/{content_id}` | PATCH | 更新文件元数据 | UpdateMetadataRequest | Status |
| `/{content_id}` | GET | 获取文件信息 | - | FileInfoResponse |
| `/bulk` | GET | 批量获取文件 | id[]参数 | FileInfoResponse[] |

### ObjectHandler - 对象操作接口
| 路径 | 方法 | 描述 | 请求体 | 响应 |
|------|------|------|--------|------|
| `/{id}` | GET | 获取对象信息 | - | Object |
| `/{id}` | DELETE | 删除对象 | - | Status |
| `/{id}/upload-url` | GET | 获取上传URL | - | UploadURL |
| `/{id}/download-url` | GET | 获取下载URL | - | DownloadURL |
| `/{id}/upload` | POST | 直接上传 | 文件流 | Status |
| `/{id}/download` | GET | 直接下载 | - | 文件流 |

### StorageBackendHandler - 存储后端管理接口
提供存储后端的CRUD操作和配置管理。

## 关键依赖与配置

### 内部依赖
```go
// 业务服务层
"github.com/tendant/simple-content/pkg/service"

// 领域模型
"github.com/tendant/simple-content/internal/domain"

// 数据模型
"github.com/tendant/simple-content/pkg/model"
```

### 外部依赖
```go
// HTTP路由和渲染
"github.com/go-chi/chi/v5"
"github.com/go-chi/render"

// 标准库
"net/http"
"encoding/json"
"log/slog"
"github.com/google/uuid"
```

### 请求处理模式
- 使用Chi路由器进行路径参数绑定
- JSON请求体解析和响应渲染
- 统一的错误处理和日志记录
- UUID参数验证和解析
- 上下文传递用于超时和取消

## 数据模型

### 请求模型

#### CreateContentRequest
```go
type CreateContentRequest struct {
    TenantID       string `json:"tenant_id"`
    OwnerID        string `json:"owner_id"`
    OwnerType      string `json:"owner_type"`
    Name           string `json:"name"`
    Description    string `json:"description,omitempty"`
    DocumentType   string `json:"document_type"`
    DerivationType string `json:"derivation_type"`
}
```

#### CreateFileRequest
```go
type CreateFileRequest struct {
    OwnerID      string `json:"owner_id"`
    OwnerType    string `json:"owner_type"`
    TenantID     string `json:"tenant_id"`
    FileName     string `json:"file_name"`
    MimeType     string `json:"mime_type,omitempty"`
    FileSize     int64  `json:"file_size,omitempty"`
    DocumentType string `json:"document_type,omitempty"`
}
```

### 响应模型

#### FileInfoResponse
```go
type FileInfoResponse struct {
    ContentID      string                 `json:"content_id"`
    FileName       string                 `json:"file_name"`
    PreviewURL     string                 `json:"preview_url"`
    DownloadURL    string                 `json:"download_url"`
    Metadata       map[string]interface{} `json:"metadata"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
    Status         string                 `json:"status"`
    MimeType       string                 `json:"mime_type"`
    FileSize       int64                  `json:"file_size"`
    DerivationType string                 `json:"derivation_type"`
    OwnerID        string                 `json:"owner_id"`
    OwnerType      string                 `json:"owner_type"`
    TenantID       string                 `json:"tenant_id"`
}
```

## 测试与质量

### 测试文件覆盖
- ✅ `content_handler_test.go`: 内容处理器测试
- ✅ `files_handler_test.go`: 文件处理器测试  
- ❌ `object_handler`: 缺少专门测试文件
- ❌ `storage_backend_handler`: 缺少专门测试文件

### 测试策略
- **单元测试**: 测试各个Handler的HTTP处理逻辑
- **Mock服务**: 使用Mock Service层避免依赖外部服务
- **HTTP测试**: 使用httptest包进行HTTP请求响应测试
- **错误场景**: 覆盖各种错误情况和边界条件

### API行为特性
- **批量操作限制**: MAX_CONTENTS_PER_REQUEST常量控制批量请求大小
- **MIME类型验证**: 支持Microsoft Office文档类型检查
- **预签名URL**: 支持客户端直传，减少服务器负载
- **版本管理**: 支持对象版本选择（最新版本优先）

## 常见问题 (FAQ)

**Q: 如何添加新的API端点？**
A: 在对应的Handler中添加新方法，并在Routes()中注册路由

**Q: 如何处理文件上传？**
A: 使用预签名URL模式：1) 创建文件获得上传URL 2) 客户端直传 3) 调用complete完成上传

**Q: 批量操作的限制是什么？**
A: 由MAX_CONTENTS_PER_REQUEST常量控制，防止单次请求数据过大

**Q: 如何支持新的MIME类型？**
A: 在FilesHandler中更新MIME类型验证逻辑，支持更多文件格式

**Q: 错误处理的统一方式？**
A: 使用http.Error统一返回错误响应，并通过slog记录详细错误信息

## 相关文件清单

```
internal/api/
├── content_handler.go           # 内容管理HTTP处理器
├── content_handler_test.go      # 内容处理器测试
├── files_handler.go             # 文件操作HTTP处理器  
├── files_handler_test.go        # 文件处理器测试
├── object_handler.go            # 对象操作HTTP处理器
├── storage_backend_handler.go   # 存储后端管理HTTP处理器
└── CLAUDE.md                   # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-05 10:41:03 - 模块文档创建
- 📝 创建API模块详细文档
- 📋 整理四个主要Handler的接口设计
- 🔗 记录请求响应模型结构
- ✅ 分析测试覆盖情况
- ⚠️ 标识object_handler和storage_backend_handler缺少测试
- 💡 提供API设计和错误处理最佳实践