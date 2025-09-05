[根目录](../../CLAUDE.md) > [pkg](../) > **storage**

# Storage 模块 - 存储后端实现

## 模块职责

Storage模块实现了可插拔的存储后端系统，为内容管理提供多种存储选择：
- 定义统一的存储接口抽象
- 实现内存存储（开发和测试）
- 实现文件系统存储（单机部署）
- 实现S3存储（云端和分布式部署）
- 支持预签名URL和直接上传下载
- 提供存储后端的动态注册机制

## 入口与启动

Storage模块通过配置创建不同的存储后端实例：

```go
// 内存存储
import "github.com/tendant/simple-content/pkg/storage/memory"
memBackend := memory.NewMemoryBackend()

// 文件系统存储
import "github.com/tendant/simple-content/pkg/storage/fs"
fsBackend, err := fs.NewFSBackend(fs.Config{
    BaseDir: "./data/storage",
})

// S3存储
import "github.com/tendant/simple-content/pkg/storage/s3"
s3Backend, err := s3.NewS3Backend(s3.Config{
    Region: "us-east-1",
    Bucket: "my-bucket",
})
```

## 对外接口

### 核心存储接口（定义在internal/storage）
```go
type Storage interface {
    // 直接上传下载
    Upload(ctx context.Context, key string, reader io.Reader) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // 预签名URL（支持客户端直传）
    GetUploadURL(ctx context.Context, key string) (string, error)
    GetDownloadURL(ctx context.Context, key string) (string, error)
    
    // 元数据操作
    GetMetadata(ctx context.Context, key string) (map[string]interface{}, error)
}
```

### 存储后端特性对比

| 存储类型 | 持久化 | 分布式 | 预签名URL | 适用场景 |
|---------|--------|--------|-----------|---------|
| Memory | ❌ 内存 | ❌ 单机 | ❌ 不支持 | 开发测试 |
| FileSystem | ✅ 磁盘 | ❌ 单机 | ⚠️ 需配置 | 单机部署 |
| S3 | ✅ 云端 | ✅ 分布式 | ✅ 原生支持 | 生产环境 |

## 关键依赖与配置

### 文件系统存储配置
```go
type Config struct {
    BaseDir   string // 基础存储目录，默认: "./data/storage"
    URLPrefix string // URL前缀，用于生成下载链接
}
```

### S3存储配置
```go
type Config struct {
    // AWS基础配置
    Region          string // AWS区域
    Bucket          string // S3桶名称
    AccessKeyID     string // 访问密钥ID
    SecretAccessKey string // 访问密钥
    
    // 自定义端点（支持MinIO等S3兼容服务）
    Endpoint        string // 自定义端点地址
    UseSSL          bool   // 是否使用SSL，默认true
    UsePathStyle    bool   // 是否使用路径风格，默认false
    
    // 预签名URL配置
    PresignDuration int    // 预签名URL有效期（秒），默认3600
    
    // 服务端加密
    EnableSSE       bool   // 启用服务端加密
    SSEAlgorithm    string // 加密算法：AES256 或 aws:kms
    SSEKMSKeyID     string // KMS密钥ID（可选）
    
    // MinIO特有配置
    CreateBucketIfNotExist bool // 自动创建桶
}
```

### 外部依赖
```go
// S3存储依赖
"github.com/aws/aws-sdk-go-v2/service/s3"
"github.com/aws/aws-sdk-go-v2/config"
"github.com/aws/aws-sdk-go-v2/credentials"

// 文件系统存储依赖
"os"
"path/filepath"
"io"
```

## 数据模型

### 存储键规范
系统使用统一的键格式：`{content-id}/{object-id}`
- 例如：`123e4567-e89b-12d3-a456-426614174000/456e7890-e12b-34d5-a678-901234567890`

### 文件系统存储结构
```
./data/storage/
├── 123e4567-e89b-12d3-a456-426614174000/
│   ├── 456e7890-e12b-34d5-a678-901234567890
│   └── 789e0123-e45b-67d8-a901-234567890123
└── 234f5678-f90c-23e4-b567-123456789012/
    └── 567f8901-f23c-45e6-b789-012345678901
```

### S3存储结构
```
bucket-name/
├── 123e4567-e89b-12d3-a456-426614174000/456e7890-e12b-34d5-a678-901234567890
├── 123e4567-e89b-12d3-a456-426614174000/789e0123-e45b-67d8-a901-234567890123
└── 234f5678-f90c-23e4-b567-123456789012/567f8901-f23c-45e6-b789-012345678901
```

### 内存存储结构
```go
type MemoryBackend struct {
    mu   sync.RWMutex
    data map[string][]byte // key -> content
}
```

## 测试与质量

### 测试文件覆盖
- ✅ `fs_test.go`: 文件系统存储测试
- ✅ `s3_test.go`: S3存储测试
- ❌ Memory存储缺少专门测试文件

### 测试策略

#### 文件系统存储测试
- 测试上传下载基本功能
- 测试目录自动创建
- 测试文件不存在的错误处理

#### S3存储测试
- 支持MinIO集成测试
- 测试预签名URL生成
- 测试服务端加密配置
- 测试桶自动创建功能

### Docker测试环境
通过docker-compose提供MinIO测试环境：
```yaml
services:
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

**Q: 如何选择存储后端？**
A: 开发测试用Memory，单机部署用FileSystem，生产环境用S3

**Q: 文件系统存储如何支持HTTP访问？**
A: 需要配置URLPrefix并通过Web服务器（如nginx）提供静态文件服务

**Q: S3存储支持哪些服务提供商？**
A: 支持AWS S3、MinIO、阿里云OSS、腾讯云COS等S3兼容服务

**Q: 如何实现存储迁移？**
A: 目前需要手动实现，读取源存储内容并写入目标存储

**Q: 预签名URL的安全性如何保证？**
A: 通过时间限制（默认1小时）和访问权限控制，建议生产环境设置较短的有效期

## 相关文件清单

```
pkg/storage/
├── fs/                    # 文件系统存储实现
│   ├── fs.go             # 文件系统存储主要实现
│   ├── fs_test.go        # 文件系统存储测试
│   └── README.md         # 文件系统存储文档
├── memory/               # 内存存储实现
│   └── memory.go         # 内存存储实现
├── s3/                   # S3存储实现
│   ├── s3.go             # S3存储主要实现
│   ├── s3_test.go        # S3存储测试
│   └── README.md         # S3存储文档
└── CLAUDE.md            # 本模块文档
```

## 变更记录 (Changelog)

### 2025-09-04 15:26:32 - 模块文档初始化
- 📝 创建Storage模块详细文档
- 🏗️ 记录三种存储后端的特性和配置
- 📋 整理存储接口设计和数据模型
- ✅ 分析测试覆盖情况
- 📚 整合现有README文档内容
- ⚠️ 标识Memory存储缺少测试
- 💡 提供存储选择和迁移建议