import { describe, expect, it } from "vitest"
import {
  acceptedThinkEffort,
  thinkChoices,
  type ThinkTarget,
} from "./think-effort"

function target(partial: Partial<ThinkTarget>): ThinkTarget {
  return {
    style: "openai",
    baseUrl: "",
    vendor: "",
    model: "",
    ...partial,
  }
}

describe("thinkChoices", () => {
  it("follows each vendor's documented levels", () => {
    expect(
      thinkChoices(
        target({
          baseUrl: "https://api.moonshot.cn/v1",
          vendor: "moonshot",
          model: "kimi-k3",
        })
      )
    ).toEqual(["low", "high", "max"])
    expect(
      thinkChoices(
        target({
          style: "anthropic",
          baseUrl: "https://api.moonshot.cn/anthropic",
          vendor: "moonshot",
          model: "kimi-k2.7-code",
        })
      )
    ).toEqual([])
    expect(
      thinkChoices(
        target({
          baseUrl: "https://api.deepseek.com",
          vendor: "deepseek",
          model: "deepseek-v4-flash",
        })
      )
    ).toEqual(["off", "low", "high", "max"])
    expect(
      thinkChoices(
        target({
          baseUrl: "https://open.bigmodel.cn/api/paas/v4",
          vendor: "zhipu",
          model: "glm-5.2",
        })
      )
    ).toEqual(["off", "high", "max"])
    expect(
      thinkChoices(
        target({
          style: "ollama",
          baseUrl: "https://ollama.com",
          vendor: "ollama_cloud",
          model: "gpt-oss:20b",
        })
      )
    ).toEqual(["low", "medium", "high"])
    expect(
      thinkChoices(
        target({
          baseUrl: "https://api.siliconflow.cn/v1",
          vendor: "siliconflow",
          model: "Qwen/Qwen2.5",
        })
      )
    ).toEqual([])
  })

  it("drops a level the current model does not accept", () => {
    const gptOss = target({
      style: "ollama",
      baseUrl: "https://ollama.com",
      vendor: "ollama_cloud",
      model: "gpt-oss:20b",
    })
    expect(acceptedThinkEffort(gptOss, "max")).toBe("")
    expect(acceptedThinkEffort(gptOss, "high")).toBe("high")
  })
})
