import { describe, expect, it } from "vitest"
import {
  firstTextLine,
  isTurnConclusion,
  latestTextLine,
  type FoldEntry,
} from "./activity-fold"

function entry(id: string, role: string, content: string): FoldEntry {
  return { id, role, content }
}

describe("text lines", () => {
  it("follows the newest thinking line and the first reply line", () => {
    expect(latestTextLine("先看结构\n再核对调用")).toBe("再核对调用")
    expect(latestTextLine("一行还没换行")).toBe("一行还没换行")
    expect(firstTextLine("# 结论\n后面的细节")).toBe("结论")
  })
})

describe("isTurnConclusion", () => {
  const turn = [
    entry("u1", "user", "hi"),
    entry("a1", "assistant", "先查一下"),
    entry("t1", "assistant", ""),
    entry("a2", "assistant", "最终答案"),
  ]

  it("keeps only the live tail open while the turn is still running", () => {
    expect(isTurnConclusion(turn, 1, true)).toBe(false)
    expect(isTurnConclusion(turn, 3, true)).toBe(true)
  })

  it("opens the last reply once the turn finishes, including earlier turns", () => {
    const mid = turn.slice(0, 3)
    expect(isTurnConclusion(mid, 1, true)).toBe(false)
    expect(isTurnConclusion(mid, 1, false)).toBe(true)
    const next = [
      ...turn.slice(0, 2),
      entry("u2", "user", "继续"),
      entry("a3", "assistant", ""),
    ]
    expect(isTurnConclusion(next, 1, true)).toBe(true)
  })
})
