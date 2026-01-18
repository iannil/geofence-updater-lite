# 技术规范

## 核心设计理念

1. **Git 思想 (Merkle Tree)**：使用 Merkle Tree 或 Hash Chain 管理版本，确保客户端只需下载"差异部分"（Delta）
2. **去中心化分发 (CDN/IPFS friendly)**：核心数据为静态文件，可部署在 CDN、OSS 甚至 IPFS 上，无需复杂后端 API
3. **数字签名 (Security First)**：每条禁飞指令必须由权威机构（CA）签名，无人机只认签名，不认下载源

---

## 数据结构设计

### 围栏项 (Fence Item)

```json
{
  "id": "fence-20240310-001",       // 唯一ID
  "type": "TEMP_RESTRICTION",       // 类型：临时管控/永久禁飞/限高
  "geometry": { ... },              // 多边形坐标 (Polygon)
  "start_ts": 1709880000,           // 生效时间
  "end_ts": 1709990000,             // 失效时间 (过期自动清理)
  "priority": 10,                   // 优先级 (覆盖旧规则)
  "signature": "Ed25519_Sig_..."    // 权威机构对上述字段的签名
}
```

### 索引文件 (Manifest)

```json
{
  "version": 1024,                  // 当前全局版本号 (递增)
  "timestamp": 1709882231,
  "root_hash": "sha256_of_tree",    // 默克尔树根哈希
  "delta_url": "/patches/v1023_to_v1024.bin", // 增量包下载地址
  "snapshot_url": "/snapshots/v1024.bin"      // 全量包下载地址 (供新设备用)
}
```

---

## 架构设计

### 服务端 (Publisher Tool)

* CLI 工具或 Web 后台，供管理员使用
* **处理流程**：
  1. 生成 Fence Item JSON
  2. 用私钥对 Item 进行签名
  3. 计算新的 Merkle Tree，生成新的 Manifest
  4. 生成从上一版本到当前版本的 Delta Patch
  5. 将所有生成的文件上传到对象存储（AWS S3 / 阿里云 OSS）

### 客户端 (Drone SDK)

* 运行在无人机机载电脑或遥控器 APP 上
* **处理流程**：
  1. **Polling**: 每隔 N 分钟下载 `manifest.json`
  2. **Check**: 对比本地 `version` 和远端 `version`
  3. **Fetch**:
     - 版本差 1：下载 `delta_url`（极小，可能只有几 KB）
     - 版本差太多：下载 `snapshot_url`（全量）
  4. **Verify**: 校验数字签名，签名不对则丢弃
  5. **Apply**: 更新本地围栏数据库（SQLite / 内存 R-Tree）

---

## 关键技术细节

### 极低带宽优化

* **二进制序列化**：使用 Protobuf 或 FlatBuffers 代替 JSON，体积减少 60%
* **多边形压缩**：使用 Polyline Algorithm 压缩坐标点序列
* **差分更新**：假设禁飞区没变，只是延期了 1 小时，只下发"修改了 end_ts 字段"的指令

### 空间索引 (R-Tree)

* SDK 内置 R-Tree 引擎
* 将所有多边形的 Bounding Box 放入 R-Tree
* 查询输入当前 GPS 坐标，R-Tree 能在微秒级返回是否在某个禁飞区内

### 安全体系

* **离线验签**：无人机出厂时内置权威机构的公钥，无论数据从哪里下载，签名解不开就拒绝执行
* **防重放攻击**：数据包包含 `timestamp` 和 `version`，飞控拒绝回退到旧版本

---

## SDK 接口设计 (Go)

```go
// 初始化
updater, _ := gul.NewUpdater(&gul.Config{
    ManifestURL: "https://cdn.gov.cn/uav/manifest.json",
    PublicKey:   "Ed25519_Public_Key_...",
    StorePath:   "/data/geofence.db",
})

// 启动同步协程
go updater.StartAutoSync(1 * time.Minute)

// 飞控核心循环调用：判断当前点是否禁飞
func flightLoop() {
    currentLat, currentLon := gps.GetLocation()

    // 毫秒级查询
    allowed, restriction := updater.Check(currentLat, currentLon)

    if !allowed {
        drone.Hover() // 悬停
        drone.Alert("进入禁飞区: " + restriction.Reason)
    }
}
```

---

## 技术栈

| 组件 | 技术选型 |
|------|----------|
| 编程语言 | Go 和/或 C++ |
| 本地存储 | SQLite |
| 空间索引 | R-Tree |
| 密码学 | Ed25519 |
| 数据序列化 | Protobuf / FlatBuffers |
| 部署方式 | 静态文件托管（CDN/S3/OSS） |
