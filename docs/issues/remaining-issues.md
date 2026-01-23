# Geofence-Updater-Lite 剩余问题清单

> 创建日期: 2026-01-23
> 项目版本: v0.1.0
> 整体完成度: ~95%

---

## 概述

本文档整理了 Geofence-Updater-Lite 项目的剩余问题，按优先级从高到低、风险从大到小排列，便于逐条修复。

### 统计汇总

| 优先级 | 数量 | 范围 | 状态 |
|--------|------|------|------|
| **P0** | 4 | 安全/稳定性 | 待修复 |
| **P1** | 5 | 功能/重大缺陷 | 待修复 |
| **P2** | 5 | 代码质量 | 待修复 |
| **P3** | 8 | 优化增强 | 可选 |
| **总计** | **22** | | |

### 修复顺序建议

1. **第一阶段**: P0-1 ~ P0-4 (安全问题)
2. **第二阶段**: P1-1 ~ P1-5 (功能完善)
3. **第三阶段**: P2-1 ~ P2-5 (代码质量)
4. **第四阶段**: P3-* (优化增强，按需)

---

## P0 - 阻断性问题（安全/稳定性）

### P0-1. Ed25519 模块使用 panic 导致程序崩溃

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/crypto/ed25519.go:124, 140` |
| **风险等级** | 高 |
| **影响范围** | 全局 - 整个程序崩溃 |

**问题描述**：
`Sign()` 方法在私钥不可用时直接调用 `panic()`，生产环境下会导致整个程序崩溃，可被攻击者利用造成 DoS。

**当前代码**：
```go
func (kp *KeyPair) Sign(data []byte) []byte {
    if kp.PrivateKey == nil {
        panic("private key is required for signing")  // 第124行
    }
    // ...
}
```

**修复建议**：
改为返回 `([]byte, error)` 并由调用方处理：
```go
func (kp *KeyPair) Sign(data []byte) ([]byte, error) {
    if kp.PrivateKey == nil {
        return nil, errors.New("private key is required for signing")
    }
    // ...
}
```

---

### P0-2. 并发竞态条件 - currentVer 访问

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/sync/syncer.go:106-109` |
| **风险等级** | 高 |
| **影响范围** | 同步模块 - 版本判断错误 |

**问题描述**：
`currentVer` 是 `uint64` 类型，使用普通赋值/读取而非原子操作。在并发环境下可能导致数据不一致或版本判断错误。

**当前代码**：
```go
// 第106-109行
if manifest.Version <= s.currentVer {
    result.UpToDate = true
    return result
}
// RACE: 其他 goroutine 可能在比较后修改 s.currentVer
```

**修复建议**：
使用 `sync/atomic.Uint64` 或加锁保护：
```go
type Syncer struct {
    currentVer atomic.Uint64  // 替换 uint64
    // ...
}

// 读取时
if manifest.Version <= s.currentVer.Load() {
    // ...
}

// 写入时
s.currentVer.Store(newVersion)
```

---

### P0-3. 事务一致性问题 - R-Tree 与 fence 表不同步

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/storage/storage.go:209-212, 285-287` |
| **风险等级** | 高 |
| **影响范围** | 存储层 - 数据库状态不一致 |

**问题描述**：
插入 fence 和更新 R-Tree 索引不在同一事务中。程序中断时可能导致数据库状态不一致（fence 存在但 R-Tree 索引缺失，或反之）。

**当前代码**：
```go
// AddFence 函数
result, err := s.db.ExecContext(ctx, insertSQL, ...)  // 第189行
if err != nil { return err }

// 单独插入 R-Tree，不在同一事务中
_, err = s.db.ExecContext(ctx, rtreeInsertSQL, ...)   // 第209行
```

**修复建议**：
将两个操作包装在同一事务中：
```go
tx, err := s.db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback()

_, err = tx.ExecContext(ctx, insertSQL, ...)
if err != nil { return err }

_, err = tx.ExecContext(ctx, rtreeInsertSQL, ...)
if err != nil { return err }

return tx.Commit()
```

---

### P0-4. 未配置公钥时跳过签名验证

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/client/http.go:119-122` |
| **风险等级** | 高 |
| **影响范围** | 安全 - 可注入恶意清单 |

**问题描述**：
当 `PublicKeyHex` 未配置时完全跳过签名验证。攻击者可以利用此漏洞注入恶意清单。

**当前代码**：
```go
// 第119-122行
if c.publicKey == nil {
    // Skip verification if no public key configured
    return manifest, nil
}
```

**修复建议**：
生产环境应强制要求公钥配置：
```go
if c.publicKey == nil {
    if c.cfg.RequireSignature {  // 新增配置项
        return nil, errors.New("signature verification required but no public key configured")
    }
    log.Warn("WARNING: signature verification skipped - not recommended for production")
}
```

---

## P1 - 高优先级（功能缺失/重大缺陷）

### P1-1. 5个核心模块完全缺少单元测试

| 属性 | 值 |
|------|-----|
| **影响模块** | 5个 |
| **代码行数** | 1,684 行 |
| **风险等级** | 中-高 |

**未测试模块列表**：

| 模块 | 文件 | 行数 | 功能 |
|------|------|------|------|
| HTTP客户端 | `pkg/client/http.go` | 358 | 网络请求、下载、重试 |
| 同步逻辑 | `pkg/sync/syncer.go` | 366 | 增量同步、自动轮询 |
| 发布流程 | `pkg/publisher/publisher.go` | 370 | 版本发布、签名 |
| 版本管理 | `pkg/version/manager.go` | 333 | 版本生命周期 |
| 数据转换 | `pkg/converter/converter.go` | 257 | 格式转换 |

**问题描述**：
38.5% 的模块无测试覆盖。难以保证功能正确性，重构时容易引入 bug。

**修复建议**：
为每个模块添加单元测试，优先覆盖核心路径。建议创建：
- `pkg/client/http_test.go`
- `pkg/sync/syncer_test.go`
- `pkg/publisher/publisher_test.go`
- `pkg/version/manager_test.go`
- `pkg/converter/converter_test.go`

---

### P1-2. CI/CD 流水线完全缺失

| 属性 | 值 |
|------|-----|
| **当前状态** | 无自动化流程 |
| **风险等级** | 中 |

**问题描述**：
无 GitHub Actions / GitLab CI 配置，无法自动化测试、构建、发布。

**修复建议**：
创建 `.github/workflows/ci.yml`：
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: make test
      - run: make lint
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: make cross-compile
```

---

### P1-3. Protobuf 已定义但实际使用 JSON

| 属性 | 值 |
|------|-----|
| **Proto 文件** | `pkg/protocol/protobuf/*.proto` |
| **实际使用** | `json.Marshal()` |
| **风险等级** | 中 |

**问题描述**：
定义了 Protobuf 但代码使用 `json.Marshal()`（见 `pkg/merkle/merkle.go:90`）。未实现 CLAUDE.md 声称的 60% 体积减少。

**当前代码**：
```go
// pkg/merkle/merkle.go:90
data, err := json.Marshal(fence)  // 使用 JSON 而非 Protobuf
```

**修复建议**：
选择其一：
1. 全面切换到 Protobuf 序列化
2. 或移除未使用的 proto 文件并更新文档

---

### P1-4. 事务回滚模式错误

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/sync/syncer.go:243-279` |
| **风险等级** | 中 |

**问题描述**：
`defer tx.Rollback()` 在 `Commit()` 成功后仍会执行。虽然 SQLite 通常会忽略已提交事务的回滚，但这是不良实践，可能导致边界条件下的问题。

**当前代码**：
```go
tx, err := s.store.BeginTx(ctx)
if err != nil { return err }
defer tx.Rollback()  // 总是执行，即使 Commit 成功

// ... 操作 ...

if err := tx.Commit(); err != nil {
    return err
}
return nil  // 之后 defer tx.Rollback() 仍会执行
```

**修复建议**：
使用条件回滚模式：
```go
tx, err := s.store.BeginTx(ctx)
if err != nil { return err }

committed := false
defer func() {
    if !committed {
        tx.Rollback()
    }
}()

// ... 操作 ...

if err := tx.Commit(); err != nil {
    return err
}
committed = true
return nil
```

---

### P1-5. UpdateFence 错误处理逻辑缺陷

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/sync/syncer.go:254-260` |
| **风险等级** | 中 |

**问题描述**：
假设 `UpdateFence` 失败意味着 fence 不存在，但可能是其他错误（如数据库损坏、权限错误）。这可能导致静默数据丢失。

**当前代码**：
```go
for _, f := range fences {
    err := s.store.UpdateFence(ctx, &f)
    if err != nil {
        // 假设不存在，直接添加
        if err := s.store.AddFence(ctx, &f); err != nil {
            return fmt.Errorf("failed to add fence %s: %w", f.ID, err)
        }
    }
}
```

**修复建议**：
检查具体错误类型：
```go
for _, f := range fences {
    err := s.store.UpdateFence(ctx, &f)
    if err != nil {
        if errors.Is(err, ErrFenceNotFound) {
            if err := s.store.AddFence(ctx, &f); err != nil {
                return fmt.Errorf("failed to add fence %s: %w", f.ID, err)
            }
        } else {
            return fmt.Errorf("failed to update fence %s: %w", f.ID, err)
        }
    }
}
```

---

## P2 - 中优先级（代码质量）

### P2-1. 多处忽略错误返回值

| 属性 | 值 |
|------|-----|
| **风险等级** | 中 |
| **数量** | 4+ 处 |

**问题位置**：

| 文件 | 行号 | 代码 |
|------|------|------|
| `pkg/publisher/publisher.go` | 167 | `manifestData, _ := manifest.MarshalBinaryForSigning()` |
| `pkg/publisher/publisher.go` | 354 | `os.Remove(storePath)` |
| `pkg/publisher/publisher.go` | 101 | `oldFences, _ := p.getCurrentFences(ctx)` |
| `pkg/storage/storage.go` | 279, 316 | `rows, _ := result.RowsAffected()` |

**修复建议**：
正确处理所有错误返回值：
```go
manifestData, err := manifest.MarshalBinaryForSigning()
if err != nil {
    return fmt.Errorf("failed to marshal manifest: %w", err)
}
```

---

### P2-2. .golangci.yml 禁用了关键安全检查

| 属性 | 值 |
|------|-----|
| **文件** | `.golangci.yml:63-65` |
| **风险等级** | 中 |

**问题描述**：
禁用了 `G104`（未处理错误）和 `G307`（defer close）检查。

**当前配置**：
```yaml
gosec:
  excludes:
    - G104  # Errors unhandled
    - G307  # Deferring file close
```

**修复建议**：
移除这些排除项，修复所有触发的警告。

---

### P2-3. 网络重试缺少 Jitter

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/client/http.go:299-324` |
| **风险等级** | 中 |

**问题描述**：
指数退避无随机抖动，所有客户端同时重试造成 "thundering herd" 问题。

**当前代码**：
```go
delay := time.Duration(1<<uint(i)) * time.Second  // 1, 2, 4, 8...
if delay > 30*time.Second {
    delay = 30 * time.Second
}
time.Sleep(delay)
```

**修复建议**：
添加随机 jitter：
```go
baseDelay := time.Duration(1<<uint(i)) * time.Second
jitter := time.Duration(rand.Float64() * 0.5 * float64(baseDelay))
delay := baseDelay + jitter
if delay > 30*time.Second {
    delay = 30 * time.Second
}
time.Sleep(delay)
```

---

### P2-4. 日志不一致且无结构化

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/sync/syncer.go:112, 118, 121, 135` |
| **风险等级** | 低-中 |

**问题描述**：
混用 `log.Printf()`，无日志级别，无 context。

**修复建议**：
引入结构化日志库（如 Go 1.21+ 的 `log/slog`）：
```go
import "log/slog"

slog.Info("sync completed",
    "version", manifest.Version,
    "fences_updated", len(fences),
)
```

---

### P2-5. KeyID 碰撞风险

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/crypto/ed25519.go:114-119` |
| **风险等级** | 低 |

**问题描述**：
仅使用 SHA-256 前 8 字节作为 KeyID，存在 64 位碰撞可能（虽然概率低）。

**当前代码**：
```go
hash := sha256.Sum256(kp.PublicKey)
kp.KeyID = hex.EncodeToString(hash[:8])  // 只取前8字节
```

**修复建议**：
增加到至少 16 字节：
```go
kp.KeyID = hex.EncodeToString(hash[:16])  // 128位
```

---

## P3 - 低优先级（优化增强）

### P3-1. Merkle Tree findParent O(n) 性能

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/merkle/merkle.go:218-237` |
| **影响** | 大型树性能下降 |

**问题**：递归遍历整棵树查找父节点，O(n) 复杂度。

**修复**：维护 parent 指针或使用 level-order 结构。

---

### P3-2. 存储层锁粒度过粗

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/storage/storage.go:63-64` |
| **影响** | 高并发性能 |

**问题**：所有操作共用单个 RWMutex。

**修复**：考虑按表或操作类型分离锁。

---

### P3-3. HTTP 连接池无大小限制

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/client/http.go:46-52` |
| **影响** | 资源管理 |

**问题**：`MaxIdleConns: 10` 但无 `MaxConnsPerHost`。

**修复**：添加 `MaxConnsPerHost` 限制。

---

### P3-4. 缺少性能基准测试

| 属性 | 值 |
|------|-----|
| **影响** | 无法验证性能声明 |

**问题**：无法验证 "毫秒级查询" 的性能声明。

**修复**：添加 `*_bench_test.go` 文件。

---

### P3-5. Merkle 证明验证未明确哈希组合顺序

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/merkle/merkle.go:263-265` |
| **影响** | 安全性 |

**问题**：未指定左右节点哈希的组合顺序。

**修复**：在 Proof 结构中添加位置标记。

---

### P3-6. 常时间比较未使用

| 属性 | 值 |
|------|-----|
| **文件** | `pkg/crypto/ed25519.go:83-87` |
| **影响** | 时序攻击风险 |

**问题**：使用普通字节比较，存在时序攻击风险。

**修复**：使用 `subtle.ConstantTimeCompare()`。

---

### P3-7. Polyline 坐标压缩未实现

| 属性 | 值 |
|------|-----|
| **规范** | CLAUDE.md |
| **影响** | 带宽优化 |

**问题**：CLAUDE.md 提到的坐标压缩功能未实现。

**修复**：添加 Polyline 编码/解码支持。

---

### P3-8. 断点续传未实现

| 属性 | 值 |
|------|-----|
| **影响** | 大文件下载 |

**问题**：大文件下载时无断点续传支持。

**修复**：添加 HTTP Range 请求支持。

---

## 附录

### 已测试模块覆盖情况

| 模块 | 实现行数 | 测试行数 | 覆盖比 | 状态 |
|------|---------|---------|-------|------|
| crypto | 209 | 239 | 114% | 优秀 |
| config | 221 | 362 | 164% | 优秀 |
| merkle | 384 | 223 | 58% | 良好 |
| storage | 547 | 525 | 96% | 优秀 |
| binarydiff | 449 | 477 | 106% | 优秀 |
| geofence | 432 | 825 | 191% | 优秀 |

### 问题跟踪

修复后请在此记录：

- [x] P0-1: Ed25519 panic 问题 ✓ (Sign 方法改为返回 error)
- [x] P0-2: 并发竞态条件 ✓ (使用 atomic.Uint64 和 RWMutex)
- [x] P0-3: 事务一致性 ✓ (fence 和 R-Tree 操作包装在事务中)
- [x] P0-4: 签名验证跳过 ✓ (强制要求公钥或显式跳过)
- [ ] P1-1: 单元测试补充
- [ ] P1-2: CI/CD 配置
- [ ] P1-3: Protobuf 使用
- [x] P1-4: 事务回滚模式 ✓ (使用 committed flag)
- [x] P1-5: UpdateFence 错误处理 ✓ (检查 ErrFenceNotFound)
- [x] P2-1: 错误返回值处理 ✓ (正确处理 getCurrentFences, writeManifest 等)
- [x] P2-2: golangci.yml 配置 ✓ (移除 G104, G307 排除项)
- [x] P2-3: 网络重试 Jitter ✓ (添加随机抖动)
- [ ] P2-4: 结构化日志 (可选优化)
- [x] P2-5: KeyID 长度 ✓ (从 8 字节增加到 16 字节)
- [ ] P3-1: Merkle findParent 性能
- [ ] P3-2: 存储层锁粒度
- [ ] P3-3: HTTP 连接池限制
- [ ] P3-4: 性能基准测试
- [ ] P3-5: Merkle 证明顺序
- [ ] P3-6: 常时间比较
- [ ] P3-7: Polyline 压缩
- [ ] P3-8: 断点续传
