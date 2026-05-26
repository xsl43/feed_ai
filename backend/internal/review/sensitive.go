package review

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// DFAFilter DFA 敏感词过滤器
type DFAFilter struct {
	mu   sync.RWMutex
	root *dfaNode
}

type dfaNode struct {
	children map[rune]*dfaNode
	isEnd    bool
}

// NewDFAFilter 创建 DFA 过滤器并从词库文件加载敏感词
func NewDFAFilter(wordFile string) (*DFAFilter, error) {
	f := &DFAFilter{root: &dfaNode{children: make(map[rune]*dfaNode)}}
	if err := f.Load(wordFile); err != nil {
		return nil, err
	}
	return f, nil
}

// Load 从文件加载敏感词（支持 # 开头的注释行）
func (f *DFAFilter) Load(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	f.root = &dfaNode{children: make(map[rune]*dfaNode)}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f.addWord(line)
	}
	return scanner.Err()
}

func (f *DFAFilter) addWord(word string) {
	word = strings.ToLower(word)
	node := f.root
	for _, ch := range word {
		if node.children[ch] == nil {
			node.children[ch] = &dfaNode{children: make(map[rune]*dfaNode)}
		}
		node = node.children[ch]
	}
	node.isEnd = true
}

// Check 检测文本是否包含敏感词，返回命中的敏感词列表
func (f *DFAFilter) Check(text string) []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	text = strings.ToLower(text)
	runes := []rune(text)
	var hits []string

	for i := 0; i < len(runes); i++ {
		node := f.root
		for j := i; j < len(runes); j++ {
			ch := runes[j]
			if node.children[ch] == nil {
				break
			}
			node = node.children[ch]
			if node.isEnd {
				hits = append(hits, string(runes[i:j+1]))
			}
		}
	}
	return hits
}

// Contains 检测文本是否包含任何敏感词
func (f *DFAFilter) Contains(text string) bool {
	return len(f.Check(text)) > 0
}
