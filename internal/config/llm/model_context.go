package configllm

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Cache TTL so we re-fetch occasionally (e.g. after provider update).
const modelContextCacheTTL = 30 * time.Minute

type modelContextCacheEntry struct {
	contextLength int
	expiresAt     time.Time
}

var (
	modelContextMu    sync.Mutex
	modelContextCache = make(map[string]modelContextCacheEntry)
)

// knownModelContextTokens is a fallback when the provider does not return context_length (e.g. OpenAI official).
// Keys are model id prefixes or full ids; value is max context in tokens.
var knownModelContextTokens = map[string]int{
	"gpt-4o":        128000,
	"gpt-4o-mini":   128000,
	"gpt-4-turbo":   128000,
	"gpt-4":         8192,
	"gpt-3.5-turbo": 16385,
	"o1":            200000,
	"o1-mini":       128000,
}

// FetchModelContextLength calls GET baseURL/v1/models, finds the model by id, and returns
// context_length if the provider includes it (many OpenAI-compatible APIs do). If the API
// does not return it, checks knownModelContextTokens by model name prefix. Returns 0 if
// not found. Results are cached per (baseURL, model) for modelContextCacheTTL.
func FetchModelContextLength(baseURL, apiKey, model string) int {
	if model == "" {
		return 0
	}
	key := strings.TrimRight(baseURL, "/") + "\t" + model
	modelContextMu.Lock()
	if e, ok := modelContextCache[key]; ok && time.Now().Before(e.expiresAt) {
		modelContextMu.Unlock()
		return e.contextLength
	}
	modelContextMu.Unlock()

	n := fetchModelContextLengthNoCache(baseURL, apiKey, model)
	if n == 0 {
		n = knownModelContextTokensByPrefix(strings.TrimSpace(model))
	}
	modelContextMu.Lock()
	modelContextCache[key] = modelContextCacheEntry{contextLength: n, expiresAt: time.Now().Add(modelContextCacheTTL)}
	modelContextMu.Unlock()
	return n
}

func knownModelContextTokensByPrefix(model string) int {
	if n, ok := knownModelContextTokens[model]; ok {
		return n
	}
	var best string
	for k := range knownModelContextTokens {
		if strings.HasPrefix(model, k) && len(k) > len(best) {
			best = k
		}
	}
	if best != "" {
		return knownModelContextTokens[best]
	}
	return 0
}

func fetchModelContextLengthNoCache(baseURL, apiKey, model string) int {
	url := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var out struct {
		Data []struct {
			ID             string `json:"id"`
			ContextLength  int    `json:"context_length"`
			ContextLength2 int    `json:"context_length_limit"` // some providers use this
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0
	}
	model = strings.TrimSpace(model)
	for _, m := range out.Data {
		if m.ID == model {
			if m.ContextLength > 0 {
				return m.ContextLength
			}
			if m.ContextLength2 > 0 {
				return m.ContextLength2
			}
			return 0
		}
	}
	return 0
}
