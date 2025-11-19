# 架构概览

1. 入口：`cmd/` 提供服务端与 worker 可执行程序。
2. 领域：`internal/` 按 app/domain/service/repository 层次解耦。
3. 通用能力：`pkg/` 暴露可复用组件。
4. 接口：`api/` 统一描述外部契约（REST/Proto）。
5. 运维：`configs/`、`deploy/`、`scripts/` 支撑配置与部署。
