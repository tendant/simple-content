[根目录](../../CLAUDE.md) > [pkg](../) > **service**

# Service 模块 - 业务逻辑层

## 模块职责

Service模块是系统的业务逻辑层，协调Repository和Storage组件，实现核心业务逻辑：
- 内容生命周期管理（创建、更新、删除）
- 对象存储操作协调（上传、下载、版本管理）
- 存储后端动态注册和管理
- 元数据统一管理
- 业务规则验证和执行

## 入口与启动

Service模块通过依赖注入的方式初始化，无独立启动入口：

```go
import "github.com/tendant/simple-content/pkg/service"

// 在主程序中初始化
contentService := service.NewContentService(contentRepo, metadataRepo)
objectService := service.NewObjectService(objectRepo, metadataRepo, contentRepo, contentMetadataRepo)
storageBackendService := service.NewStorageBackendService(storageBackendRepo)
```

## 对外接口

### ContentService - 内容服务
```go
// 核心操作
func (s *ContentService) CreateContent(ctx context.Context, req *model.CreateContentRequest) (*domain.Content, error)
func (s *ContentService) GetContent(ctx context.Context, id uuid.UUID) (*domain.Content, error) 
func (s *ContentService) DeleteContent(ctx context.Context, id uuid.UUID) error
func (s *ContentService) ListContents(ctx context.Context, filter *model.ContentFilter) ([]*domain.Content, error)

// 元数据操作
func (s *ContentService) UpdateContentMetadata(ctx context.Context, contentID uuid.UUID, metadata map[string]interface{}) error
func (s *ContentService) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*domain.ContentMetadata, error)
```

### ObjectService - 对象服务
```go
// 对象管理
func (s *ObjectService) CreateObject(ctx context.Context, req *model.CreateObjectRequest) (*domain.Object, error)
func (s *ObjectService) GetObject(ctx context.Context, id uuid.UUID) (*domain.Object, error)
func (s *ObjectService) DeleteObject(ctx context.Context, id uuid.UUID) error

// 存储操作
func (s *ObjectService) Upload(ctx context.Context, objectID uuid.UUID, content io.Reader) error
func (s *ObjectService) Download(ctx context.Context, objectID uuid.UUID) (io.ReadCloser, error)

// 后端管理
func (s *ObjectService) RegisterBackend(name string, backend storage.Storage)
func (s *ObjectService) GetBackend(name string) (storage.Storage, error)
```

### StorageBackendService - 存储后端服务
```go
func (s *StorageBackendService) CreateStorageBackend(ctx context.Context, name, backendType string, config map[string]interface{}) (*domain.StorageBackend, error)
func (s *StorageBackendService) GetStorageBackend(ctx context.Context, id uuid.UUID) (*domain.StorageBackend, error)
func (s *StorageBackendService) ListStorageBackends(ctx context.Context) ([]*domain.StorageBackend, error)
func (s *StorageBackendService) DeleteStorageBackend(ctx context.Context, id uuid.UUID) error
```

## 关键依赖与配置

### 内部依赖
```go
// Domain层
"github.com/tendant/simple-content/internal/domain"
"github.com/tendant/simple-content/internal/repository"

// 数据模型
"github.com/tendant/simple-content/pkg/model"

// 存储抽象
"github.com/tendant/simple-content/internal/storage"
```

### 外部依赖
```go
"context"                 // 上下文管理
"io"                      // 流操作
"github.com/google/uuid"  // UUID处理
```

### 依赖注入模式
所有Service通过构造函数注入Repository依赖，遵循依赖倒置原则：

```go
// ContentService依赖注入
type ContentService struct {
    contentRepo  repository.ContentRepository         // 抽象接口
    metadataRepo repository.ContentMetadataRepository // 抽象接口
}
```

## 数据模型

### 请求模型 (pkg/model)
- `CreateContentRequest`: 创建内容请求
- `CreateObjectRequest`: 创建对象请求  
- `ContentFilter`: 内容查询过滤器

### 业务流程

#### 内容创建流程
1. 验证请求参数（租户ID、所有者ID）
2. 生成内容UUID
3. 设置初始状态为"created"
4. 保存内容实体到Repository
5. 返回创建的内容对象

#### 对象上传流程
1. 验证对象存在且状态为"created"
2. 获取关联的存储后端
3. 生成对象键（ObjectKey）
4. 调用存储后端上传接口
5. 更新对象状态为"uploaded"
6. 更新对象元数据（大小、校验和等）

## 测试与质量

### 测试文件覆盖
- `content_service_test.go`: 内容服务测试
- `object_service_test.go`: 对象服务测试
- `storage_backend_service.go`: 存储后端服务（无测试文件）

### 测试覆盖范围
- ✅ 内容CRUD操作测试
- ✅ 对象上传下载测试
- ✅ 元数据操作测试
- ✅ 错误场景测试
- ⚠️ 存储后端服务缺少专门测试

### 质量保证
- 所有公开方法都支持context.Context
- 完整的错误处理和返回
- Repository接口抽象，便于单元测试Mock
- 业务逻辑与数据访问清晰分离

## 常见问题 (FAQ)

**Q: 如何添加新的业务逻辑？**
A: 在对应的Service中添加新方法，通过Repository接口操作数据

**Q: 如何扩展新的存储后端？**
A: 实现storage.Storage接口，然后通过ObjectService.RegisterBackend()注册

**Q: Service层如何处理事务？**
A: 当前通过Repository层处理，Service协调多个Repository操作

**Q: 如何进行Service层测试？**
A: 使用Mock Repository实现，测试业务逻辑而不依赖具体数据存储

## 相关文件清单

```
pkg/service/
├── content_service.go         # 内容服务实现
├── content_service_test.go    # 内容服务测试  
├── object_service.go          # 对象服务实现
├── object_service_test.go     # 对象服务测试
├── storage_backend_service.go # 存储后端服务实现
└── CLAUDE.md                 # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-04 15:26:32 - 模块文档初始化
- 📝 创建Service模块详细文档
- 🏗️ 记录三个核心服务的接口设计
- 🔧 标识依赖注入模式和抽象层次
- 📋 分析业务流程和数据模型
- ✅ 评估测试覆盖情况
- ⚠️ 标识存储后端服务测试缺口