// Package paste flags checkpoint answers that look copy-pasted from
// Claude's own earlier output in the session transcript, rather than
// written in the engineer's own words.
//
// v1 note: uses word-set (Jaccard) similarity rather than the plan's
// embedding-similarity idea — no model/API call needed, and it's a strong
// enough signal for "this is basically the same paragraph".
package paste

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

const MinWords = 6
const SimilarityThreshold = 0.6

// RecentAssistantTexts reads a Claude Code transcript (JSONL) and returns
// the text content of assistant messages, most recent last.
func RecentAssistantTexts(transcriptPath string, limit int) []string {
	if transcriptPath == "" {
		return nil
	}
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	var texts []string
	for _, line := range lines {
		var entry struct {
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Message.Role != "assistant" {
			continue
		}
		var asString string
		if err := json.Unmarshal(entry.Message.Content, &asString); err == nil {
			texts = append(texts, asString)
			continue
		}
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(entry.Message.Content, &blocks); err == nil {
			for _, b := range blocks {
				if b.Type == "text" {
					texts = append(texts, b.Text)
				}
			}
		}
	}
	return texts
}

func wordSet(s string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, w := range strings.Fields(strings.ToLower(s)) {
		set[w] = struct{}{}
	}
	return set
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for w := range a {
		if _, ok := b[w]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// Warning returns a non-empty warning if answer looks like it was pasted
// from one of the transcript texts.
func Warning(answer string, transcriptTexts []string) string {
	if len(strings.Fields(answer)) < MinWords {
		return ""
	}
	answerSet := wordSet(answer)
	for _, text := range transcriptTexts {
		for _, chunk := range strings.Split(text, "\n\n") {
			chunk = strings.TrimSpace(chunk)
			if len(strings.Fields(chunk)) < MinWords {
				continue
			}
			if jaccard(answerSet, wordSet(chunk)) > SimilarityThreshold {
				return "This answer closely matches something Claude said earlier in this " +
					"session — write it in your own words, not a paste of the model's output."
			}
		}
	}
	return ""
}
