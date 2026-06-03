package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/config"
	"github.com/regulus-academy/regulus-academy/internal/llm"
)

func (h *Handler) llmConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.buildLLMConfigResponse())
}

func (h *Handler) buildLLMConfigResponse() map[string]any {
	state, err := config.LoadLLMProfiles()
	if err != nil {
		state = config.MigrateProfilesFromEnv()
	}
	activeName := ""
	for _, p := range state.Profiles {
		if p.ID == state.ActiveID {
			activeName = p.Name
			break
		}
	}
	display := activeName
	if display == "" {
		display = h.llmClient().Name()
	}
	return map[string]any{
		"provider":        display,
		"providerId":      llmProviderID(),
		"model":           h.llmClient().Model(),
		"configured":      h.llmClient().Configured(),
		"presets":         llm.ListPresetInfos(),
		"settings":        config.LLMSettingsViewFromEnv(),
		"profiles":        config.ProfilesViewFromState(state),
		"activeProfileId": state.ActiveID,
		"needsRestart":    false,
	}
}

func (h *Handler) updateLLMProfiles(w http.ResponseWriter, r *http.Request) {
	var payload config.LLMProfilesState
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	existing, err := config.LoadLLMProfiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if payload.GlobalAPIKey == "" {
		payload.GlobalAPIKey = existing.GlobalAPIKey
	}
	payload = config.MergeProfileAPIKeysFromExisting(existing, payload)
	if err := config.SaveLLMProfiles(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := config.ApplyActiveLLMProfile(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.reloadLLM(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.buildLLMConfigResponse())
}

func (h *Handler) activateLLMProfile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	id := strings.TrimSpace(body.ID)
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少模型 id")
		return
	}
	state, err := config.LoadLLMProfiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	found := false
	for _, p := range state.Profiles {
		if p.ID == id {
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, "模型不存在")
		return
	}
	state.ActiveID = id
	if err := config.SaveLLMProfiles(state); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := config.ApplyActiveLLMProfile(state); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.reloadLLM(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.buildLLMConfigResponse())
}

func (h *Handler) updateLLMConfig(w http.ResponseWriter, r *http.Request) {
	var payload config.LLMSettingsPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if err := config.ApplyLLMSettings(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.syncActiveProfileFromEnv(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.reloadLLM(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.buildLLMConfigResponse())
}

// syncActiveProfileFromEnv 将当前 .env 同步回活动模型条目（兼容旧版单表单保存）
func (h *Handler) syncActiveProfileFromEnv() error {
	state, err := config.LoadLLMProfiles()
	if err != nil {
		return err
	}
	cfg := llm.ConfigFromEnv()
	for i := range state.Profiles {
		if state.Profiles[i].ID == state.ActiveID {
			state.Profiles[i].Provider = cfg.Provider
			state.Profiles[i].BaseURL = cfg.BaseURL
			state.Profiles[i].Model = cfg.Model
			return config.SaveLLMProfiles(state)
		}
	}
	return nil
}

func (h *Handler) llmPingProbe(w http.ResponseWriter, r *http.Request) {
	var payload config.LLMSettingsPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	cfg, err := config.ResolvedLLMConfig(payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	client := llm.NewFromConfig(cfg)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": client.Name() + " 连接正常",
	})
}

func (h *Handler) reloadLLM() error {
	cfg := llm.ConfigFromEnv()
	client := llm.NewFromConfig(cfg)
	h.llm.Store(client)
	h.coach.SetLLM(client)
	h.sessions.SetLLM(client)
	return nil
}

func llmProviderID() string {
	return llm.ConfigFromEnv().Provider
}
