package cloud

import (
	"github.com/regulus-academy/regulus-academy/internal/llm"
)

// ResolveLLM 返回用户应使用的 LLM 客户端及计费归属 platform|byok
func (s *Service) ResolveLLM(userID string) (llm.Provider, string, error) {
	if !s.cfg.Enabled() {
		return s.platformLLM, "platform", nil
	}
	cred, err := s.store.GetUserLLMCredentials(userID)
	if err != nil {
		return nil, "", err
	}
	if cred != nil {
		apiKey, err := decryptAPIKey(s.cfg.EncryptionKey, cred.APIKeyEncrypted)
		if err != nil {
			return nil, "", err
		}
		cfg := llm.OpenAIConfig{
			Provider: cred.Provider,
			APIKey:   apiKey,
			BaseURL:  cred.BaseURL,
			Model:    cred.Model,
		}
		if cfg.Provider == "" {
			cfg.Provider = "custom"
		}
		return llm.NewFromConfig(cfg), "byok", nil
	}
	return s.platformLLM, "platform", nil
}

// SaveUserLLMKey 保存用户 BYOK
func (s *Service) SaveUserLLMKey(userID, provider, apiKey, baseURL, model string) error {
	enc, err := encryptAPIKey(s.cfg.EncryptionKey, apiKey)
	if err != nil {
		return err
	}
	return s.store.SaveUserLLMCredentials(userID, provider, enc, baseURL, model)
}
