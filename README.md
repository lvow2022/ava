# Ava

该仓库按照经典 Go 项目结构初始化，用于示例化服务端与工作进程的代码组织方式。目录说明：

- `cmd/`：每个可执行程序一个子目录，目前包含 `server` 与 `worker`。
- `internal/`：内部业务逻辑，按 app、domain、service、repository 分层。
- `pkg/`：可复用的通用库（配置、日志等）。
- `api/`：接口协议与文档（proto/openapi）。
- `configs/`：配置模板与环境样例。
- `deploy/`：容器化与编排示例。
- `scripts/`：自动化脚本。
- `build/`：构建或打包相关说明。
- `docs/`：设计与架构文档。
- `test/`：跨包集成测试资源。

## 使用

```bash
make run        # 启动示例 server
make worker     # 启动示例 worker
make test       # 运行所有测试
```
