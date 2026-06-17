package domain

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const domainReadmeTemplate = `# {{.DomainName}} Domain 包

由 [Regulus Academy](https://github.com/regulus-academy/regulus-academy) 导出。
{{if .Description}}
> {{.Description}}
{{end}}
## 安装到 Coach Skill

1. 解压本 zip
2. 将 ` + "`" + `{{.Slug}}/` + "`" + ` 目录复制到你的 ` + "`" + `regulus-coach/domains/` + "`" + ` 下
3. 在 Agent 中说「我想学 {{.DomainName}}」

若尚未安装 Coach Skill，请从 Regulus Web 主页下载 ` + "`" + `regulus-coach.zip` + "`" + `。

## 贡献回社区

1. 把 ` + "`" + `domains/{{.Slug}}/` + "`" + ` 复制到仓库的 ` + "`" + `regulus-coach/domains/{{.Slug}}/` + "`" + `
2. 检查 ` + "`" + `tree.yaml` + "`" + ` 顶部的 ` + "`" + `version` + "`" + ` 与 ` + "`" + `description` + "`" + `
3. 提 PR，说明覆盖范围与目标用户
`

var domainReadmeTmpl = template.Must(template.New("domain-readme").Parse(domainReadmeTemplate))

type skillZipData struct {
	Slug        string
	DomainName  string
	Description string
}

// BuildDomainZip 将 ExportPackage 组装为 Domain 包 zip（仅 tree.yaml + nodes + README）。
func BuildDomainZip(pkg *ExportPackage) ([]byte, error) {
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
	root := slug + "/"

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	if err := addTemplateFile(zw, root+"README.md", domainReadmeTmpl, data); err != nil {
		return nil, err
	}
	for path, content := range pkg.Files {
		if err := addBytes(zw, root+path, []byte(content)); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 zip 失败: %w", err)
	}
	return buf.Bytes(), nil
}

// coachSkillInclude 判断 Coach Skill zip 是否应包含相对路径。
func coachSkillInclude(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "SKILL.md" || rel == "protocol.md" || rel == "triggers.yaml" {
		return true
	}
	if strings.HasPrefix(rel, "schemas/") && strings.HasSuffix(rel, ".json") {
		return true
	}
	if strings.HasPrefix(rel, "domains/") {
		return true
	}
	return false
}

// BuildCoachSkillZip 打包 regulus-coach 基础 Skill（protocol、schemas、内置 domains，不含 prompts/）。
func BuildCoachSkillZip() ([]byte, error) {
	root := CoachRoot()
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("regulus-coach 目录不可用: %w", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	prefix := "regulus-coach/"

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == "prompts" && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !coachSkillInclude(rel) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return addBytes(zw, prefix+rel, content)
	})
	if walkErr != nil {
		return nil, fmt.Errorf("遍历 regulus-coach 失败: %w", walkErr)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 zip 失败: %w", err)
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("Coach Skill zip 为空")
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
