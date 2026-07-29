package agent

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	reNonWord = regexp.MustCompile(`[^\p{L}\p{N}\s\-_./]+`)
	reSpaces  = regexp.MustCompile(`\s+`)
)

// 常见同义映射：归一化后的键 → 规范概念
var conceptSynonyms = map[string]string{
	"idempotency":   "幂等",
	"idempotent":    "幂等",
	"idempotence":   "幂等",
	"幂等性":           "幂等",
	"hash":          "哈希",
	"hashing":       "哈希",
	"hashtable":     "哈希表",
	"hash table":    "哈希表",
	"哈希算法":          "哈希",
	"cache":         "缓存",
	"caching":       "缓存",
	"mutex":         "互斥锁",
	"lock":          "锁",
	"goroutine":     "goroutine",
	"channel":       "channel",
	"rpc":           "RPC",
	"http":          "HTTP",
	"api":           "API",
	"rest":          "REST",
	"jwt":           "JWT",
	"oauth":         "OAuth",
	"tcp":           "TCP",
	"udp":           "UDP",
	"sql":           "SQL",
	"orm":           "ORM",
	"mq":            "消息队列",
	"message queue": "消息队列",
	"kafka":         "Kafka",
	"redis":         "Redis",
	"ttl":           "TTL",
	"cdn":           "CDN",
	"dns":           "DNS",
	"负载均衡":          "负载均衡",
	"load balancing": "负载均衡",
	"load balancer":  "负载均衡",
}

// NormalizeConcept 概念归一化：小写、去缀、同义合并
func NormalizeConcept(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = reNonWord.ReplaceAllString(s, " ")
	s = reSpaces.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	lower := strings.ToLower(s)
	if canon, ok := conceptSynonyms[lower]; ok {
		return canon
	}
	if canon, ok := conceptSynonyms[s]; ok {
		return canon
	}

	// 去常见中文后缀
	for _, suf := range []string{"的概念", "概念", "性", "机制", "原理", "技术"} {
		if strings.HasSuffix(s, suf) && len([]rune(s)) > len([]rune(suf))+1 {
			trimmed := strings.TrimSuffix(s, suf)
			trimmed = strings.TrimSpace(trimmed)
			if n := NormalizeConcept(trimmed); n != "" && n != trimmed {
				return n
			}
			if canon, ok := conceptSynonyms[strings.ToLower(trimmed)]; ok {
				return canon
			}
			s = trimmed
			break
		}
	}

	// 纯 ASCII 术语保持小写；含中文则保留原大小写（去空格边缘）
	if isASCIIWord(s) {
		return strings.ToLower(s)
	}
	return s
}

func isASCIIWord(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// NormalizeConceptList 批量归一并去重（保序）
func NormalizeConceptList(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		n := NormalizeConcept(r)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// ConceptsLooselyMatch 判断两个概念是否同一（归一后相等或互相包含）
func ConceptsLooselyMatch(a, b string) bool {
	na := NormalizeConcept(a)
	nb := NormalizeConcept(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}
	la := strings.ToLower(na)
	lb := strings.ToLower(nb)
	return strings.Contains(la, lb) || strings.Contains(lb, la)
}
