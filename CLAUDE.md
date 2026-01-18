# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此代码仓库中工作时提供指导。

## 项目概述

Geofence-Updater-Lite (GUL) 是一个轻量级、高可靠的地理围栏数据同步系统，专为在低带宽、不稳定网络环境下运行的无人机/无人驾驶飞行器设计（例如 GPRS 级别的连接，带宽仅几 KB/s）。

核心设计理念：

- 使用 Merkle Tree 实现 Git 风格的增量更新，最小化数据传输
- 去中心化 CDN 友好分发（仅静态文件，零服务器成本）
- 安全优先架构，使用 Ed25519 数字签名实现防篡改验证

## 架构设计

### 双组件架构

1. 发布工具 (Publisher Tool，服务端)

- 管理员使用的 CLI 工具或 Web 后台
- 生成带 Ed25519 数字签名的围栏项
- 创建 Merkle 树结构以实现高效增量更新
- 发布到静态存储（CDN/S3/OSS）- 无需后端 API

2. 无人机 SDK (Drone SDK，客户端)

- 运行在无人机机载电脑或遥控器 APP 上
- 轮询 `manifest.json` 检查版本更新
- 支持增量更新（补丁）或全量快照
- 应用更新前离线验证数字签名
- 使用 R-Tree 空间索引实现毫秒级围栏查询

### 数据格式

- Fence Item（围栏项）: 单个地理围栏限制，包含类型、几何形状（多边形）、起止时间戳、优先级和签名
- Manifest（清单）: 小型 JSON 文件，包含版本号、根哈希、增量 URL 和快照 URL
- 序列化: 优先使用 Protobuf 或 FlatBuffers 替代 JSON，可减少 60% 体积
- 坐标压缩: 使用 Polyline 算法压缩坐标序列

## 计划技术栈

- 编程语言: Go 和/或 C++
- 本地存储: SQLite 用于持久化存储
- 空间索引: R-Tree 用于快速地理空间查询
- 密码学: Ed25519 用于数字签名
- 部署方式: 静态文件托管（CDN/S3/OSS）

## 项目状态

目前处于概念设计/规范阶段。尚未实现源代码、构建系统或依赖项。仓库仅包含 README.md 中的架构设计文档。

## 关键约束与要求

1. 带宽: 系统必须在 GPRS 级别连接（几 KB/s）下正常工作
2. 安全性: 所有围栏项必须可使用内置公钥验证，无论数据来源
3. 性能: 地理围栏检查必须在微秒/毫秒级完成
4. 可靠性: 必须优雅处理网络中断

## SDK 接口设计（计划中）

```go
updater, _ := gul.NewUpdater(&gul.Config{
    ManifestURL: "https://cdn.example.com/geofence/manifest.json",
    PublicKey:   "Ed25519_Public_Key_...",
    StorePath:   "/data/geofence.db",
})

go updater.StartAutoSync(1 * time.Minute)

allowed, restriction := updater.Check(lat, lon)
```

## 文档语言

主要文档使用中文编写。技术讨论和代码注释应尊重此上下文。
