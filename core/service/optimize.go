package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexakit/provider"
)

// userPromptOptimizer is the meta prompt for optimizing user inputs.
const userPromptOptimizer = `你是一个提示词优化专家。你的任务是理解用户的原始意图，并将其优化为更清晰、更具体、更结构化的提示词。
规则：
1. 保持用户的原始意图不变
2. 补充必要的上下文和约束条件
3. 使指令更明确、可执行
4. 如果提示词已经足够清晰，可以只做微调
5. 只返回优化后的提示词，不要添加任何解释或前缀`

// systemPromptOptimizer is the meta prompt for restructuring an agent's
// system prompt into a tagged, structured form.
const systemPromptOptimizer = `你是一个专业的提示词工程专家，擅长将用户提供的原始系统提示词重构成清晰、结构化、带 XML 标签的优化提示词。

<task>
用户会提供一段原始系统提示词（可能包含角色、规则、流程、风格等）。
将它优化为一个结构化的系统提示词，使用以下标签体系。请严格遵守输出格式和规则。
</task>

<required_tags>
- <role>：定义 AI 的角色，一句话。
- <core_rules>：包含所有必须遵守的硬性规则，每条规则用 <rule> 包裹，关键限制可添加 priority="highest" 属性。
- <workflow>：描述工作流程或步骤，用 <step> 包裹，可使用 trigger="触发条件" 属性。
- <tone>：定义回答的语气和风格。
</required_tags>

<optimization_principles>
1. 提取角色、规则、流程、风格等要素，分别放入对应标签；原始提示词缺少的要素不要编造对应标签。
2. 将关键限制条件（绝对不能做的事）放入 <core_rules> 并设为 priority="highest"。
3. 如果有条件逻辑（如果...就...），转化为 <step trigger="..."> 的形式。
4. 保持原始意图完全不变，不要添加或删减要求。
5. 保留原始提示词中的 {{...}} 占位符，不要修改、不要新增。
6. 对于不确定的内容，可推断但保留原意，不要过度发挥。
</optimization_principles>

<output_format>
只输出优化后的结构化提示词本身（<role>/<core_rules>/<workflow>/<tone> 标签块），
不要包裹 <system> 标签，不要附加 <user_input> 块，不要包含任何解释或前缀。
</output_format>`

// OptimizeOptions selects which model performs the optimization.
type OptimizeOptions struct {
	Agent    string // agent name; empty = first available agent
	Provider string // optional explicit provider, overrides the agent's
	Model    string // optional explicit model, overrides the agent's
}

// OptimizePrompt calls the LLM to optimize a user's prompt for intent clarity.
// It uses the agent/provider/model selected by opts.
func (s *Service) OptimizePrompt(ctx context.Context, prompt string, opts OptimizeOptions) (string, error) {
	if prompt == "" {
		return "", &ArgError{Field: "prompt", Message: "prompt is required"}
	}

	a, err := s.resolveAgentForOptimize(opts)
	if err != nil {
		return "", err
	}

	p, err := s.NewProvider(a)
	if err != nil {
		return "", &ServerError{Message: fmt.Sprintf("create provider: %v", err)}
	}

	optimized, err := streamOptimized(ctx, p, a.Model, userPromptOptimizer, prompt)
	if err != nil {
		return "", &ServerError{Message: fmt.Sprintf("stream: %v", err)}
	}
	return optimized, nil
}

// OptimizeSystemPromptOnce rewrites a system prompt into a tagged, structured
// form via a single optimizer LLM call. This is a manual, editor-triggered
// action — the result is returned for the user to review and save, never
// applied implicitly at run time.
func (s *Service) OptimizeSystemPromptOnce(ctx context.Context, systemPrompt string, opts OptimizeOptions) (string, error) {
	if strings.TrimSpace(systemPrompt) == "" {
		return "", &ArgError{Field: "prompt", Message: "prompt is required"}
	}

	a, err := s.resolveAgentForOptimize(opts)
	if err != nil {
		return "", err
	}

	p, err := s.NewProvider(a)
	if err != nil {
		return "", &ServerError{Message: fmt.Sprintf("create provider: %v", err)}
	}

	optimized, err := streamOptimized(ctx, p, a.Model, systemPromptOptimizer, systemPrompt)
	if err != nil {
		return "", &ServerError{Message: fmt.Sprintf("stream: %v", err)}
	}
	// Sanity check: the optimizer must honor the tagged output format.
	if !strings.Contains(optimized, "<role>") {
		return "", &ServerError{Message: "optimizer returned an invalid format"}
	}
	return optimized, nil
}

// OptimizeUserPrompt returns the optimized user prompt when the agent enables
// optimize.user_prompt; otherwise (or on failure) it returns prompt unchanged.
// Mock providers are skipped for the same reason as OptimizeSystemPrompt.
func OptimizeUserPrompt(ctx context.Context, a *agent.Agent, p provider.Provider, prompt string, log *slog.Logger) string {
	if a.Optimize == nil || !a.Optimize.UserPrompt || prompt == "" || isMockProvider(p) {
		return prompt
	}
	log = defaultLogger(log)

	optimized, err := streamOptimized(ctx, p, a.Model, userPromptOptimizer, prompt)
	if err != nil {
		log.Warn("log.optimize.user_prompt_failed", "agent", a.Name, "error", err)
		return prompt
	}
	return optimized
}

// streamOptimized runs a single optimizer LLM call and returns the trimmed
// text content of the response.
func streamOptimized(ctx context.Context, p provider.Provider, model, metaPrompt, content string) (string, error) {
	req := provider.Request{
		Model: model,
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: metaPrompt},
			{Role: provider.RoleUser, Content: content},
		},
		MaxTokens: 2048,
	}

	ch, err := p.Stream(ctx, req)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	for chunk := range ch {
		if chunk.TextDelta != "" {
			result.WriteString(chunk.TextDelta)
		}
		if chunk.Done {
			break
		}
	}

	optimized := strings.TrimSpace(result.String())
	if optimized == "" {
		return "", fmt.Errorf("LLM returned empty response")
	}
	return optimized, nil
}

// isMockProvider reports whether p is the scripted test double.
func isMockProvider(p provider.Provider) bool {
	_, ok := p.(*provider.MockProvider)
	return ok
}

func defaultLogger(log *slog.Logger) *slog.Logger {
	if log == nil {
		return slog.Default()
	}
	return log
}

// resolveAgentForOptimize builds the agent whose provider/model runs the
// optimization. An explicit provider (from the editor form) wins; otherwise
// the named agent is loaded, or the first available agent when no name given.
func (s *Service) resolveAgentForOptimize(opts OptimizeOptions) (*agent.Agent, error) {
	if opts.Provider != "" {
		return &agent.Agent{Name: "optimizer", Provider: opts.Provider, Model: opts.Model}, nil
	}

	if opts.Agent != "" {
		a, err := agent.LoadByName(s.AgentsDir, NormalizeAgentName(opts.Agent))
		if err != nil {
			return nil, fmt.Errorf("load agent: %w", err)
		}
		if opts.Model != "" {
			a.Model = opts.Model
		}
		return a, nil
	}

	names, err := agent.ListAvailable(s.AgentsDir)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	if len(names) == 0 {
		return nil, &ArgError{Field: "agent", Message: "no agents configured"}
	}
	a, err := agent.LoadByName(s.AgentsDir, names[0])
	if err != nil {
		return nil, fmt.Errorf("load agent: %w", err)
	}
	if opts.Model != "" {
		a.Model = opts.Model
	}
	return a, nil
}
