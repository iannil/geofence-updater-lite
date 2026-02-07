# 项目路线图

> 最后更新: 2026-02-07

## 当前状态

**项目完成度: ~95%**

核心功能已全部实现，进入优化和完善阶段。

---

## Phase 0: 项目初始化 ✅

**状态**: 已完成

- [x] 技术架构设计
- [x] 数据结构设计
- [x] 协议设计
- [x] SDK 接口规范
- [x] 初始化 Git 仓库
- [x] 选择 Go 作为主要开发语言
- [x] 创建项目基础结构

---

## Phase 1: 基础设施搭建 ✅

**状态**: 已完成

- [x] 初始化 Go module
- [x] 搭建基础目录结构
- [x] 配置开发环境
- [x] 设置 CI/CD 流水线 (GitHub Actions)
- [x] 编写基础测试框架

**交付物**:
- ✅ 可编译的项目骨架
- ✅ 基础测试框架
- ✅ `.github/workflows/ci.yml` CI/CD 配置

---

## Phase 2: 核心数据结构 ✅

**状态**: 已完成

- [x] 实现 Fence Item 数据结构 (`pkg/geofence/types.go`)
- [x] 实现 Manifest 数据结构 (`pkg/geofence/manifest.go`)
- [x] 实现 Protobuf 序列化 (`pkg/protocol/protobuf/`)
- [x] 实现数据签名/验证 Ed25519 (`pkg/crypto/ed25519.go`)
- [x] 单元测试 (覆盖率 ~114%)

**交付物**:
- ✅ 完整的数据结构定义
- ✅ Protobuf 序列化/反序列化
- ✅ Ed25519 数字签名验证模块

---

## Phase 3: 存储与索引 ✅

**状态**: 已完成

- [x] SQLite 本地存储封装 (`pkg/storage/storage.go`)
- [x] R-Tree 空间索引集成
- [x] 地理围栏查询 API
- [x] 事务支持
- [x] 单元测试 (覆盖率 ~96%)

**交付物**:
- ✅ 可用的本地存储模块
- ✅ 毫秒级空间查询能力

---

## Phase 4: Merkle Tree 与增量更新 ✅

**状态**: 已完成

- [x] Merkle Tree 实现 (`pkg/merkle/merkle.go`)
- [x] 版本比较逻辑
- [x] Delta Patch 生成算法 (`pkg/binarydiff/delta.go`)
- [x] Delta Patch 应用算法
- [x] 二进制差分工具
- [x] 单元测试 (覆盖率 ~106%)

**交付物**:
- ✅ 完整的 Merkle Tree 实现
- ✅ 增量更新机制
- ✅ 差分工具

---

## Phase 5: 客户端 SDK ✅

**状态**: 已完成

- [x] HTTP 下载模块 (`pkg/client/http.go`)
- [x] 轮询机制 (`pkg/sync/syncer.go`)
- [x] 签名验证集成
- [x] 更新应用逻辑
- [x] R-Tree 查询 API
- [x] 错误处理与重试机制 (带 jitter)
- [x] 单元测试 (覆盖率 ~67%)

**交付物**:
- ✅ 完整的 Go 客户端 SDK
- ✅ API 使用示例 (`cmd/sdk-example/`)

**待完善**:
- [ ] HTTP Range 断点续传 (P3-8)

---

## Phase 6: 发布工具 (Publisher Tool) ✅

**状态**: 已完成

- [x] CLI 框架搭建 (`cmd/publisher/main.go`)
- [x] 围栏项创建命令 (`add`)
- [x] 签名命令 (`keys`)
- [x] Manifest 生成命令 (`publish`)
- [x] 初始化命令 (`init`)
- [x] 列表/删除命令 (`list`, `remove`)
- [x] 文档

**交付物**:
- ✅ 可用的 CLI 工具
- ✅ 使用文档

**待完善**:
- [ ] 上传命令（S3/OSS/CDN 集成）

---

## Phase 7: 测试与优化 🔄

**状态**: 进行中

- [x] 单元测试 (当前覆盖率 ~68%)
- [ ] 端到端测试
- [ ] 压力测试（模拟低带宽环境）
- [ ] 安全测试（签名伪造、重放攻击）
- [x] 性能优化 (R-Tree 索引、Protobuf 序列化)
- [ ] 性能基准测试 (P3-4)
- [ ] 模糊测试

**目标**:
- 提升测试覆盖率到 80%+
- 完成端到端集成测试
- 完成性能基准测试报告

---

## Phase 8: 部署与文档 🔄

**状态**: 部分完成

- [x] README 文档
- [x] CLAUDE.md 项目指南
- [x] CONTRIBUTING.md 贡献指南
- [x] SECURITY.md 安全策略
- [ ] 完整 API 文档生成
- [ ] 集成指南（针对无人机厂商）
- [ ] 示例项目
- [ ] Release v1.0.0

---

## 待实现功能

### 高优先级

| 功能 | 优先级 | 状态 | 说明 |
|------|--------|------|------|
| 测试覆盖率提升 | P1 | 进行中 | 目标 80%+ |
| 端到端测试 | P1 | 待开始 | Playwright / 集成测试 |

### 中优先级

| 功能 | 优先级 | 状态 | 说明 |
|------|--------|------|------|
| 性能基准测试 | P2 | 待开始 | 验证毫秒级查询声明 |
| Polyline 压缩 | P2 | 待开始 | 坐标序列压缩 |
| 断点续传 | P2 | 待开始 | HTTP Range 请求 |

### 低优先级

| 功能 | 优先级 | 状态 | 说明 |
|------|--------|------|------|
| C++ SDK | P3 | 待开始 | 跨语言支持 |
| Web 管理界面 | P3 | 待开始 | 可视化管理 |
| CDN 上传集成 | P3 | 待开始 | S3/OSS 自动上传 |

---

## 技术债务追踪

详见 `docs/issues/remaining-issues.md`

### 已修复

- [x] P0-1: Ed25519 panic 问题
- [x] P0-2: 并发竞态条件
- [x] P0-3: 事务一致性
- [x] P0-4: 签名验证跳过
- [x] P1-4: 事务回滚模式
- [x] P1-5: UpdateFence 错误处理
- [x] P2-1: 错误返回值处理
- [x] P2-3: 网络重试 Jitter
- [x] P2-5: KeyID 长度

### 待修复

- [ ] P1-1: 补充单元测试
- [ ] P3-1: Merkle Tree findParent 性能
- [ ] P3-4: 性能基准测试

---

## 版本规划

### v0.1.0 (当前)

- 核心功能完成
- Go SDK 可用
- Publisher CLI 可用

### v1.0.0 (计划中)

- 测试覆盖率 80%+
- 完整文档
- 性能基准验证
- 生产环境就绪

### v1.1.0 (未来)

- C++ SDK
- Web 管理界面
- CDN 集成
