package domain

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const skillMDTemplate = `---
name: regulus-coach-{{.Slug}}
description: >
  Regulus Academy 学习教练 · {{.DomainName}}：知识树导航、节点讲解、微练习出题与批改。
  用户提到{{.DomainName}}学习/练习时使用。
---

# Regulus Academy Coach — {{.DomainName}}

## 何时使用

- 用户要学习 **{{.DomainName}}** 相关知识
- 需要 **知识树导航**、**单节点讲解**、**微练习**、**作答批改**
- 在 IDE 里边看代码边学，或终端里碎片化练习

## 怎么做

1. 阅读 [protocol.md](./protocol.md) — 学习方式说明（只读这一份）
2. 读 ` + "`" + `domains/{{.Slug}}/tree.yaml` + "`" + ` 了解知识路径
3. 根据用户进度读对应 ` + "`" + `domains/{{.Slug}}/nodes/<节点key>.yaml` + "`" + ` 获取节点边界
4. 按节点推进：**讲解** → 用户回复「开始练习」→ **出一道题**（见 ` + "`" + `schemas/exercise.json` + "`" + `）→ **批改**（见 ` + "`" + `schemas/grade.json` + "`" + `）

## 与 Regulus Academy App 的关系

- 本 Skill 包是从 Regulus Academy App 导出的独立包，可在任意 Agent 中使用
- App 负责进度 SQLite、知识树可视化；Skill 可在 IDE / 终端 / 任意 Agent 入口使用，进度由用户口述或自行记录
`

const readmeMDTemplate = `# regulus-coach-{{.Slug}}

**{{.DomainName}}** 学习 Skill 包，由 [Regulus Academy](https://github.com/regulus-academy/regulus-academy) 导出。
{{if .Description}}
> {{.Description}}
{{end}}
## 使用方式

### 方式一：安装到 Agent 直接练习

将 ` + "`" + `regulus-coach-{{.Slug}}/` + "`" + ` 目录整体放入你的 Agent skills 目录（如 Cursor 的 ` + "`" + `.cursor/skills/` + "`" + `），重启 Agent 后即可说「我想练习{{.DomainName}}」开始学习。

### 方式二：贡献回 Regulus Academy 社区

1. 把 ` + "`" + `domains/{{.Slug}}/` + "`" + ` 复制到仓库的 ` + "`" + `regulus-coach/domains/{{.Slug}}/` + "`" + `
2. 检查 ` + "`" + `tree.yaml` + "`" + ` 顶部的 ` + "`" + `version: 1` + "`" + `，补充 ` + "`" + `description` + "`" + `
3. 提 PR，说明覆盖范围、目标用户、与现有公共库的差异

## 目录结构

` + "```" + `
regulus-coach-{{.Slug}}/
├── SKILL.md            # Agent 触发入口
├── README.md           # 本文件
├── protocol.md         # 教学协议
├── schemas/            # 出题 / 批改 JSON Schema
└── domains/{{.Slug}}/
    ├── tree.yaml
    └── nodes/
` + "```" + `
`

var skillMDTmpl = template.Must(template.New("skill").Parse(skillMDTemplate))
var readmeMDTmpl = template.Must(template.New("readme").Parse(readmeMDTemplate))

type skillZipData struct {
	Slug       string
	DomainName string
	Description string
}

// BuildSkillZip 将 ExportPackage 组装为 self-contained Skill zip 字节
func BuildSkillZip(pkg *ExportPackage) ([]byte, error) {
	if pkg == nil {
		return nil, fmt.Errorf("ExportPackage 为空")
	}
	slug := strings.TrimSpace(pkg.Slug)
	if slug == "" {
		return nil, fmt.Errorf("缺少 slug")
	}

	data := skillZipData{
		Slug:        slug,
		DomainName:  pkg.DomainName,
		Description: pkg.Description,
	}
	root := "regulus-coach-" + slug + "/"

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// SKILL.md
	if err := addTemplateFile(zw, root+"SKILL.md", skillMDTmpl, data); err != nil {
		return nil, err
	}

	// README.md
	if err := addTemplateFile(zw, root+"README.md", readmeMDTmpl, data); err != nil {
		return nil, err
	}

	// protocol.md（从 coach 根目录读）
	if proto, err := ReadCoachFile("protocol.md"); err == nil {
		if err := addBytes(zw, root+"protocol.md", proto); err != nil {
			return nil, err
		}
	}

	// schemas/*.json
	schemasDir := filepath.Join(CoachRoot(), "schemas")
	if entries, err := os.ReadDir(schemasDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			rel := filepath.Join("schemas", e.Name())
			content, err := ReadCoachFile(rel)
			if err != nil {
				continue
			}
			if err := addBytes(zw, root+"schemas/"+e.Name(), content); err != nil {
				return nil, err
			}
		}
	}

	// domains/<slug>/tree.yaml + nodes/*.yaml
	for path, content := range pkg.Files {
		if err := addBytes(zw, root+"domains/"+slug+"/"+path, []byte(content)); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 zip 失败: %w", err)
	}
	return buf.Bytes(), nil
}

func addTemplateFile(zw *zip.Writer, name string, tmpl *template.Template, data any) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("渲染 %s 失败: %w", name, err)
	}
	return addBytes(zw, name, buf.Bytes())
}

func addBytes(zw *zip.Writer, name string, content []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("创建 zip 条目 %s 失败: %w", name, err)
	}
	_, err = w.Write(content)
	return err
}
