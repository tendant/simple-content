[根目录](../../CLAUDE.md) > [pkg](../) > **repository**

# Repository 模块 - 数据访问层

## 模块职责

Repository模块是系统的数据访问层，提供统一的数据接口抽象和多种实现：
- 定义数据访问接口契约（在internal/repository中）
- 提供内存实现（开发和测试使用）
- 提供PostgreSQL实现（生产环境使用）
- 支持内容、对象、元数据的CRUD操作
- 处理衍生内容关系管理

## 入口与启动

Repository模块通过工厂模式或直接构造函数创建实例：

```go
// 内存实现
import "github.com/tendant/simple-content/pkg/repository/memory"
contentRepo := memory.NewContentRepository()

// PostgreSQL实现  
import "github.com/tendant/simple-content/pkg/repository/psql"
contentRepo := psql.NewContentRepository(db)
```

## 对外接口

### 核心Repository接口（定义在internal/repository）

#### ContentRepository
```go
type ContentRepository interface {
    Create(ctx context.Context, content *domain.Content) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Content, error)
    Update(ctx context.Context, content *domain.Content) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, filter *ContentFilter) ([]*domain.Content, error)
    
    // 衍生内容相关
    CreateDerivedContent(ctx context.Context, derivedContent *domain.DerivedContent) error
    GetDerivedContentsByParentID(ctx context.Context, parentID uuid.UUID) ([]domain.DerivedContent, error)
}
```

#### ObjectRepository  
```go
type ObjectRepository interface {
    Create(ctx context.Context, object *domain.Object) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Object, error)
    Update(ctx context.Context, object *domain.Object) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetByContentID(ctx context.Context, contentID uuid.UUID) ([]*domain.Object, error)
}
```

#### MetadataRepository接口
```go
// ContentMetadataRepository
type ContentMetadataRepository interface {
    Create(ctx context.Context, metadata *domain.ContentMetadata) error
    GetByContentID(ctx context.Context, contentID uuid.UUID) (*domain.ContentMetadata, error)
    Update(ctx context.Context, metadata *domain.ContentMetadata) error
    Delete(ctx context.Context, contentID uuid.UUID) error
}

// ObjectMetadataRepository
type ObjectMetadataRepository interface {
    Create(ctx context.Context, metadata *domain.ObjectMetadata) error
    GetByObjectID(ctx context.Context, objectID uuid.UUID) (*domain.ObjectMetadata, error)
    Update(ctx context.Context, metadata *domain.ObjectMetadata) error
    Delete(ctx context.Context, objectID uuid.UUID) error
}
```

## 关键依赖与配置

### 内存实现依赖
```go
"context"
"sync"                    // 并发安全
"github.com/google/uuid"
"github.com/tendant/simple-content/internal/domain"
```

### PostgreSQL实现依赖
```go
"database/sql"
"github.com/jackc/pgx/v5"              // PostgreSQL驱动
"github.com/jackc/pgx/v5/pgxpool"     // 连接池
"github.com/tendant/simple-content/internal/domain"
```

### 实现特性对比

| 特性 | Memory实现 | PostgreSQL实现 |
|------|-----------|---------------|
| 并发安全 | ✅ sync.RWMutex | ✅ 数据库锁 |
| 持久化 | ❌ 内存存储 | ✅ 磁盘持久化 |
| 事务支持 | ❌ | ✅ ACID事务 |
| 查询性能 | 🚀 极快 | ⚡ 依赖索引 |
| 适用场景 | 开发/测试 | 生产环境 |

## 数据模型

### 内存实现存储结构
```go
type ContentRepository struct {
    mu               sync.RWMutex
    contents         map[uuid.UUID]*domain.Content                    // 主要内容存储
    derivedRelations map[uuid.UUID][]domain.DerivedContent          // 衍生关系映射
}

type ObjectRepository struct {
    mu      sync.RWMutex  
    objects map[uuid.UUID]*domain.Object                            // 对象存储
}
```

### PostgreSQL实现数据表

#### 内容相关表
```sql
-- content表
CREATE TABLE content (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    owner_id UUID NOT NULL,
    owner_type VARCHAR,
    name VARCHAR,
    description TEXT,
    document_type VARCHAR,
    status VARCHAR NOT NULL,
    derivation_type VARCHAR,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- content_metadata表
CREATE TABLE content_metadata (
    content_id UUID REFERENCES content(id),
    tags JSONB,
    file_size BIGINT,
    file_name VARCHAR,
    mime_type VARCHAR,
    checksum VARCHAR,
    checksum_algorithm VARCHAR,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

#### 对象相关表
```sql
-- object表  
CREATE TABLE object (
    id UUID PRIMARY KEY,
    content_id UUID REFERENCES content(id),
    storage_backend_name VARCHAR NOT NULL,
    storage_class VARCHAR,
    object_key VARCHAR NOT NULL,
    file_name VARCHAR,
    version INTEGER DEFAULT 1,
    object_type VARCHAR,
    status VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- object_metadata表
CREATE TABLE object_metadata (
    object_id UUID REFERENCES object(id),
    size_bytes BIGINT,
    mime_type VARCHAR,
    etag VARCHAR,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

## 测试与质量

### 测试文件覆盖
- ✅ Memory实现: 每个Repository都有对应测试文件
- ✅ PostgreSQL实现: 有部分Repository的测试文件
- ✅ 测试辅助: `test_helper.go`提供测试数据库支持

### 测试策略
```go
// Memory实现测试 - 快速单元测试
func TestContentRepository_Create(t *testing.T) {
    repo := memory.NewContentRepository()
    // 测试逻辑...
}

// PostgreSQL实现测试 - 集成测试
func TestContentRepository_Create_PostgreSQL(t *testing.T) {
    db := setupTestDB(t) // 使用测试数据库
    repo := psql.NewContentRepository(db)
    // 测试逻辑...
}
```

### Repository工厂模式
```go
// pkg/repository/psql/repository_factory.go
type RepositoryFactory struct {
    db *pgxpool.Pool
}

func (f *RepositoryFactory) NewContentRepository() repository.ContentRepository {
    return NewContentRepository(f.db)
}
```

## 常见问题 (FAQ)

**Q: 什么时候使用Memory实现vs PostgreSQL实现？**
A: Memory用于开发、测试和演示；PostgreSQL用于生产环境和需要持久化的场景

**Q: 如何切换Repository实现？**
A: 由于都实现了相同接口，只需在初始化时选择不同的构造函数

**Q: 如何处理数据库连接？**
A: PostgreSQL实现使用pgxpool管理连接池，在初始化时传入

**Q: 衍生内容关系如何存储？**
A: Memory实现用map存储关系；PostgreSQL可能需要专门的关系表

**Q: 如何进行数据迁移？**
A: 暂未发现迁移脚本，需要手动管理PostgreSQL表结构

## 相关文件清单

```
pkg/repository/
├── memory/                           # 内存实现
│   ├── content_metadata_repository.go
│   ├── content_metadata_repository_test.go
│   ├── content_repository.go
│   ├── content_repository_test.go
│   ├── object_metadata_repository.go
│   ├── object_repository.go
│   └── storage_backend_repository.go
├── psql/                            # PostgreSQL实现
│   ├── base_repository.go           # 基础Repository功能
│   ├── content_metadata_repository.go
│   ├── content_metadata_repository_test.go
│   ├── content_repository.go
│   ├── content_repository_test.go
│   ├── object_metadata_repository.go
│   ├── object_metadata_repository_test.go
│   ├── object_repository.go
│   ├── object_repository_test.go
│   ├── repository_factory.go        # Repository工厂
│   └── test_helper.go              # 测试辅助工具
└── CLAUDE.md                       # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-04 15:26:32 - 模块文档初始化
- 📝 创建Repository模块详细文档
- 🏗️ 记录Memory和PostgreSQL双实现架构
- 📋 整理核心接口定义和数据表结构
- ✅ 分析测试覆盖情况和测试策略
- 🔧 标识工厂模式和依赖注入设计
- ⚠️ 注意到缺少数据迁移脚本