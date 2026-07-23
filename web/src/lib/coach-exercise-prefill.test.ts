import { describe, expect, it } from 'vitest'
import {
  extractExerciseStarterCode,
  findExerciseStarterPrefill,
  shouldPrefillExerciseStarter,
} from './coach-exercise'

describe('exercise starter prefill', () => {
  const dtsQuestion = `找出下面 .d.ts 文件中的错误，并写出修正后的完整代码。

\`\`\`typescript
// config-helper.d.ts
declare module 'config-helper' {
  export function loadConfig(path: string): Config;
  export interface Config {
    host: string;
    port: number;
  }
  declare var defaultConfig: string; // 这里的声明位置有问题
}
\`\`\`

做完后直接把答案发给我。`

  it('extracts fenced code from bug_find question', () => {
    const code = extractExerciseStarterCode(dtsQuestion)
    expect(code).toContain("declare module 'config-helper'")
    expect(code).toContain('declare var defaultConfig')
    expect(code).not.toContain('```')
  })

  it('prefers code after --- separator', () => {
    const mixed = `讲错了，再看一遍。

---

补全 TODO：

\`\`\`ts
type X = {
  // TODO
}
\`\`\`
`
    expect(extractExerciseStarterCode(mixed)).toContain('type X')
  })

  it('shouldPrefill for multi-line fill questions', () => {
    const starter = extractExerciseStarterCode(dtsQuestion)
    expect(shouldPrefillExerciseStarter(dtsQuestion, { answerFormat: 'text' }, starter)).toBe(true)
  })

  it('shouldPrefill json format with code fence', () => {
    const q = `补全 compose：\n\n\`\`\`json\n{"services":{}}\n\`\`\``
    const starter = extractExerciseStarterCode(q)
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'json' }, starter)).toBe(true)
  })

  it('skips short illustrative snippets without fill cues', () => {
    const q = `channel 用法示意：\n\n\`\`\`go\nch := make(chan int)\n\`\`\`\n\n请用一句话说明。`
    const starter = extractExerciseStarterCode(q)
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, starter)).toBe(false)
  })

  it('extracts dockerfile without fences', () => {
    const q = `请对下列 Dockerfile 做多阶段构建优化。

FROM golang:1.20
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o myapp .
CMD ["./myapp"]

请写出优化后的完整 Dockerfile。
做完后直接把答案发给我。`
    const code = extractExerciseStarterCode(q)
    expect(code).toContain('FROM golang:1.20')
    expect(code).toContain('CMD ["./myapp"]')
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, code)).toBe(true)
  })

  it('extracts indented code blocks', () => {
    const q = `修正下面代码：

    declare module 'x' {
      export const a: number
    }

做完后直接把答案发给我。`
    expect(extractExerciseStarterCode(q)).toContain("declare module 'x'")
  })

  it('does not pull previous Dockerfile into a conceptual question', () => {
    const codeQ = `请对下列 Dockerfile 做多阶段构建优化。

\`\`\`dockerfile
FROM node:18 AS builder
WORKDIR /app
COPY . .
RUN npm ci
\`\`\`

请写出优化后的完整 Dockerfile。
做完后直接把答案发给我。`

    const conceptQ = `合并 RUN 不错。下面换一道：

---

在Docker多阶段构建的最终阶段，为了进一步减小镜像体积，常常将基础镜像从 python:3.11-slim 换成 python:3.11-alpine。请说明 Alpine 相比 Slim 的两个主要优势，并指出一个使用 Alpine 时可能需要注意的缺点。
做完后直接把答案发给我。`

    const messages = [
      { role: 'assistant', content: codeQ },
      { role: 'user', content: 'FROM node...' },
      { role: 'assistant', content: conceptQ },
    ]
    expect(findExerciseStarterPrefill(messages, { answerFormat: 'text' })).toBe('')
  })

  it('still prefills same question after feedback-only bubble', () => {
    const messages = [
      { role: 'assistant', content: dtsQuestion },
      { role: 'user', content: '乱写' },
      { role: 'assistant', content: '声明位置不对，再改一次。' },
    ]
    const got = findExerciseStarterPrefill(messages, { answerFormat: 'text' })
    expect(got).toContain("declare module 'config-helper'")
  })

  it('rejects conceptual alpine short_answer even if history has code', () => {
    const q = `请说明 Alpine 相比 Slim 的两个主要优势。做完后直接把答案发给我。`
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, 'FROM x\nRUN y\nCMD z')).toBe(
      false
    )
  })

  it('does not prefill when asking for program output', () => {
    const q = `写出以下代码的输出结果：

\`\`\`go
package main
import "fmt"
func main() {
    defer fmt.Print("1")
    defer fmt.Print("2")
    defer fmt.Print("3")
}
\`\`\`

做完后直接把答案发给我。`
    const starter = extractExerciseStarterCode(q)
    expect(starter).toContain('defer fmt.Print("1")')
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, starter)).toBe(false)
    expect(findExerciseStarterPrefill([{ role: 'assistant', content: q }], { answerFormat: 'text' })).toBe(
      ''
    )
  })

  it('prefills code_fill even when expected output is mentioned', () => {
    const q = `补全以下代码，使程序输出 "[2 4]"。要求：实现 \`filter\` 函数，它接受一个函数 \`f func(int) bool\` 和一个整数切片 \`nums\`。

\`\`\`go
package main
import "fmt"

// TODO: 定义 filter 函数
func filter(nums []int, f func(int) bool) []int {
    // 请补全代码

}

func main() {
    nums := []int{1,2,3,4,5}
    result := filter(nums, func(n int) bool { return n%2==0 })
    fmt.Println(result)
}
\`\`\`

做完后直接把答案发给我。`
    const starter = extractExerciseStarterCode(q)
    expect(starter).toContain('func filter')
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, starter)).toBe(true)
    expect(findExerciseStarterPrefill([{ role: 'assistant', content: q }], { answerFormat: 'text' })).toContain(
      'func filter'
    )
  })

  it('short_answer output question still does not prefill', () => {
    const q = `写出以下代码的输出结果：\n\n\`\`\`go\npackage main\nfunc main(){}\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(
      shouldPrefillExerciseStarter(q, { answerFormat: 'text', questionType: 'short_answer' }, s)
    ).toBe(false)
  })

  it('short_answer + 补全 still prefills via heuristic', () => {
    const q = `补全 filter：\n\n\`\`\`go\nfunc filter(nums []int) []int {\n  // TODO\n}\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(
      shouldPrefillExerciseStarter(q, { answerFormat: 'text', questionType: 'short_answer' }, s)
    ).toBe(true)
  })

  it('mislabeled code_fill output question does not prefill', () => {
    const q = `写出以下代码的输出结果：\n\n\`\`\`go\npackage main\nimport "fmt"\nfunc main() {\n    defer fmt.Print("1")\n}\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(
      shouldPrefillExerciseStarter(q, { answerFormat: 'text', questionType: 'code_fill' }, s)
    ).toBe(false)
  })

  it('trusts questionType code_fill even without fill keywords', () => {
    const q = `看这段骨架，让它能编译：\n\n\`\`\`go\npackage main\nfunc main() {\n}\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(
      shouldPrefillExerciseStarter(q, { answerFormat: 'text', questionType: 'code_fill' }, s)
    ).toBe(true)
  })

  it('执行结果 / 运行后的结果 should not prefill', () => {
    for (const q of [
      `写出这段代码的执行结果：\n\n\`\`\`go\npackage main\nfunc main(){ fmt.Print(1) }\n\`\`\``,
      `写出以下代码运行后的结果：\n\n\`\`\`go\npackage main\nfunc main(){}\n\`\`\``,
    ]) {
      const s = extractExerciseStarterCode(q)
      expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, s)).toBe(false)
    }
  })

  it('TODO inside code must not poison output question', () => {
    const q = `写出以下代码的输出结果：\n\n\`\`\`go\n// TODO: ignore\npackage main\nfunc main(){ fmt.Print(1) }\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, s)).toBe(false)
  })

  it('Dockerfile conceptual should not prefill', () => {
    const q = `下面 Dockerfile 中 FROM 指令的作用是什么？\n\n\`\`\`dockerfile\nFROM golang:1.20\nWORKDIR /app\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, s)).toBe(false)
  })

  it('改一下 / 填写 should prefill', () => {
    const q1 = `改一下下面这段代码，使它能够编译通过：\n\n\`\`\`ts\ntype X = { a: string\nb: number\n}\n\`\`\``
    const q2 = `请填写 filter 的实现：\n\n\`\`\`go\nfunc filter(nums []int, f func(int) bool) []int {\n  // ...\n}\n\`\`\``
    expect(
      shouldPrefillExerciseStarter(q1, { answerFormat: 'text' }, extractExerciseStarterCode(q1))
    ).toBe(true)
    expect(
      shouldPrefillExerciseStarter(q2, { answerFormat: 'text' }, extractExerciseStarterCode(q2))
    ).toBe(true)
  })

  it('请优化下面代码 should prefill', () => {
    const q = `请优化下面这段代码，减少不必要的拷贝：\n\n\`\`\`go\nfunc Copy(a []int) []int {\n  out := make([]int, len(a))\n  for i := range a { out[i] = a[i] }\n  return out\n}\n\`\`\``
    const s = extractExerciseStarterCode(q)
    expect(shouldPrefillExerciseStarter(q, { answerFormat: 'text' }, s)).toBe(true)
  })

  it('feedback with submit marker still finds prior starter', () => {
    const got = findExerciseStarterPrefill(
      [
        {
          role: 'assistant',
          content: `补全：\n\`\`\`go\npackage main\nfunc main(){}\n\`\`\`\n做完后直接把答案发给我。`,
        },
        { role: 'user', content: 'x' },
        { role: 'assistant', content: '还不对，请做完后直接把答案发给我。' },
      ],
      { answerFormat: 'text' }
    )
    expect(got).toContain('package main')
  })

  it('feedback containing 写下再交 still finds prior starter', () => {
    const got = findExerciseStarterPrefill(
      [
        {
          role: 'assistant',
          content: `补全：\n\`\`\`go\npackage main\nfunc main(){}\n\`\`\`\n做完后直接把答案发给我。`,
        },
        { role: 'user', content: 'x' },
        { role: 'assistant', content: '写下正确实现再交一次。做完后直接把答案发给我。' },
      ],
      { answerFormat: 'text' }
    )
    expect(got).toContain('package main')
  })

  it('short conceptual new question does not pull prior Dockerfile', () => {
    const codeQ = `请写出优化后的完整 Dockerfile。

\`\`\`dockerfile
FROM node:18 AS builder
WORKDIR /app
COPY . .
RUN npm ci
\`\`\`

做完后直接把答案发给我。`

    const conceptQ = `Alpine 相比 Slim 的两个主要优势是什么？做完后直接把答案发给我。`

    const messages = [
      { role: 'assistant', content: codeQ },
      { role: 'user', content: 'FROM node...' },
      { role: 'assistant', content: conceptQ },
    ]
    expect(findExerciseStarterPrefill(messages, { answerFormat: 'text' })).toBe('')
    expect(findExerciseStarterPrefill(messages, null)).toBe('')
  })

  it('dual fence prefers incomplete TODO block over complete example', () => {
    const dual = extractExerciseStarterCode(
      `补全 TODO：\n\`\`\`go\nfunc f() int {\n  // TODO\n}\n\`\`\`\n对照：\n\`\`\`go\nfunc f() int { return 42 }\n\`\`\``
    )
    expect(dual).toContain('TODO')
    expect(dual).not.toContain('return 42')
  })

  it('dual fence does not prefer complete example with ellipsis in comment', () => {
    const dual = extractExerciseStarterCode(
      `补全：\n\`\`\`go\nfunc f() int {\n}\n\`\`\`\n对照：\n\`\`\`go\nfunc f() int {\n  // use nums...\n  return 42\n}\n\`\`\``
    )
    expect(dual).not.toContain('return 42')
    expect(dual).toContain('func f() int')
  })
})
