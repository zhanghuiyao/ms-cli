# MS-CLI 最小 MVP 使用指南

## 快速开始

### 1. 配置 LLM API

创建 `configs/mscli.yaml`：

```yaml
model:
  provider: openai
  endpoint: https://api.openai.com/v1
  api_key: sk-your-api-key-here
  model: gpt-4

budget:
  max_tokens: 32768
  max_cost_usd: 10

context:
  max_tokens: 24000
  compaction_threshold: 0.85

memory:
  max_items: 200
  max_bytes: 2097152
  store_path: .cache/ms-cli/memory.json
```

或使用环境变量：
```bash
export MSCLI_MODEL_API_KEY=sk-your-api-key-here
export MSCLI_MODEL_ENDPOINT=https://api.openai.com/v1
```

### 2. 运行

```bash
# 构建
go build -o ms-cli ./app

# 运行 (有 API Key 时使用 LLM 模式)
./ms-cli

# 运行 Demo 模式 (无 LLM)
./ms-cli --demo
```

### 3. 使用

#### 自然语言任务
直接输入任务描述，Agent 会自动规划并执行：
```
> 列出所有 Go 文件
> 读取 README.md 内容
> 查找包含 "TODO" 的文件
> 执行 go version
```

#### 命令
```
/roadmap status          # 查看项目路线图
/weekly status           # 查看周报状态
```

#### 快捷键
- `Enter` - 发送消息
- `PgUp/PgDn` - 滚动聊天
- `Ctrl+C` - 退出

## 架构

```
User Input → TUI → Executor → SmartAgent/RuleAgent → Tools → Result
                    ↓
                 LLM Provider (OpenAI)
```

### 组件

| 组件 | 功能 |
|------|------|
| **SmartAgent** | LLM 驱动的任务规划和执行 |
| **RuleAgent** | 基于规则的简单模式匹配 |
| **Tools** | fs_read, fs_write, fs_edit, fs_glob, fs_grep, shell_exec |
| **Context** | 对话历史、预算管理、自动压缩 |
| **Memory** | 跨会话持久化存储 |

## 工具列表

| 工具 | 描述 | 示例 |
|------|------|------|
| `fs_read` | 读取文件 | `{"path": "README.md"}` |
| `fs_write` | 写入文件 | `{"path": "test.txt", "content": "hello"}` |
| `fs_edit` | 编辑文件 | `{"path": "file.go", "old_text": "foo", "new_text": "bar"}` |
| `fs_glob` | 文件匹配 | `{"pattern": "*.go"}` |
| `fs_grep` | 文本搜索 | `{"pattern": "TODO", "path": "."}` |
| `shell_exec` | 执行命令 | `{"command": "go version"}` |

## 开发状态

### Phase 1 ✅ 完成
- [x] 配置系统
- [x] 工具系统 (fs + shell)
- [x] Agent 循环 (Rule-based)
- [x] 上下文管理
- [x] 预算控制
- [x] 记忆存储

### Phase 2 ✅ 完成 (MVP)
- [x] LLM Provider 接口
- [x] SmartAgent (LLM 驱动)
- [x] Function Calling
- [x] 自动工具选择

### Phase 3 待实现
- [ ] 流式响应
- [ ] 技能系统
- [ ] 权限控制
- [ ] 高级搜索

## 示例对话

```
> 帮我分析这个项目的结构

[Agent 思考中...]
> 我来帮你分析项目结构。让我先查看文件列表。

[工具调用] fs_glob: *.go
→ 找到 15 个文件

[工具调用] fs_read: README.md
→ 读取 50 行

[完成] 这是一个 Go CLI 项目，主要包含：
- app/: 应用程序入口
- agent/: AI 代理核心
- tools/: 工具系统
- ui/: TUI 界面
...
```
