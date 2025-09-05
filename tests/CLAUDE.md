[根目录](../CLAUDE.md) > **tests**

# Tests 模块 - 测试与质量保证

## 模块职责

Tests模块负责系统的全面测试覆盖，确保代码质量和功能正确性：
- 提供集成测试验证端到端业务流程
- 提供测试工具和辅助函数
- 验证衍生内容功能的完整性
- 测试S3存储后端集成
- 建立标准化的测试模式和最佳实践

## 入口与启动

测试模块通过标准Go测试工具链运行：

```bash
# 运行所有测试
go test ./...

# 运行集成测试
go test ./tests/integration -v

# 生成覆盖率报告
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 对外接口

### 测试工具函数 (testutil包)

#### 服务器测试工具
```go
// 设置完整的内存测试服务器
func SetupTestServer() *httptest.Server

// 创建测试内容
func CreateContent(t *testing.T, serverURL string) *ContentResponse

// 设置内容元数据  
func SetContentMetadata(t *testing.T, serverURL string, contentID string, metadata map[string]interface{})

// 创建衍生内容
func CreateDerivedContent(t *testing.T, serverURL string, parentID string, derivedType string) *ContentResponse
```

#### 辅助工具函数
```go
// HTTP请求辅助函数
func MakeRequest(method, url string, body io.Reader) (*http.Response, error)

// JSON响应解析
func ParseResponse(resp *http.Response, target interface{}) error
```

### 集成测试套件

#### 衍生内容测试 (`derived_content_test.go`)
- ✅ 创建原始内容并设置元数据
- ✅ 创建多级衍生内容链（最多5级深度）
- ✅ 验证每级衍生内容的独立元数据
- ✅ 测试衍生树结构和关系查询
- ✅ 验证衍生深度限制机制

#### S3后端集成测试 (`s3_backend_test.go`)  
- ✅ 测试S3存储后端的文件上传下载
- ✅ 验证预签名URL功能
- ✅ 测试MinIO兼容性

## 关键依赖与配置

### 测试依赖
```go
// 测试框架
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"

// HTTP测试
"net/http/httptest"
"net/http"

// 内部组件
"github.com/tendant/simple-content/internal/api"
"github.com/tendant/simple-content/pkg/service"
"github.com/tendant/simple-content/pkg/repository/memory"
"github.com/tendant/simple-content/pkg/storage/memory"
```

### 测试配置
- **内存存储**: 使用内存仓储和存储后端进行快速测试
- **HTTP测试服务器**: 通过httptest.Server提供完整API测试环境
- **数据隔离**: 每个测试用例使用独立的内存实例

## 数据模型

### 测试响应模型

#### ContentResponse
```go
type ContentResponse struct {
    ID             string    `json:"id"`
    TenantID       string    `json:"tenant_id"`
    OwnerID        string    `json:"owner_id"`
    OwnerType      string    `json:"owner_type"`
    Name           string    `json:"name"`
    Description    string    `json:"description"`
    DocumentType   string    `json:"document_type"`
    Status         string    `json:"status"`
    DerivationType string    `json:"derivation_type"`
    DerivationLevel int      `json:"derivation_level"`
    ParentID       *string   `json:"parent_id,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

#### MetadataResponse
```go
type MetadataResponse struct {
    ContentID string                 `json:"content_id"`
    Tags      []string               `json:"tags"`
    Metadata  map[string]interface{} `json:"metadata"`
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
}
```

### 测试数据模式
- **租户隔离**: 使用固定的测试租户ID
- **用户标识**: 模拟不同用户的操作
- **内容类型**: 涵盖视频、图片、文档等多种内容类型
- **衍生类型**: 测试缩略图、转换等衍生场景

## 测试策略

### 测试分层结构

#### 单元测试
- **Domain层**: 实体创建、状态转换、业务规则验证
- **Repository层**: 数据存取操作、查询过滤、并发安全性
- **Service层**: 业务逻辑、错误处理、依赖协调
- **Storage层**: 存储操作、预签名URL、错误恢复

#### 集成测试  
- **API工作流**: 完整的业务流程验证
- **跨模块交互**: 验证模块间协作正确性
- **存储集成**: 真实存储后端集成测试

### 测试覆盖重点

#### 衍生内容测试要点
1. **深度限制**: 验证最大5级衍生深度
2. **元数据独立**: 每个衍生内容有独立元数据
3. **关系完整**: 父子关系正确建立和查询
4. **类型多样**: 支持缩略图、转换等多种衍生类型

#### 错误场景覆盖
- **无效输入**: 测试各种无效参数和格式
- **资源不存在**: 测试访问不存在资源的行为
- **权限验证**: 测试跨租户访问控制
- **并发操作**: 测试并发读写的一致性

## 常见问题 (FAQ)

**Q: 如何运行特定的测试套件？**
A: 使用go test命令指定包路径，如`go test ./tests/integration -v`

**Q: 集成测试失败怎么排查？**
A: 检查测试服务器启动、内存存储初始化、HTTP请求构造是否正确

**Q: 如何添加新的集成测试？**
A: 在tests/integration目录下创建*_test.go文件，使用testutil包的辅助函数

**Q: 衍生内容测试的核心验证点是什么？**
A: 验证深度限制、元数据独立性、父子关系正确性和类型支持

**Q: 如何模拟生产环境的测试场景？**
A: 使用PostgreSQL和S3的测试实例，配置接近生产的测试数据

## 相关文件清单

```
tests/
├── README.md                        # 测试策略和运行指南
├── integration/                     # 集成测试目录
│   ├── derived_content_test.go      # 衍生内容完整流程测试
│   └── s3_backend_test.go          # S3存储后端集成测试
├── testutil/                       # 测试工具和辅助函数
│   ├── helpers.go                  # 测试辅助函数
│   └── server.go                   # 测试服务器设置
└── CLAUDE.md                       # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-05 10:41:03 - 模块文档创建
- 📝 创建Tests模块详细文档  
- 🧪 整理集成测试策略和工具函数
- ✅ 分析衍生内容测试覆盖情况
- 🗄️ 记录S3后端集成测试
- 📋 建立测试数据模型和响应结构
- 💡 提供测试最佳实践和排查指南
- 📊 强调测试分层和覆盖率目标