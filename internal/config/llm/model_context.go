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
// Keys are model id prefixes or full ids; value is max context in tokens (approximate; providers change limits).
// Longest matching prefix wins in [knownModelContextTokensByPrefix].
var knownModelContextTokens = map[string]int{
	// OpenAI / ChatGPT-class
	"gpt-5.4":       1000000,
	"gpt-5.3":       400000,
	"gpt-5.2":       250000,
	"gpt-5.1":       200000,
	"gpt-5":         128000,
	"gpt-4.1":       128000,
	"gpt-4o-mini":   128000,
	"gpt-4o":        128000,
	"gpt-4-turbo":   128000,
	"gpt-4":         8192,
	"gpt-3.5-turbo": 16385,
	"o4-mini":       128000,
	"o4":            200000,
	"o3-mini":       128000,
	"o3":            200000,
	"o1-pro":        200000,
	"o1-mini":       128000,
	"o1":            200000,

	// Anthropic (OpenAI-compatible gateways often expose these ids)
	"claude-opus-4":     200000,
	"claude-sonnet-4":   200000,
	"claude-3-5-sonnet": 200000,
	"claude-3-5-haiku":  200000,
	"claude-3-opus":     200000,
	"claude-3-sonnet":   200000,
	"claude-3-haiku":    200000,
	"claude-3":          200000,

	// Google Gemini
	"gemini-2.5-pro":   1000000,
	"gemini-2.5-flash": 1000000,
	"gemini-2.5":       1000000,
	"gemini-2.0":       1000000,
	"gemini-1.5-pro":   2000000,
	"gemini-1.5-flash": 1000000,
	"gemini-1.5":       1000000,
	"gemini-1.0":       32000,
	"gemini-pro":       32000,
	"gemini":           32000,

	// Alibaba Qwen / DashScope
	"qwen3-max":     256000,
	"qwen3-coder":   256000,
	"qwen3":         128000,
	"qwen2.5-max":   128000,
	"qwen2.5-coder": 128000,
	"qwen2.5":       128000,
	"qwen2":         131072,
	"qwen-long":     10000000,
	"qwen-max":      32000,
	"qwen-plus":     128000,
	"qwen-turbo":    8000,
	"qwen":          8192,

	// Moonshot / Kimi
	"kimi-k2":          128000,
	"moonshot-v1-128k": 128000,
	"moonshot-v1-32k":  32000,
	"moonshot-v1-8k":   8000,
	"moonshot-v1":      128000,
	"kimi":             128000,

	// Zhipu GLM
	"glm-4.6":     200000,
	"glm-4.5":     128000,
	"glm-4-air":   128000,
	"glm-4":       128000,
	"glm-3-turbo": 128000,
	"glm-3":       128000,
	"chatglm4":    128000,
	"chatglm3":    8192,
	"chatglm":     8192,

	// MiniMax
	"minimax-m2":      200000,
	"abab7-chat":      256000,
	"abab6.5":         245760,
	"abab6-chat":      8192,
	"abab5.5":         16384,
	"minimax-text-01": 400000,
	"minimax":         8192,

	// xAI Grok
	"grok-4":      256000,
	"grok-3-mini": 131072,
	"grok-3":      131072,
	"grok-2":      131072,
	"grok":        8192,

	// DeepSeek
	"deepseek-r1":       64000,
	"deepseek-v3.2":     128000,
	"deepseek-v3.1":     128000,
	"deepseek-v3":       64000,
	"deepseek-chat":     64000,
	"deepseek-coder":    64000,
	"deepseek-reasoner": 64000,

	// Meta Llama (common local / hosted ids)
	"llama-3.3": 128000,
	"llama-3.2": 128000,
	"llama-3.1": 128000,
	"llama-3":   8192,
	"llama":     8192,

	// Mistral / Mixtral
	"mistral-large":  128000,
	"mistral-medium": 32000,
	"mistral-small":  32000,
	"mixtral":        32000,
	"codestral":      32000,
	"mistral":        8192,

	// Microsoft / others
	"phi-4":          128000,
	"phi-3":          128000,
	"wizardlm":       8192,
	"yi-large":       32000,
	"yi-":            16000,
	"command-r-plus": 128000,
	"command-r":      128000,
	"starcoder2":     16384,
	"starcoder":      8192,

	// Cohere (OpenAI-compatible surfaces)
	"command-a": 256000,
	"command":   128000,
	"aya":       128000,
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
