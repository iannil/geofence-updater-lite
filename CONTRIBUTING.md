# Contributing to Geofence-Updater-Lite

感谢你的关注！本项目欢迎各种形式的贡献。

## 如何贡献

### 报告 Bug

请在 [Issues](../../issues) 提交问题，并填写以下模板：

```markdown
**Bug 报告**

**描述**
简要描述发生了什么

**复现步骤**
1. 执行 `...`
2. 看到 `...`
3. 出现 `...`

**环境信息**
- GUL 版本：
- Go 版本：
- 操作系统：

**期望行为**

**实际行为**

**日志**

**附加信息**
```

### 提交功能请求

请在 [Issues](../../issues) 提交功能请求。

## 开发环境搭建

### 1. Fork 项目并克隆

```bash
# Fork 项目
git clone https://github.com/yourname/geofence-updater-lite.git
cd geofence-updater-lite
cd pkg
go mod download
```

### 2. 创建功能分支

```bash
git checkout -b feature/your-feature-name
```

### 3. 开发并测试

```bash
# 运行测试
go test ./...

# 代码格式化
go fmt ./...

# 静态检查
go vet ./...
```

### 4. 提交 PR

```bash
git add .
git commit -m "Add your feature"
git push origin feature/your-feature-name
```

## 代码规范

### Go 代码风格

本项目遵循以下规范：

- [Effective Go](https://go.dev/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guidelines)

### 测试要求

- 新功能必须包含单元测试
- 测试覆盖率需保持在 80% 以上
- 确保所有测试通过

### 提交信息规范

提交格式：
```
<type>(<scope>): <subject>

<body>
```

示例：
```
fix(storage): 修复 R-Tree 查询中的边界检查 bug

- 修复点在边界上被错误包含的问题
- 添加边界测试用例
- 更新相关文档
```

### Pull Request Checklist

- [ ] 代码通过所有测试
- [ ] 代码通过 `go vet`
- [ ] 代码通过 `gofmt`
- [ ] 添加/更新了测试
- [ ] 添加/更新了文档
- [ ] 更新 README（如需要）
- [ ] 添加了适当的错误处理

## 编码标准

### 文件命名

- Go 文件：`snake_case`
- 测试文件：`xxx_test.go`
- 接口文件：`interface.go`
- 常量：`UPPER_CASE`

### 注释规范

```go
// Package crypto provides Ed25519 digital signature functionality.
//
// 该包实现了 Ed25519 数字签名和验证功能，用于：
// - 围栏项签名
// - 清单签名
// - 密钥管理
//
// 示例：
//
//    keyPair, err := crypto.GenerateKeyPair()
//    signature := crypto.Sign(keyPair.PrivateKey, data)
//    valid := crypto.Verify(keyPair.PublicKey, data, signature)
package crypto
```

### 错误处理

```go
// 函数签名
func (t *Tree) GetProof(fenceID string) ([][]byte, error) {
    // 实现...
    if fenceID == "" {
        return nil, fmt.Errorf("fenceID cannot be empty")
    }
    // ...
}
```

## 性能考虑

### 性能测试

如果你声称代码有性能改进，请包含基准测试结果：

```go
func BenchmarkGetProof(b *testing.B) {
    tree := createTestTree(1000)
    b.ResetTimer()
    for i := 0; b.N < 10000; i++ {
        _, _ = tree.GetProof("test-fence-001")
    }
    b.ReportAllocs()
}
```

### 性能目标

- 围栏检查：< 1ms (1000 次查询)
- Merkle Tree 构建：< 100ms (1000 个围栏)
- Delta 计算：< 50ms (1000 个围栏)

## 安全问题

### 安全漏洞报告

**请不要**在公开 Issue 中报告安全漏洞。

请发送邮件至：security@example.com

包含以下信息：
- 漏洞描述
- 影响范围
- 复现步骤
  (可选) 建议的修复方案

## 许可证

贡献的代码将根据项目许可证进行许可。

详见：[Apache License 2.0](./LICENSE)

---

**欢迎参与！你的贡献将使项目变得更好。**
