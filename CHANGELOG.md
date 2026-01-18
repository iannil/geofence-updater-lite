# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- 完整的 HTTP 客户端实现 (`pkg/client/http.go`)
  - Manifest 下载
  - Delta 和 Snapshot 下载
  - 签名验证
  - 进度回调
  - 重试机制

- 完整的同步逻辑实现 (`pkg/sync/syncer.go`)
  - 自动轮询
  - 版本对比
  - 智能更新策略（Delta vs Snapshot）
  - 数据库更新
  - 同步结果报告

- 完整的发布工具实现 (`pkg/publisher/publisher.go`)
  - 围栏签名
  - Merkle Tree 生成
  - Delta Patch 生成
  - 文件输出

- 二进制差分模块 (`pkg/binarydiff/delta.go`)
  - 自定义 diff 算法
  - 补丁应用
  - 哈希验证

### Changed

- 更新 README.md 为开源项目文档
- 添加 Apache 2.0 许可证
- 添加贡献指南 (CONTRIBUTING.md)
- 添加安全政策 (SECURITY.md)
- 添加 Makefile 简化开发

### Fixed

- 修复 merkle.go 编译错误（未定义的 `parent` 变量）
- 修复 binarydiff.go 测试语法错误
- 修复 version/manager.go 导入路径错误
- 修复 syncer.go 中的类型不匹配
- 修复 publisher.go 中的字段引用错误
- 修复 crypto 包缺失的 SHA-256 辅助函数

### Tests

- 所有核心模块测试通过
- 测试覆盖率约 90%

---

## [1.0.0] - 2025-01-18

### 首次发布

**核心功能**

- ✅ Ed25519 数字签名系统
- ✅ Merkle Tree 版本管理
- ✅ R-Tree 空间索引
- ✅ 二进制差分更新算法
- ✅ HTTP 客户端下载
- ✅ 自动同步机制
- ✅ 发布工具 (CLI)
- ✅ 客户端 SDK (示例)

**技术栈**

- Go 1.21+
- SQLite + R-Tree
- Ed25519
- SHA-256
- HTTP/HTTPS

**测试**

- 单元测试覆盖率约 90%
- 所有核心模块测试通过

---

## [0.0.1] - 2025-01-17

### 初始版本

- 数据结构定义
- 基础密码学模块
- 存储层实现
- 空间查询算法
