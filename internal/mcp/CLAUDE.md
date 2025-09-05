[根目录](../../CLAUDE.md) > [internal](../) > **mcp**

# MCP 模块 - Model Context Protocol 处理器

## 模块职责

MCP模块实现Model Context Protocol的具体处理逻辑，为AI模型提供内容管理工具：
- 实现MCP工具的具体业务逻辑
- 处理Base64编码的文件传输
- 提供内容上传下载的AI友好接口  
- 管理MCP资源的访问和返回
- 协调Service层完成内容操作

## 入口与启动

MCP模块通过mcpserver程序引用，无独立启动入口：

```go
import "github.com/tendant/simple-content/internal/mcp"

// 在MCP服务器中初始化
handler := mcp.NewHandler(contentService, objectService)
server.RegisterTools(handler.GetTools())
server.RegisterResources(handler.GetResources())
```

## 对外接口

### MCP工具实现

#### upload_content 工具
```go
func (h *Handler) UploadContent(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error)
```
**功能**: 接收Base64编码的文件内容并存储
**参数**:
- `content`: Base64编码的文件数据
- `filename`: 文件名
- `content_type`: MIME类型
- `owner_id`: 所有者UUID
- `tenant_id`: 租户UUID

**处理流程**:
1. 验证和解析输入参数
2. Base64解码文件内容
3. 创建Content和Object实体
4. 上传文件到S3存储
5. 更新元数据信息
6. 返回内容ID和状态

#### download_content 工具
```go
func (h *Handler) DownloadContent(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error)
```
**功能**: 下载内容并返回Base64编码数据
**参数**:
- `content_id`: 内容UUID

**处理流程**:
1. 验证内容ID格式
2. 查询内容和对象信息
3. 从S3存储下载文件
4. Base64编码文件数据
5. 返回编码内容和元数据

#### list_contents 工具
```go
func (h *Handler) ListContents(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error)
```
**功能**: 列出系统中的内容清单
**参数**: 
- `tenant_id` (可选): 租户过滤
- `owner_id` (可选): 所有者过滤
- `limit` (可选): 返回数量限制

#### get_content_info 工具
```go
func (h *Handler) GetContentInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error)
```
**功能**: 获取内容的详细元数据信息
**参数**:
- `content_id`: 内容UUID

### MCP资源实现

#### content://list 资源
提供内容列表的只读访问，返回JSON格式的内容摘要

#### storage://status 资源
提供存储后端状态信息，包括可用性和配置

## 关键依赖与配置

### MCP框架依赖
```go
// MCP协议库
"github.com/mark3labs/mcp-go/mcp"
"github.com/mark3labs/mcp-go/server"

// Base64和文件处理
"encoding/base64"
"bytes"
"os"
"path/filepath"

// 业务服务层
"github.com/tendant/simple-content/pkg/service"
"github.com/tendant/simple-content/pkg/model"
```

### 错误处理模式
- **参数验证错误**: 返回详细的参数格式错误信息
- **业务逻辑错误**: 包装Service层错误为MCP标准格式
- **编码解码错误**: Base64相关错误的友好提示
- **资源不存在错误**: 标准的404类型错误响应

## 数据模型

### MCP工具响应格式
```go
type ToolResult struct {
    Content []Content `json:"content"`
    IsError bool      `json:"isError"`
}

type Content struct {
    Type string      `json:"type"`
    Text string      `json:"text"`
    Data interface{} `json:"data,omitempty"`
}
```

### 内容上传响应
```go
type UploadResponse struct {
    ContentID string `json:"content_id"`
    ObjectID  string `json:"object_id"`
    Status    string `json:"status"`
    Message   string `json:"message"`
}
```

### 内容下载响应
```go
type DownloadResponse struct {
    ContentID   string                 `json:"content_id"`
    Filename    string                 `json:"filename"`
    ContentType string                 `json:"content_type"`
    Content     string                 `json:"content"`      // Base64编码
    Size        int64                  `json:"size"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

### Base64处理特性
- **编码验证**: 检查Base64格式正确性
- **大文件支持**: 支持大文件的Base64编解码
- **MIME类型检测**: 自动检测上传文件的MIME类型
- **文件名处理**: 支持Unicode文件名的正确处理

## 测试与质量

### MCP工具测试策略
```go
// 模拟MCP工具调用
func TestUploadContent(t *testing.T) {
    handler := NewHandler(mockContentService, mockObjectService)
    params := map[string]interface{}{
        "content":      base64Content,
        "filename":     "test.txt",
        "content_type": "text/plain",
    }
    result, err := handler.UploadContent(ctx, params)
    assert.NoError(t, err)
    assert.False(t, result.IsError)
}
```

### 集成测试要点
- **端到端工具调用**: 测试完整的上传下载流程
- **Base64编解码**: 验证各种文件类型的正确处理
- **错误场景**: 测试无效参数和系统错误的处理
- **大文件处理**: 测试大文件上传下载的性能

### AI模型集成测试
- **Claude Desktop**: 通过实际Claude Desktop测试工具调用
- **参数传递**: 验证AI模型参数传递的正确性
- **响应解析**: 确保AI能正确解析工具返回结果

## 常见问题 (FAQ)

**Q: 如何处理大文件的Base64编码？**
A: 系统支持流式Base64编码，但建议大文件使用预签名URL直传

**Q: MCP工具调用失败怎么排查？**
A: 检查参数格式、Base64编码正确性、服务依赖状态和日志输出

**Q: 支持哪些MCP协议版本？**
A: 基于mark3labs/mcp-go实现，支持MCP协议的标准版本

**Q: 如何添加新的MCP工具？**
A: 在Handler中实现新工具方法，并在GetTools()中注册

**Q: Base64编码的文件大小限制？**
A: 理论上无限制，但受系统内存和网络传输时间影响，建议单文件不超过100MB

## 相关文件清单

```
internal/mcp/
├── handler.go       # MCP协议处理器主要实现
└── CLAUDE.md       # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-05 10:41:03 - 模块文档创建
- 📝 创建MCP处理器模块详细文档
- 🔧 记录四个核心MCP工具的实现逻辑
- 📋 整理Base64编码处理和数据模型
- 🤖 建立AI模型集成测试策略
- ⚡ 记录错误处理和性能优化要点
- 💡 提供MCP工具开发和调试指南