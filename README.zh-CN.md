# Geofence-Updater-Lite (GUL)

[English](README.md) | [简体中文](README.zh-CN.md)

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)
![Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen.svg)

**一个轻量级、高可靠的地理围栏数据同步系统**

专为无人机/无人驾驶飞行器在低带宽、不稳定网络环境下运行而设计

[功能特性](#功能特性) • [快速开始](#快速开始) • [使用指南](#使用指南) • [API 文档](#api-文档) • [协议规范](#协议规范)

</div>

---

## 项目简介

Geofence-Updater-Lite (GUL) 是一个去中心化的地理围栏数据同步系统，核心设计理念是通过 **Merkkle Tree 实现增量更新**，将版本差异压缩至几 KB，使其能够在 GPRS 级别的网络环境中稳定运行。

**核心特性：**

- **极低带宽** - 使用 Merkle Tree + 二进制差分，增量更新仅需几 KB
- **去中心化分发** - 纯静态文件，可部署在任意 CDN/OSS，零服务器成本
- **安全优先** - Ed25519 数字签名，离线验证，防篡改
- **高性能查询** - 基于 R-Tree 空间索引，毫秒级围栏检查
- **跨平台** - 纯 Go 实现，支持 Linux/macOS/Windows

---

## 功能特性

| 特性 | 说明 |
| ------ | ------ |
| **极低带宽** | Merkle Tree 实现增量更新，版本差异可能只有几 KB |
| **去中心化分发** | 核心数据为静态文件，可部署在 CDN/OSS/IPFS |
| **数字签名** | Ed25519 签名 + KeyID 机制，防篡改验证 |
| **高性能查询** | 基于 R-Tree 空间索引，毫秒级围栏检查 |
| **离线验签** | 内置公钥验证，不依赖数据来源 |
| **版本回滚保护** | 拒绝应用旧版本数据 |
| **进度回调** | 大文件下载支持进度报告 |
| **易集成** | 纯 Go 实现，跨平台编译支持 |

---

## 架构设计

### 核心原则

```
┌─────────────────────────────────────────────────────────────────┐
│                     Git 思想                                    │
│              Merkle Tree 管理版本，只下载差异                    │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                     CDN 友好                                    │
│              纯静态文件，可部署在任意 CDN/OSS                    │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                     安全优先                                    │
│              Ed25519 签名，离线验证，防篡改                      │
└─────────────────────────────────────────────────────────────────┘
```

### 双组件架构

```
┌──────────────────────────────────────────────────────────────────────┐
│                                                                       │
│                        服务端 (Publisher)                            │
│                      CLI 工具 / Web 后台                             │
│                                                                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │ 围栏数据  │  │ Merkle   │  │  Delta   │  │ Snapshot │  │  签名   │ │
│  │  输入    │  │  Tree    │  │  Patch   │  │   文件   │  │  生成   │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘ │
│       │           │           │           │              │        │   │
│       ▼           ▼           ▼           ▼              ▼        ▼   │
│   ┌─────────────────────────────────────────────────────────────────┐ │
│   │              静态文件存储 (CDN/OSS/IPFS)                         │ │
│   │  manifest.json │  v1.bin  │  v1_v2.delta  │  v2.snapshot.bin   │ │
│   └─────────────────────────────────────────────────────────────────┘ │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP 轮询
                                    ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        客户端 (Drone SDK)                            │
│                      运行在无人机 / 遥控器 APP                        │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  HTTP 客户端   │  同步逻辑  │  SQLite  │  R-Tree  │  签名验证  │  │
│  │  (下载/重试)   │  (轮询)   │  (持久化) │ (查询)  │  (离线)    │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│                         API: Check(lat, lon) → Allowed?              │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 快速开始

### 前置要求

- **Go 1.25+** （推荐使用最新稳定版本）
- **Make** （可选，用于便捷构建）
- **Docker** （可选，用于容器化部署）

### 安装

#### 方式一：从源码构建

```bash
# 克隆仓库
git clone https://github.com/iannil/geofence-updater-lite.git
cd geofence-updater-lite

# 下载依赖
go mod download

# 构建所有二进制文件
make build-all

# 或仅构建发布工具
make build
```

构建产物位于 `bin/` 目录：

- `publisher` - 发布工具（服务端）
- `sdk-example` - SDK 使用示例（客户端）

#### 方式二：Docker 构建

```bash
# 构建镜像
docker build -t gul-publisher .

# 运行容器
docker run -it --rm -v $(pwd)/data:/data gul-publisher
```

#### 方式三：交叉编译

```bash
# 为多个平台构建
make cross-compile
```

### 基本使用流程

**1. 生成密钥对**

```bash
$ ./bin/publisher keys

生成的密钥对：
  私钥: 0x7c3a9f2e... (请妥善保管)
  公钥: 0x8d4b1c5a... (用于客户端验证)
  KeyID: k1_20240118
```

**2. 初始化数据库**

```bash
$ ./bin/publisher init

初始化完成：
  数据库: ./data/fences.db
  版本: v1
```

**3. 添加围栏**

创建围栏数据文件 `fence.json`：

```json
{
  "id": "fence-20240118-001",
  "type": "TEMP_RESTRICTION",
  "geometry": {
    "polygon": [
      {"lat": 39.9042, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4174},
      {"lat": 39.9042, "lon": 116.4174}
    ]
  },
  "start_ts": 1709880000,
  "end_ts": 1709990000,
  "priority": 10,
  "name": "北京三环临时管控区",
  "description": "临时活动禁飞区"
}
```

添加到数据库：

```bash
$ ./bin/publisher add fence.json

添加成功：fence-20240118-001 (类型: TEMP_RESTRICTION)
```

**4. 发布更新**

```bash
$ ./bin/publisher publish --output ./output

发布完成：
  版本: v2
  增量包: ./output/v1_v2.delta (2.3 KB)
  快照: ./output/v2.snapshot.bin (15.6 KB)
  清单: ./output/manifest.json
```

**5. 部署到 CDN**

将 `output/` 目录上传到你的 CDN/OSS：

```bash
# 示例：使用 AWS CLI
aws s3 sync ./output s3://your-bucket/geofence/
```

**6. 客户端使用**

```bash
$ ./bin/sdk-example \
  -manifest https://cdn.example.com/geofence/manifest.json \
  -public-key 0x8d4b1c5a... \
  -store ./geofence.db

启动同步...
  当前版本: v0
  远程版本: v2
  下载增量包: 2.3 KB
  应用更新完成: v0 → v2
  验签通过

开始围栏检查...
  检查 (39.9042, 116.4074): 禁止飞行 - 北京三环临时管控区
```

---

## 使用指南

### 发布工具 (Publisher Tool)

发布工具用于管理和发布地理围栏更新。

#### 命令说明

```bash
# 生成 Ed25519 密钥对
$ publisher keys

# 初始化围栏数据库
$ publisher init [--db-path ./data/fences.db]

# 添加新围栏
$ publisher add <fence.json>

# 批量添加围栏
$ publisher add --batch <fences-dir>

# 列出所有围栏
$ publisher list [--type TEMP_RESTRICTION]

# 删除围栏
$ publisher remove <fence-id>

# 发布新版本
$ publisher publish [--output ./output] [--message "更新说明"]

# 查看版本历史
$ publisher history
```

#### 支持的围栏类型

| 类型 | 说明 | 优先级建议 |
| ------ | ------ | ----------- |
| `TEMP_RESTRICTION` | 临时管控区 | 10-50 |
| `PERMANENT_NO_FLY` | 永久禁飞区 | 100 |
| `ALTITUDE_LIMIT` | 高度限制区 | 20-40 |
| `ALTITUDE_MINIMUM` | 最低高度要求 | 20-40 |
| `SPEED_LIMIT` | 速度限制区 | 10-30 |

#### 几何形状支持

```json
// 多边形（Polygon）
{
  "geometry": {
    "polygon": [
      {"lat": 39.9042, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4174},
      {"lat": 39.9042, "lon": 116.4174}
    ]
  }
}

// 圆形（Circle）
{
  "geometry": {
    "circle": {
      "center": {"lat": 39.9042, "lon": 116.4074},
      "radius_m": 5000
    }
  }
}

// 矩形（Rectangle）
{
  "geometry": {
    "rectangle": {
      "min": {"lat": 39.9000, "lon": 116.4000},
      "max": {"lat": 39.9200, "lon": 116.4200}
    }
  }
}
```

---

### 客户端 SDK (Drone SDK)

SDK 提供地理围栏查询和自动同步功能。

#### Go SDK 集成

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/iannil/geofence-updater-lite/pkg/config"
    "github.com/iannil/geofence-updater-lite/pkg/sync"
)

func main() {
    ctx := context.Background()

    // 创建配置
    cfg := &config.ClientConfig{
        ManifestURL:    "https://cdn.example.com/geofence/manifest.json",
        PublicKeyHex:   "8d4b1c5a...", // 公钥十六进制
        StorePath:      "./geofence.db",
        SyncInterval:   1 * time.Minute,
        HTTPTimeout:    30 * time.Second,
    }

    // 创建同步器
    syncer, err := sync.NewSyncer(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer syncer.Close()

    // 启动自动同步
    results := syncer.StartAutoSync(ctx, 1*time.Minute)

    // 处理同步结果
    go func() {
        for result := range results {
            if result.Error != nil {
                log.Printf("同步错误: %v", result.Error)
                continue
            }
            if result.UpToDate {
                log.Printf("已是最新版本 (v%d)", result.CurrentVer)
            } else {
                log.Printf("更新完成: v%d → v%d，耗时 %v",
                    result.PreviousVer, result.CurrentVer, result.Duration)
            }
        }
    }()

    // 围栏检查
    allowed, restriction, err := syncer.Check(ctx, 39.9042, 116.4074)
    if err != nil {
        log.Fatal(err)
    }

    if !allowed {
        log.Printf("禁止飞行: %s - %s", restriction.Name, restriction.Description)
        // 执行禁飞逻辑...
    }
}
```

#### SDK API 参考

| 方法 | 说明 | 返回值 |
| ------ | ------ | -------- |
| `NewSyncer(ctx, cfg)` | 创建同步器 | `(*Syncer, error)` |
| `StartAutoSync(ctx, interval)` | 启动自动同步 | `<-chan SyncResult` |
| `CheckForUpdates(ctx)` | 检查更新 | `(*Manifest, error)` |
| `Sync(ctx)` | 执行同步 | `(*SyncResult, error)` |
| `Check(ctx, lat, lon)` | 围栏检查 | `(allowed, restriction, error)` |
| `Close()` | 关闭同步器 | `error` |

---

## API 文档

### pkg/crypto - 密码学模块

```go
// 生成 Ed25519 密钥对
keyPair, err := crypto.GenerateKeyPair()

// 对数据签名
signature := crypto.Sign(privateKey, data)

// 验证签名
valid := crypto.Verify(publicKey, data, signature)

// 计算密钥 ID（用于密钥轮换）
keyID := crypto.PublicKeyToKeyID(publicKey)
```

### pkg/merkle - Merkle Tree 模块

```go
// 从围栏项构建 Merkle Tree
tree, err := merkle.NewTree(fences)

// 获取根哈希
rootHash := tree.RootHash()

// 生成 Merkle 证明
proof, err := tree.GetProof(fenceID)

// 验证 Merkle 证明
valid := merkle.VerifyProof(fenceID, fenceData, proof, rootHash)
```

### pkg/storage - 存储模块

```go
// 打开数据库
store, err := storage.Open(ctx, &storage.Config{Path: "./geofence.db"})

// 添加围栏
store.AddFence(ctx, &fence)

// 点查询（使用 R-Tree）
fences, err := store.QueryAtPoint(ctx, lat, lon)

// 版本管理
version, _ := store.GetVersion(ctx)
store.SetVersion(ctx, newVersion)
```

### pkg/sync - 同步模块

```go
// 创建同步器
syncer, _ := sync.NewSyncer(ctx, cfg)

// 检查更新
manifest, _ := syncer.CheckForUpdates(ctx)

// 同步数据
result, _ := syncer.Sync(ctx)

// 自动同步
results := syncer.StartAutoSync(ctx, interval)
```

### pkg/binarydiff - 二进制差分模块

```go
// 计算差异
delta, err := binarydiff.Diff(oldFences, newFences)

// 应用差异
newFences, err := binarydiff.PatchFences(oldFences, delta)
```

---

## 性能指标

| 操作 | 性能 | 说明 |
| ------ | ------ | ------ |
| 围栏检查 | < 1ms | 1000 次查询，R-Tree 索引 |
| Merkle Tree 构建 | < 100ms | 1000 个围栏 |
| Delta 计算 | < 50ms | 1000 个围栏对比 |
| 增量包大小 | ~2-5 KB | 典型 100 个围栏的变更 |
| 全量快照 | ~15 KB | 100 个围栏（Protobuf 编码） |

---

## 协议规范

### 围栏项 (Fence Item)

| 字段 | 类型 | 说明 |
| ------ | ------ | ------ |
| `id` | string | 唯一标识符 |
| `type` | FenceType | 围栏类型 |
| `geometry` | Geometry | 几何形状（多边形/圆形/矩形） |
| `start_ts` | int64 | 生效时间戳 |
| `end_ts` | int64 | 失效时间戳，0 表示永不过期 |
| `priority` | uint32 | 优先级，高优先级覆盖低优先级 |
| `max_alt_m` | uint32 | 最大高度限制（米），0 表示无限制 |
| `max_speed_mps` | uint32 | 最大速度限制（米/秒），0 表示无限制 |
| `name` | string | 围栏名称 |
| `description` | string | 围栏描述 |
| `signature` | []byte | Ed25519 签名 |
| `key_id` | string | 密钥 ID |

### 清单文件 (Manifest)

| 字段 | 类型 | 说明 |
| ------ | ------ | ------ |
| `version` | uint64 | 全局版本号（递增） |
| `timestamp` | int64 | 发布时间戳 |
| `root_hash` | []byte | Merkle Tree 根哈希 |
| `delta_url` | string | 增量包下载地址 |
| `snapshot_url` | string | 全量快照下载地址 |
| `delta_size` | uint64 | 增量包大小（字节） |
| `snapshot_size` | uint64 | 快照大小（字节） |
| `delta_hash` | []byte | 增量包哈希（SHA-256） |
| `snapshot_hash` | []byte | 快照哈希（SHA-256） |
| `message` | string | 版本消息 |

---

## 项目结构

```
geofence-updater-lite/
├── cmd/                          # 命令行工具
│   ├── publisher/                # 发布工具（服务端）
│   └── sdk-example/               # SDK 使用示例（客户端）
├── pkg/                          # 核心包
│   ├── binarydiff/               # 二进制差分算法
│   ├── client/                   # HTTP 客户端
│   ├── config/                   # 配置管理
│   ├── converter/                # 数据格式转换
│   ├── crypto/                   # Ed25519 密码学
│   ├── geofence/                 # 地理围栏核心逻辑
│   ├── merkle/                   # Merkle Tree 实现
│   ├── protocol/protobuf/        # Protocol Buffers 定义
│   ├── publisher/                # 发布逻辑
│   ├── storage/                  # SQLite 存储层
│   ├── sync/                     # 同步逻辑
│   └── version/                  # 版本管理
├── internal/                     # 内部包
│   ├── testutil/                 # 测试工具
│   └── version/                  # 内部版本信息
├── docs/                         # 文档
│   ├── spec/                     # 技术规范
│   ├── progress/                 # 进度文档
│   └── planning/                 # 计划文档
├── scripts/                      # 构建脚本
├── test/                         # 测试数据
├── bin/                          # 构建输出
├── Makefile                      # 构建系统
├── go.mod                        # Go 模块定义
├── go.sum                        # 依赖锁定
├── Dockerfile                    # Docker 定义
├── LICENSE                       # Apache 2.0 许可证
├── README.md                     # 英文版（默认）
├── README.zh-CN.md               # 中文版
├── CONTRIBUTING.md               # 贡献指南
├── CHANGELOG.md                  # 变更日志
├── CLAUDE.md                     # Claude Code 项目指导
└── SECURITY.md                   # 安全政策
```

---

## 开发指南

### 运行测试

```bash
# 运行所有测试
make test

# 运行带覆盖率的测试
make test-coverage

# 运行基准测试
make test-bench
```

### 代码规范

```bash
# 代码格式化
make fmt

# 静态检查
make vet

# 代码检查（需要 golangci-lint）
make lint
```

### 构建

```bash
# 构建所有二进制文件
make build-all

# 交叉编译
make cross-compile

# 构建 Docker 镜像
make docker-build
```

---

## 常见问题 (FAQ)

**Q: 为什么选择 Ed25519 而不是 RSA/ECDSA？**

A: Ed25519 提供更高的安全性和性能：

- 签名大小仅 64 字节（RSA-2048 需要 256 字节）
- 验证速度比 ECDSA 快约 5 倍
- 内置抗侧信道攻击保护

**Q: 如何处理密钥轮换？**

A: 使用 KeyID 机制，每个签名包含密钥 ID，客户端可以支持多个公钥：

```go
syncer.PublicKeys = map[string]*crypto.PublicKey{
    "k1_2024": oldPublicKey,
    "k2_2024": newPublicKey,
}
```

**Q: 支持多少个围栏？**

A: 理论上无上限。实测：

- 10,000 个围栏：全量快照约 1.5 MB，查询 < 2ms
- 100,000 个围栏：全量快照约 15 MB，查询 < 5ms

**Q: 能否在没有网络的情况下使用？**

A: 可以。SDK 会使用本地缓存的围栏数据继续工作，网络恢复后自动同步。

---

## 路线图

- [x] 核心数据结构
- [x] Ed25519 签名验证
- [x] Merkle Tree 实现
- [x] R-Tree 空间索引
- [x] 二进制差分
- [x] HTTP 同步
- [x] 发布工具
- [x] SDK 示例
- [ ] CI/CD 流水线
- [ ] C++ SDK
- [ ] 性能基准测试
- [ ] Web 管理界面

---

## 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可证。

```
Copyright 2024-2025 Geofence-Updater-Lite Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

---

## 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何参与贡献。

---

## 致谢

本项目借鉴了以下开源项目的设计思路：

- [Git](https://git-scm.com/) - Merkle Tree 版本管理思想
- [bsdiff](https://www.daemonology.net/bsdiff/) - 二进制差分算法
- [go-polyline](https://github.com/twpayne/go-polyline) - 坐标压缩算法
