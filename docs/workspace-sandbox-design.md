# FreeDinnerAgent 用户级 Workspace Sandbox 设计

## 1. 设计目标

FreeDinnerAgent 不只应该会聊天，也应该能在用户授权的工作区里处理文件、运行代码、生成报告、整理资料和执行 CLI 操作。为了避免 Agent 直接接触宿主机文件系统，需要为每个用户提供独立的 Workspace Sandbox。

目标：

- 用户可以选择是否启用 workspace。
- 每个用户拥有独立目录和资源配额。
- Agent 只能在该用户 workspace 内读写文件。
- CLI 执行必须受 CPU、内存、磁盘、网络和超时限制。
- Workspace 长时间不活跃可以归档或销毁。
- 所有文件操作和命令执行都写审计日志。
- 后续可从本地目录 MVP 升级到 Docker、Podman、nsjail、Firecracker 等更强隔离方案。

## 2. 总体架构

```text
Agent Tool Call
  -> Workspace Tool Router
  -> Workspace Service
  -> Sandbox Manager
  -> Per-user Workspace
  -> File / CLI / Artifact Result
```

后端不应该让 Agent 直接执行任意系统命令，而是通过受控工具：

```text
workspace.list_files
workspace.read_file
workspace.write_file
workspace.delete_file
workspace.run_command
workspace.create_artifact
```

每个工具都必须校验路径、配额、权限、超时和输出大小。

## 3. Workspace 生命周期

状态：

- `disabled`：用户未启用。
- `provisioning`：正在创建。
- `active`：可使用。
- `idle`：长时间未使用，等待归档或销毁。
- `archived`：已归档，保留元数据或压缩包。
- `destroying`：正在销毁。
- `destroyed`：已销毁。
- `suspended`：因安全或配额问题暂停。

生命周期流程：

```text
用户开启 workspace
  -> 后端创建 workspace 记录
  -> Sandbox Manager 创建目录或容器
  -> 状态 active
  -> Agent 工具调用更新 last_active_at
  -> 超过 idle_ttl 进入 idle
  -> 超过 destroy_ttl 自动销毁或归档
```

## 4. 目录规划

Linux 部署时推荐：

```text
/var/lib/freedinner/workspaces/{user_id}/
```

如果未来支持多租户：

```text
/var/lib/freedinner/workspaces/{tenant_id}/{user_id}/
```

目录内建议：

```text
workspace/
  files/        # 用户文件
  artifacts/    # 生成物
  tmp/          # 临时文件
  logs/         # 命令输出摘要或执行日志引用
```

后端必须做路径归一化，禁止通过 `../`、软链接或绝对路径逃逸 workspace。

## 5. 数据结构建议

新增表：

```text
user_workspaces
workspace_files
workspace_command_runs
workspace_events
workspace_quota_snapshots
```

`user_workspaces`：

- `id`
- `user_id`
- `status`
- `root_path`
- `sandbox_type`：`local_dir`、`docker`、`podman`、`nsjail`、`firecracker`
- `network_policy`：`disabled`、`allowlist`、`full`
- `network_allowlist`
- `max_disk_bytes`
- `max_file_count`
- `max_single_file_bytes`
- `max_command_seconds`
- `max_stdout_bytes`
- `max_stderr_bytes`
- `cpu_limit`
- `memory_limit_bytes`
- `last_active_at`
- `idle_after_seconds`
- `destroy_after_seconds`
- `created_at`
- `updated_at`

`workspace_files`：

- `id`
- `user_id`
- `workspace_id`
- `relative_path`
- `file_type`
- `size_bytes`
- `content_hash`
- `mime_type`
- `created_by`：`user`、`agent`、`tool`、`system`
- `status`
- `metadata`
- `created_at`
- `updated_at`

`workspace_command_runs`：

- `id`
- `user_id`
- `workspace_id`
- `conversation_id`
- `agent_turn_id`
- `tool_call_id`
- `command`
- `args`
- `working_dir`
- `network_policy`
- `exit_code`
- `status`
- `stdout_preview`
- `stderr_preview`
- `stdout_truncated`
- `stderr_truncated`
- `started_at`
- `finished_at`
- `duration_ms`
- `error_message`
- `metadata`

`workspace_events`：

- `id`
- `user_id`
- `workspace_id`
- `event_type`
- `actor_type`
- `actor_id`
- `file_id`
- `command_run_id`
- `metadata`
- `created_at`

`workspace_quota_snapshots`：

- `id`
- `user_id`
- `workspace_id`
- `used_disk_bytes`
- `file_count`
- `command_count`
- `active_process_count`
- `metadata`
- `created_at`

## 6. 资源隔离

MVP 可以从本地目录做起，但设计上要预留强隔离。

### local_dir

优点：

- 实现快。
- 本地开发方便。
- 适合早期演示。

限制：

- 不能强隔离 CPU、内存和网络。
- 必须严格做命令白名单和路径校验。
- 不适合直接跑不可信代码。

### container

推荐生产方向：

- Docker 或 Podman 每个用户一个短生命周期容器。
- workspace 作为 volume 挂载。
- 容器内普通用户运行命令。
- 配置 CPU、内存、进程数、网络模式。

### stronger sandbox

更强方案：

- nsjail / bubblewrap：轻量 Linux namespace 隔离。
- Firecracker：微虚拟机隔离，安全性更强，成本更高。

## 7. 网络策略

网络默认关闭。

策略：

- `disabled`：禁止联网。
- `allowlist`：只允许访问白名单域名或 IP。
- `full`：允许全部网络访问。

推荐默认：

```text
个人文件处理：disabled
代码依赖安装：allowlist
联网搜索或下载：通过专门 search/download tool，不直接给 shell 全网权限
```

即使开启网络，也要限制：

- 请求超时。
- 下载大小。
- 访问域名。
- 是否允许访问内网地址。

禁止默认访问：

- `localhost`
- 云厂商 metadata 地址
- 数据库内网地址
- 其他用户 workspace 路径

## 8. CLI 执行策略

`workspace.run_command` 必须受控。

执行前校验：

- workspace 是否 active。
- 命令是否在允许列表。
- 参数是否包含危险路径。
- 工作目录是否在 workspace 内。
- 当前用户是否超过配额。
- 当前 Agent 是否启用了 workspace tool。

执行时限制：

- 超时时间。
- stdout/stderr 最大字节数。
- MVP 阶段不经过 shell，直接 `exec` 白名单命令。
- 网络策略先记录在 workspace 和命令日志中，真正网络隔离放到容器 sandbox。
- 最大进程数、CPU 和内存限制放到 Step 15。

Step 15 后端代码增加统一 runner：

- `local_dir`：本地开发默认 runner，只做路径、白名单、超时和输出截断。
- `docker`：通过 `docker run --rm` 挂载当前用户 workspace，并设置 `--network none`、`--memory`、`--cpus`、`--pids-limit`、`--cap-drop ALL`、`no-new-privileges`。
- `podman`：参数与 Docker runner 对齐，方便 Linux 上用 rootless Podman。
- `nsjail`：通过 bind mount 将用户 workspace 挂载到 `/workspace`，设置独立网络 namespace、进程数、执行时间和地址空间限制。

本地开发不要求安装 Docker、Podman 或 nsjail。只有当用户 workspace 的 `sandbox_type` 配成对应值时，后端才会调用相应 runtime。

执行后：

- 保存 `workspace_command_runs`。
- 截断 stdout/stderr preview。
- 将生成文件同步到 `workspace_files`。
- 写入 `workspace_events`。

MVP 命令白名单可以从：

```text
ls
cat
pwd
mkdir
touch
python
python3
node
go version
go env
go test
go run
npm run
npm --version
npm test
```

MVP 中 `python`、`python3`、`node` 只允许运行 workspace 内脚本文件，不开放 `python -c`、`node -e` 这类内联执行。所有参数禁止绝对路径、`..`、换行和 NUL 字符。

开始。高风险命令默认不开放：

```text
sudo
chmod
chown
rm -rf
curl
wget
ssh
scp
docker
```

是否开放依赖安装应由用户显式授权。

## 9. Agent 与 Workspace 的关系

用户 Agent 配置可以增加：

```text
workspace_enabled
workspace_id
workspace_tool_policy
```

工具路由时：

1. 如果用户未启用 workspace，不加载 workspace 工具。
2. 如果 workspace 处于 `idle`，先唤醒或重新 provision。
3. 如果命令需要网络但策略禁止，Tool Router 直接拒绝或转为询问用户。
4. 如果写文件超过配额，返回结构化错误，Agent 不能声称成功。

Context Builder 可以注入简短 workspace policy：

```text
You may only read and write files inside the user's workspace.
You must not access absolute paths or parent directories.
Network is disabled unless the workspace policy explicitly allows it.
```

## 10. 清理与销毁

清理策略：

- `idle_after_seconds`：超过该时间无操作，标记 idle。
- `destroy_after_seconds`：超过该时间仍无操作，销毁 workspace。
- 用户可手动销毁。
- 管理员可因配额、安全事件暂停 workspace。

销毁前可选择：

- 直接删除。
- 压缩归档到对象存储。
- 仅保留 metadata 和命令日志。

默认建议：

```text
MVP：只支持手动销毁
生产：支持 idle 自动归档和 destroy 自动销毁
```

## 11. 审计与安全

必须审计：

- workspace 创建、启用、暂停、销毁。
- 文件创建、读取、更新、删除。
- 命令执行。
- 网络访问策略变化。
- 配额超限。
- sandbox 逃逸尝试。

审计日志只保存必要摘要，避免保存完整敏感文件内容。

## 12. 前端页面

设置页新增 Workspace 区域：

- 是否启用 workspace。
- 当前状态。
- 磁盘使用量。
- 文件数量。
- 网络策略。
- 最近活跃时间。
- 手动清理/销毁。

Agent 工作台右侧可展示：

- 本轮读取了哪些文件。
- 本轮创建了哪些文件。
- 运行了哪些命令。
- 命令输出摘要。
- 是否触发配额或权限限制。

## 13. MVP 落地范围

第一版建议：

1. 数据库增加 workspace 相关表。
2. 用户可以启用/禁用 workspace。
3. 后端创建本地目录。
4. 支持 list/read/write 文件。
5. 支持受限命令执行，默认网络关闭。
6. 所有操作写审计日志。

强制暂缓：

- Docker/Podman 隔离。
- 自动销毁。
- 网络白名单 enforcement。
- 复杂配额统计。
- 文件版本历史。

生产前必须补上强隔离，否则不要开放任意代码执行给真实用户。
