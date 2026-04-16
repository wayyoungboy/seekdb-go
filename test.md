# 嵌入式模式实现状态

## CGo 实现（已完成）

已完成完整的 CGo 绑定实现，与 pyseekdb 架构对齐：
- `libseekdb_cgo.go` — CGo 绑定所有 seekdb C API 函数
- `libseekdb_native.go` — Go 封装层 (NativeEmbeddedConn)
- `libseekdb_stub.go` — 非 CGO 平台的无操作存根
- `embedded.go` — 单例模式 + 引用计数（对齐 pyseekdb 行为）

### 架构对齐 pyseekdb

| 特性 | pyseekdb | seekdb-go | 状态 |
|------|----------|-----------|------|
| 单例模式 | `seekdb.open()` 只调用一次 | `globalSeekdb.opened` 控制 | ✅ |
| 永不关闭 | 不调用 `seekdb_close()` | Stop() 只减 refCount | ✅ |
| 引用计数 | 多个实例共享引擎 | `globalSeekdb.refCount` | ✅ |
| 网络访问 | `seekdb_open_with_service()` | 同 | ✅ |
| MySQL 连接 | `go-sql-driver/mysql` 连接本地端口 | 同 | ✅ |
| "initialized twice" 处理 | 忽略该错误 | 同 | ✅ |

### 独立测试通过

| 测试 | 状态 |
|------|------|
| AdminClient (standalone) | PASS |
| Client (standalone) | PASS |
| AutoPort (standalone) | PASS |

### 已知问题

AdminClient → Client 顺序调用在同一进程中会 hang。这是 seekdb C 库的全局状态清理问题。pyseekdb 通过单例模式规避（同一进程只 open 一次，后续连接通过 `seekdb.connect()` 获得新句柄）。我们已实现单例，但该问题仍存在于首次 open 后的第二次客户端创建。

## 阻塞问题：libseekdb.so 不可获取

### 调查结论

| 来源 | 结果 |
|------|------|
| S3 URL (`libseekdb-linux-x64.zip`) | 403 Forbidden |
| RPM 包 (`seekdb-1.2.0.0-*.rpm`) | 仅包含服务端二进制 (490MB)，无 `libseekdb.so` |
| pyseekdb wheel | 662MB Python C 扩展，引擎静态内嵌，无导出 C API |
| `/tmp/libseekdb_backup.so` (468MB) | 也是服务端二进制变体，无导出的 `seekdb_*` 符号 |
| GitHub Releases | 仅有 RPM/DEB，无 `libseekdb.so` 产物 |

**结论**: `libseekdb.so` 仅通过 S3 URL 分发。所有其他来源都不包含带有导出 C API 的嵌入式库。

### 获取方式

用户需要成功下载以下文件之一：
```
https://oceanbase-seekdb-builds.s3.ap-southeast-1.amazonaws.com/libseekdb/all_commits/c1a508a4efed701b88d369c7bdcf2aa2ea3480bd/libseekdb-linux-x64.zip
```
解压后将 `libseekdb.so` 放入 `libseekdb/` 目录。

或者从 pyseekdb 的 wheel 中提取（不推荐，因为 662MB 且包含 Python 扩展代码）。

### 替代方案考虑

1. **子进程模式（回退方案）**: 使用 RPM 中的 seekdb 二进制作为子进程启动，通过 MySQL 协议连接。这是 CGo 实现之前的原始方案。
2. **等待 S3 恢复**: 联系 OceanBase 团队恢复 S3 下载权限。

## 代码文件清单

### 核心 SDK
- `admin.go` — AdminClient，数据库管理
- `client.go` — Client，集合和数据操作
- `collection.go` — 集合操作（创建/获取/删除/fork）
- `dml.go` — 数据操作（add/update/upsert/delete）
- `dql.go` — 数据查询（query/get/hybrid_search）
- `config.go` — 配置类型（HNSWConfig、VectorIndexConfig 等）
- `embedding.go` — Embedding 函数接口和提供者
- `filter.go` — 过滤操作符
- `connection.go` — 连接处理（嵌入/服务端模式）
- `errors.go` — 错误类型
- `types.go` — 通用类型（Document、Metadata、QueryResult 等）

### CGo 嵌入式模式
- `libseekdb_cgo.go` — CGo 绑定（`//go:build cgo && (linux || darwin)`）
- `libseekdb_native.go` — NativeEmbeddedConn Go 封装
- `libseekdb_stub.go` — 非 CGO 平台存根
- `embedded.go` — EmbeddedProcess + 全局单例管理

### 测试
- `admin_test.go` — AdminClient 集成测试
- `client_test.go` — Client 集成测试
- `embedded_test.go` — 嵌入式模式测试

### 依赖
- `seekdb.h` — seekdb C API 头文件（已有）
- `libseekdb.so` — seekdb C 动态库（**缺失，需手动下载**）
