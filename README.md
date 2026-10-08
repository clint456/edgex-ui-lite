# edge-hy-console

EdgeX 管理控制台：**Gin 后端 + 静态 HTML/CSS/JS 前端**，直接调用 EdgeX 微服务 REST API，
不依赖、也不经过 `edgex-ui-go` 代理。

## 架构

```
cmd/edge-hy-console/     入口，读取配置并启动 Gin
internal/config/         运行期连接配置（可在「连接设置」页面修改）
internal/edgex/          各微服务直连客户端
internal/api/            Gin JSON API + go:embed 静态资源
internal/web/static/     前端（index.html / app.css / app.js，无构建步骤）
```

直连的微服务与默认地址：

| 服务 | 默认地址 | 用途 |
| --- | --- | --- |
| core-metadata | http://localhost:59881 | 设备 / 设备服务 / 设备模型 / 自动发现 |
| core-data | http://localhost:59880 | 事件 / 读数 |
| core-command | http://localhost:59882 | 设备命令 GET/PUT |
| support-mappings | http://localhost:59883 | 南北向映射 |
| core-keeper | http://localhost:59890 | 注册服务列表 `/api/v3/registry/all` |
| sys-mgmt-agent | http://localhost:59886 | 服务指标 / 配置 / 启停（可选） |

## 功能

控制台概览、系统服务、元数据（设备·设备服务·设备模型·自动发现）、数据中心（事件·读数）、
映射管理、连接设置。支持元数据增删改、设备命令执行、设备模型 YAML 导入/导出、映射查询与批量。

## 运行

```bash
go run ./cmd/edge-hy-console      # 默认监听 :8080
# 或
bash run.sh
```

环境变量（均可在页面「连接设置」中覆盖）：

| 变量 | 默认值 |
| --- | --- |
| `EDGEX_LISTEN` | `:8080` |
| `EDGEX_METADATA_URL` | `http://localhost:59881` |
| `EDGEX_DATA_URL` | `http://localhost:59880` |
| `EDGEX_COMMAND_URL` | `http://localhost:59882` |
| `EDGEX_MAPPINGS_URL` | `http://localhost:59883` |
| `EDGEX_KEEPER_URL` | `http://localhost:59890` |
| `EDGEX_SYSTEM_URL` | `http://localhost:59886` |
| `EDGEX_TOKEN` | 空（非安全模式） |
| `EDGEX_TIMEOUT` | `8`（秒） |

## HTTP API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET/PUT | `/api/config` | 读取 / 更新连接配置 |
| GET | `/api/health` | 各服务 ping 健康检查 |
| GET | `/api/services` | core-keeper 注册服务列表 |
| GET | `/api/overview` | 概览计数 + 健康 |
| GET/POST/PUT | `/api/metadata/:kind` | 列表 / 新建 / 更新（kind: device、service、profile、watcher） |
| GET/DELETE | `/api/metadata/:kind/:name` | 详情 / 删除 |
| GET | `/api/devices/:name/commands` | 设备命令元数据 |
| GET/PUT | `/api/devices/:name/commands/:command` | 执行命令 |
| GET | `/api/data/:kind` | 事件 / 读数列表（kind: event、reading） |
| GET | `/api/data/:kind/count` | 事件 / 读数总数 |
| GET/POST/PUT | `/api/mappings` | 映射列表 / 新建 / 更新 |
| DELETE | `/api/mappings/:id` | 删除映射 |
| GET | `/api/mappings/lookup` | 单点查询（direction/device/profile/resource） |
| POST | `/api/mappings/batch` | 批量查询 |
| DELETE | `/api/mappings/scope/:side/:kind/:name` | 按范围删除 |
| GET | `/api/system/metrics` `/api/system/config` | sys-mgmt-agent 指标 / 配置 |
| POST | `/api/system/operation` | 服务启停 |
| GET | `/api/profile/:name/yaml` | 导出设备模型 YAML |
| POST | `/api/profile/upload` | 导入设备模型 YAML |

## 与 edgex-ui-go 的差异

- **无代理层**：直接访问各微服务，路径以 `edgex-go` 为准（`/api/v3/...`）。
- 注册服务列表来自 core-keeper，而不是 Consul。
- 不再有 Scheduler / Notifications / RuleEngine 栏目（与精简后的功能集一致）。
- 连接配置保存在进程内存中，通过「连接设置」页面修改即可生效。
