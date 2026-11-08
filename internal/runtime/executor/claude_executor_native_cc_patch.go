package executor

import (
	"fmt"
	"strings"

	"github.com/MoeForge/moego/ojson"
	log "github.com/sirupsen/logrus"
)

// LOCAL PATCH (2026-11-08): Claude Code 2.1.319 compaction requests carry no
// message-level cache_control at all (only system markers). Some upstream relays
// fail such large cold requests with 429/503 "Service Unavailable".
// For confirmed native Claude Code requests with zero message-level markers,
// add a breakpoint on the first user message and on the last eligible message.
func patchNativeMissingMessageCacheControlDoc(doc *ojson.Doc) bool {
	if doc == nil || doc.Len() == 0 {
		return false
	}
	messages := doc.Get("messages")
	if !messages.IsArray() {
		return false
	}
	msgs := messages.Array()
	if len(msgs) == 0 {
		return false
	}
	firstUser := -1
	for i, msg := range msgs {
		content := msg.Get("content")
		if content.IsArray() {
			for _, block := range content.Array() {
				if block.Get("cache_control").Exists() {
					return false
				}
			}
		}
		if firstUser < 0 && msg.Get("role").String() == "user" {
			firstUser = i
		}
	}
	changed := false
	if firstUser >= 0 && firstUser < len(msgs)-1 {
		msg := msgs[firstUser]
		content := msg.Get("content")
		if content.Type == ojson.String && strings.TrimSpace(content.String()) != "" {
			newContent := "[" + buildTextBlock(content.String(), &claudeCodeCacheControl) + "]"
			if err := doc.SetRawBytes(fmt.Sprintf("messages.%d.content", firstUser), []byte(newContent)); err == nil {
				changed = true
			}
		} else if idx := lastCacheableClaudeMessageContentBlockIndex(msg); idx >= 0 {
			if err := doc.Set(fmt.Sprintf("messages.%d.content.%d.cache_control", firstUser, idx), claudeCodeCacheControl); err == nil {
				changed = true
			}
		}
	}
	if injectMessagesCacheControlDoc(doc) {
		changed = true
	}
	if changed {
		log.Debugf("local patch: injected message cache_control for native Claude Code request without message markers")
	}
	return changed
}
