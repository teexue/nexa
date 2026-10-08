export interface ThinkTarget {
  style: string
  baseUrl: string
  vendor: string
  model: string
}

const full = ["off", "low", "medium", "high", "max"]

const familyRules: {
  id: string
  match: (host: string, name: string, blob: string) => boolean
}[] = [
  { id: "moonshot", match: (_h, _n, blob) => blob.includes("moonshot") },
  { id: "deepseek", match: (_h, _n, blob) => blob.includes("deepseek") },
  {
    id: "zhipu",
    match: (_h, name, blob) =>
      blob.includes("bigmodel") || blob.includes("z.ai") || name === "zhipu",
  },
  {
    id: "qwen",
    match: (_h, name, blob) =>
      blob.includes("dashscope") ||
      blob.includes("aliyuncs") ||
      name === "qwen",
  },
  { id: "groq", match: (_h, _n, blob) => blob.includes("groq") },
  { id: "openrouter", match: (_h, _n, blob) => blob.includes("openrouter") },
  { id: "siliconflow", match: (_h, _n, blob) => blob.includes("siliconflow") },
  {
    id: "anthropic",
    match: (host, name) =>
      host.includes("anthropic.com") || name === "anthropic",
  },
  {
    id: "openai",
    match: (host, name) => host.includes("openai.com") || name === "openai",
  },
  {
    id: "ollama",
    match: (host, _n, blob) =>
      blob.includes("ollama") || host.includes(":11434"),
  },
]

export function thinkChoices(target: ThinkTarget): string[] {
  switch (thinkFamily(target)) {
    case "ollama":
      return modelHas(target.model, "gpt-oss")
        ? ["low", "medium", "high"]
        : full
    case "moonshot":
      return moonshotChoices(target.model)
    case "deepseek":
      return ["off", "low", "high", "max"]
    case "zhipu":
      return zhipuChoices(target.model)
    case "qwen":
    case "openrouter":
    case "openai":
    case "anthropic":
      return full
    case "groq":
      return groqChoices(target.model)
    default:
      return []
  }
}

export function effortForProvider(
  providers: { name: string; api_style: string; base_url: string }[],
  provider: string,
  model: string,
  effort: string
): string {
  const info = providers.find((item) => item.name === provider)
  if (!info) return ""
  return acceptedThinkEffort(
    {
      style: info.api_style,
      baseUrl: info.base_url,
      vendor: info.name,
      model,
    },
    effort
  )
}

export function acceptedThinkEffort(
  target: ThinkTarget,
  effort: string
): string {
  if (!effort) return ""
  return thinkChoices(target).includes(effort) ? effort : ""
}

export function thinkEffortLabelKey(id: string): string {
  switch (id) {
    case "off":
      return "conversation.thinkOff"
    case "on":
      return "conversation.thinkOn"
    case "low":
      return "conversation.thinkLow"
    case "medium":
      return "conversation.thinkMedium"
    case "high":
      return "conversation.thinkHigh"
    case "max":
      return "conversation.thinkMax"
    default:
      return "conversation.thinkEffort"
  }
}

function thinkFamily(target: ThinkTarget): string {
  if (target.style === "ollama") return "ollama"
  const host = requestHost(target.baseUrl)
  const name = target.vendor.trim().toLowerCase()
  const blob = `${host} ${name}`
  return familyRules.find((rule) => rule.match(host, name, blob))?.id ?? ""
}

function requestHost(baseUrl: string): string {
  try {
    return new URL(baseUrl).host.toLowerCase()
  } catch {
    return baseUrl.trim().toLowerCase()
  }
}

function modelHas(model: string, part: string): boolean {
  return model.toLowerCase().includes(part)
}

function moonshotChoices(model: string): string[] {
  if (modelHas(model, "kimi-k3")) return ["low", "high", "max"]
  if (modelHas(model, "kimi-k2.7-code")) return []
  if (modelHas(model, "kimi-k2.6") || modelHas(model, "kimi-k2.5")) {
    return ["off", "on"]
  }
  return []
}

function zhipuChoices(model: string): string[] {
  if (modelHas(model, "glm-5.3")) return ["low", "high", "max"]
  if (modelHas(model, "glm-5.2") || modelHas(model, "glm-5.1")) {
    return ["off", "high", "max"]
  }
  if (modelHas(model, "glm")) return ["off", "on"]
  return []
}

function groqChoices(model: string): string[] {
  if (modelHas(model, "gpt-oss")) return ["low", "medium", "high"]
  if (modelHas(model, "qwen")) return ["off", "on"]
  return []
}
