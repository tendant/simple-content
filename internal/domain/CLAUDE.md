[根目录](../../CLAUDE.md) > [internal](../) > **domain**

# Domain 模块 - 核心业务实体

## 模块职责

Domain模块是系统的核心业务领域层，定义了所有业务实体、常量和业务规则，职责包括：
- 定义核心业务实体（Content、Object、StorageBackend等）
- 声明业务状态常量和枚举值
- 建立实体间的关系模型
- 提供衍生内容（Derived Content）的类型定义

## 入口与启动

Domain模块为纯业务实体定义，无启动入口，通过其他模块import使用：

```go
import "github.com/tendant/simple-content/internal/domain"
```

## 对外接口

### 核心实体

#### Content - 逻辑内容实体
```go
type Content struct {
    ID             uuid.UUID  // 内容唯一标识
    TenantID       uuid.UUID  // 租户ID  
    OwnerID        uuid.UUID  // 所有者ID
    OwnerType      string     // 所有者类型
    Name           string     // 内容名称
    Description    string     // 内容描述
    DocumentType   string     // 文档类型
    Status         string     // 状态
    DerivationType string     // 衍生类型
    CreatedAt      time.Time  // 创建时间
    UpdatedAt      time.Time  // 更新时间
}
```

#### Object - 物理存储对象
```go
type Object struct {
    ID                 uuid.UUID  // 对象唯一标识
    ContentID          uuid.UUID  // 关联的内容ID
    StorageBackendName string     // 存储后端名称
    StorageClass       string     // 存储类别
    ObjectKey          string     // 对象键
    FileName           string     // 文件名
    Version            int        // 版本号
    ObjectType         string     // 对象类型
    Status             string     // 状态
    CreatedAt          time.Time  // 创建时间
    UpdatedAt          time.Time  // 更新时间
}
```

## 关键依赖与配置

### 外部依赖
```go
"time"                    // 时间处理
"github.com/google/uuid"  // UUID生成和处理
```

### 业务常量

#### 内容状态 (Content Status)
- `ContentStatusCreated`: "created" - 已创建
- `ContentStatusUploaded`: "uploaded" - 已上传

#### 对象状态 (Object Status)  
- `ObjectStatusCreated`: "created" - 已创建
- `ObjectStatusUploading`: "uploading" - 上传中
- `ObjectStatusUploaded`: "uploaded" - 已上传
- `ObjectStatusProcessing`: "processing" - 处理中
- `ObjectStatusProcessed`: "processed" - 已处理
- `ObjectStatusFailed`: "failed" - 失败
- `ObjectStatusDeleted`: "deleted" - 已删除

#### 衍生类型 (Derivation Types)
- `ContentDerivationTypeOriginal`: "original" - 原始内容
- `ContentDerivationTypeDerived`: "derived" - 衍生内容

#### 衍生内容类型 (Content Derived Types)
- `ContentDerivedTHUMBNAIL720`: "THUMBNAIL_720" - 720p缩略图
- `ContentDerivedTHUMBNAIL480`: "THUMBNAIL_480" - 480p缩略图  
- `ContentDerivedTHUMBNAIL256`: "THUMBNAIL_256" - 256p缩略图
- `ContentDerivedTHUMBNAIL128`: "THUMBNAIL_128" - 128p缩略图
- `ContentDerivedConversion`: "CONVERSION" - 格式转换

## 数据模型

### 实体关系图
```
Content (1) ──→ (N) Object
    │                │
    │                │
    ↓                ↓
ContentMetadata  ObjectMetadata
    │                │
    │                ↓
    │           ObjectPreview
    │
    ↓
DerivedContent ──→ Content (parent)
```

### 元数据结构

#### ContentMetadata - 内容元数据
- 文件大小、文件名、MIME类型
- 校验和及算法
- 标签和自定义元数据（JSONB）

#### ObjectMetadata - 对象元数据  
- 字节大小、MIME类型、ETag
- 自定义元数据（JSONB）

#### DerivedContent - 衍生内容关系
- 父内容ID、衍生类型
- 衍生参数和处理元数据
- 支持最多5级衍生深度

## 测试与质量

### 测试文件
- `content_test.go`: 内容实体相关测试
- `object_test.go`: 对象实体相关测试

### 测试覆盖
- ✅ 实体创建和字段验证
- ✅ 状态常量正确性
- ✅ JSON序列化/反序列化

### 数据完整性约束
- 所有UUID字段必填且有效
- 时间字段自动维护CreatedAt/UpdatedAt
- Status字段必须使用预定义常量
- 衍生内容必须有有效的ParentID

## 常见问题 (FAQ)

**Q: 如何添加新的状态值？**
A: 在对应的常量声明部分添加新常量，建议遵循现有命名规范

**Q: Content和Object的区别是什么？**
A: Content是逻辑实体，表示一个内容的概念；Object是物理实体，表示具体存储在某个后端的文件

**Q: 衍生内容如何工作？**
A: 通过DerivedContent实体建立父子关系，子内容保存衍生参数和处理元数据

**Q: 为什么使用UUID作为主键？**
A: UUID提供全局唯一性，支持分布式环境，无需中央ID生成器

## 相关文件清单

```
internal/domain/
├── audit.go           # 审计相关实体
├── content.go         # 内容相关实体和常量  
├── content_test.go    # 内容实体测试
├── object.go          # 对象相关实体和常量
├── object_test.go     # 对象实体测试
├── storage_backend.go # 存储后端实体
└── CLAUDE.md         # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-04 15:26:32 - 模块文档初始化
- 📝 创建Domain模块详细文档
- 🏗️ 记录核心实体结构和关系
- 📋 整理业务状态常量定义
- 🔗 建立实体关系图谱
- ✅ 标识测试覆盖情况