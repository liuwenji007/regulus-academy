package storage

import "encoding/json"

// SampleTree 返回 Phase 1 固定示例知识树（中文节点名，便于前端联调）
func SampleTree(domainID, domainName string) *KnowledgeTree {
	return &KnowledgeTree{
		DomainID:   domainID,
		DomainName: domainName,
		Layers: []TreeLayer{
			{
				Key:   "entry",
				Label: "入门",
				Time:  "~2 小时",
				Goal:  "能看懂并发代码，能创建简单的 goroutine",
				Nodes: []TreeNode{
					{Key: "goroutine_basics", Title: "goroutine 是什么"},
					{Key: "first_goroutine", Title: "启动第一个 goroutine"},
					{Key: "waitgroup", Title: "sync.WaitGroup 等待完成"},
				},
			},
			{
				Key:   "intermediate",
				Label: "熟悉",
				Time:  "~8 小时",
				Goal:  "能独立写生产级并发代码",
				Nodes: []TreeNode{
					{Key: "channel", Title: "channel 通信"},
					{Key: "select", Title: "select 多路复用"},
					{Key: "context", Title: "context 超时控制"},
					{Key: "mutex", Title: "sync.Mutex 互斥锁"},
				},
			},
			{
				Key:   "advanced",
				Label: "精通",
				Time:  "~20 小时",
				Goal:  "理解调度模型，能排查并发 bug",
				Nodes: []TreeNode{
					{Key: "gmp", Title: "GMP 调度模型"},
					{Key: "channel_internals", Title: "channel 底层数据结构"},
					{Key: "sync_pool", Title: "sync.Pool 对象复用"},
				},
			},
		},
	}
}

// SampleTreeJSON 序列化示例树
func SampleTreeJSON(domainID, domainName string) (string, error) {
	tree := SampleTree(domainID, domainName)
	b, err := json.Marshal(tree)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
