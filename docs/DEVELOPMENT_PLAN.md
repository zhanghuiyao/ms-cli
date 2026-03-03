# MS-CLI 开发计划

## 阶段目标

基于当前项目状态，制定以下开发计划。遵循"非必要不修改目录结构"原则，所有新功能将在现有包结构内实现。

---

## Phase 1: 核心引擎完善 (Week 1-2)

### 目标
完成 E2E 流程：CLI -> Agent -> Tool -> Result

### 任务清单

#### 1.1 配置系统接入 (Day 1-2)
**文件**: `app/bootstrap.go`, `configs/`

- [ ] 实现 `Config` 结构体，绑定 `configs/mscli.yaml`
- [ ] 创建配置加载器 `internal/config/loader.go`
- [ ] 在 Bootstrap 中加载配置并注入到 Engine
- [ ] 支持环境变量覆盖配置项

```go
// internal/config/config.go
package config

type Config struct {
    Model    ModelConfig    `yaml:"model"`
    Budget   BudgetConfig   `yaml:"budget"`
    UI       UIConfig       `yaml:"ui"`
    Permissions PermissionConfig `yaml:"permissions"`
    Context  ContextConfig  `yaml:"context"`
    Memory   MemoryConfig   `yaml:"memory"`
}
```

#### 1.2 真实执行器实现 (Day 3-4)
**文件**: `executor/runner.go`, `agent/loop/engine.go`

- [ ] 实现 `RealExecutor`，集成 tools 调用
- [ ] 重构 `Engine.Run()` 为异步事件流
- [ ] 支持任务队列和并发控制
- [ ] 实现基本错误处理和重试

```go
// executor/runner.go 扩展
func Run(task loop.Task, eventCh chan<- model.Event) error {
    // 1. 解析任务意图
    // 2. 选择工具链
    // 3. 顺序/并行执行
    // 4. 发射事件到 UI
}
```

#### 1.3 工具系统实现 (Day 5-7)
**文件**: `tools/fs/fs.go`, `tools/shell/shell.go`

| 工具 | 功能 | 事件类型 |
|------|------|----------|
| `fs.Read` | 读取文件内容 | `ToolRead` |
| `fs.Write` | 写入文件 | `ToolWrite` |
| `fs.Edit` | 编辑文件（diff） | `ToolEdit` |
| `fs.Glob` | 文件匹配 | `ToolGlob` |
| `fs.Grep` | 内容搜索 | `ToolGrep` |
| `shell.Exec` | 执行命令 | `CmdStarted/Output/Finished` |

- [ ] 实现 `fs.Tool` 的所有方法
- [ ] 实现 `shell.Tool` 的安全执行
- [ ] 添加命令超时控制
- [ ] 实现输出流式发射

#### 1.4 工具调用循环 (Day 8-10)
**文件**: `agent/loop/engine.go`

实现 ReAct 模式:
```
Thought -> Action (Tool) -> Observation -> ... -> Answer
```

- [ ] 创建 `AgentLoop` 结构体
- [ ] 实现步骤迭代器
- [ ] 集成工具注册表
- [ ] 实现停止条件判断

### 验收标准
- [ ] 可以在 TUI 中输入任务描述，看到工具调用序列
- [ ] `/roadmap status` 显示正确进度
- [ ] Demo 模式和 Real 模式都能正常运行

---

## Phase 2: LLM 集成 (Week 3-4)

### 目标
集成大语言模型，实现智能任务规划与执行

### 任务清单

#### 2.1 LLM Provider 接口 (Day 1-2)
**文件**: `integrations/llm/provider.go`

```go
package llm

type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

type CompletionRequest struct {
    Messages []Message
    Tools    []ToolDefinition
    MaxTokens int
}
```

- [ ] 定义 Provider 接口
- [ ] 实现 OpenAI Provider
- [ ] 实现配置驱动的 Provider 选择

#### 2.2 工具定义与 Function Calling (Day 3-5)
**文件**: `agent/loop/tools.go`

- [ ] 定义 `ToolDefinition` 结构
- [ ] 实现工具注册中心
- [ ] 将工具转换为 LLM Function Schema
- [ ] 解析 LLM 的工具调用请求

#### 2.3 智能 Agent 循环 (Day 6-10)
**文件**: `agent/loop/engine.go`

- [ ] 实现 `PlanningAgent` 类型
- [ ] 构建系统提示词模板
- [ ] 实现多轮对话管理
- [ ] 集成工具调用结果到上下文

```go
func (a *PlanningAgent) Run(task Task, eventCh chan<- model.Event) error {
    // 1. 构建初始消息
    // 2. 循环:
    //    - 调用 LLM
    //    - 如果是工具调用 -> 执行工具 -> 添加结果到上下文
    //    - 如果是最终答案 -> 返回结果
    // 3. 处理上下文超限
}
```

### 验收标准
- [ ] 可以自然语言提问，Agent 自动选择工具并执行
- [ ] 支持多轮工具调用序列
- [ ] 支持流式响应显示

---

## Phase 3: 内存与上下文 (Week 5-6)

### 目标
实现长期记忆和上下文管理

### 任务清单

#### 3.1 上下文管理器 (Day 1-3)
**文件**: `agent/context/manager.go`

- [ ] 实现 `Manager` 结构体
- [ ] 令牌计数（tiktoken 或等效实现）
- [ ] 实现滑动窗口历史管理
- [ ] 上下文压缩策略

#### 3.2 记忆系统 (Day 4-7)
**文件**: `agent/memory/`

- [ ] 实现 `Store` 接口的内存实现
- [ ] 添加持久化存储选项（JSON/BoltDB）
- [ ] 实现 `Retrieve` 语义搜索接口
- [ ] 添加记忆保留策略

```go
// agent/memory/store.go
func (s *FileStore) Put(key, value string) error
func (s *FileStore) Get(key string) (string, error)
func (s *FileStore) Search(query string, limit int) ([]RetrieveResult, error)
```

#### 3.3 预算控制 (Day 8-10)
**文件**: `agent/context/budget.go`

- [ ] 实现令牌使用跟踪
- [ ] 成本估算与限制
- [ ] 预警机制

### 验收标准
- [ ] 长对话不超限，自动压缩历史
- [ ] 跨会话记忆可检索
- [ ] 预算超限前发出警告

---

## Phase 4: 技能系统 (Week 7-8)

### 目标
完成技能仓库集成和调用

### 任务清单

#### 4.1 技能仓库 (Day 1-3)
**文件**: `integrations/skills/repo.go`

- [ ] 实现 `RepoSync` 接口
- [ ] Git 仓库克隆/更新
- [ ] 技能清单解析
- [ ] 本地缓存管理

#### 4.2 技能调用器 (Day 4-6)
**文件**: `integrations/skills/invoke.go`

- [ ] 实现 `Invoker` 接口
- [ ] YAML 工作流解析
- [ ] 步骤执行引擎
- [ ] 变量传递和上下文注入

#### 4.3 技能注册 (Day 7-8)
**文件**: `agent/loop/tools.go`

- [ ] 将技能转换为可用工具
- [ ] 动态工具发现

### 验收标准
- [ ] 配置中的技能可自动同步
- [ ] Agent 可以调用技能工作流
- [ ] 技能执行结果正确显示

---

## Phase 5: 增强与优化 (Week 9-10)

### 任务清单

#### 5.1 权限系统 (Day 1-3)
**文件**: `agent/loop/permission.go`

- [ ] 实现 `PermissionService`
- [ ] TUI 确认对话框
- [ ] 白名单/黑名单机制
- [ ] 危险操作二次确认

#### 5.2 日志与追踪 (Day 4-5)
**文件**: `trace/writer.go`

- [ ] 结构化日志输出
- [ ] 执行轨迹记录
- [ ] 性能指标收集

#### 5.3 UI 优化 (Day 6-8)
**文件**: `ui/`

- [ ] 消息搜索功能
- [ ] 会话保存/恢复
- [ ] 主题支持
- [ ] 响应式布局改进

#### 5.4 测试覆盖 (Day 9-10)

- [ ] 单元测试（覆盖率 > 60%）
- [ ] 集成测试
- [ ] E2E 测试（使用 demo 模式）

---

## 开发优先级总览

```
P0 (阻塞发布):
├── 配置系统接入
├── 真实执行器
├── 工具系统 (fs, shell)
├── Agent 工具调用循环
└── LLM Provider 集成

P1 (核心体验):
├── Function Calling
├── 上下文管理
├── 技能系统
└── 权限控制

P2 (增强功能):
├── 内存系统
├── 预算控制
└── 日志追踪

P3 (锦上添花):
├── UI 主题
├── 会话管理
└── 高级搜索
```

---

## 技术债务管理

| 债务项 | 处理时机 | 方案 |
|--------|----------|------|
| 缺少测试 | Phase 5 | 补充单元测试和集成测试 |
| 硬编码配置 | Phase 1 | 全部移入配置文件 |
| 错误处理不完整 | 持续 | 统一错误包装和处理 |
| 缺少文档 | 持续 | 为公共 API 添加注释 |

---

## 里程碑检查点

| 日期 | 里程碑 | 检查项 |
|------|--------|--------|
| Week 2 | Alpha | 基础工具链可用 |
| Week 4 | Beta | LLM 集成完成 |
| Week 6 | RC1 | 内存上下文可用 |
| Week 8 | RC2 | 技能系统完成 |
| Week 10 | GA | 生产就绪 |

