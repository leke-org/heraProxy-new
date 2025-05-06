package server

import (
	"regexp"
)

type BlacklistBroadcastMsg struct {
	Ts        int64    `json:"ts"`
	Blacklist []string `json:"blacklist"`
}

// regexpDomain 函数用于从给定的文本中提取域名。
func regexpDomain(text string) string {
	domainRegex := regexp.MustCompile(`([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)

	matches := domainRegex.FindAllStringSubmatch(text, -1)
	// 遍历所有匹配结果
	for _, match := range matches {
		// 遍历每个匹配结果中的捕获组
		for _, v := range match {
			// 返回第一个匹配到的域名
			return v
		}
	}

	return ""
}
