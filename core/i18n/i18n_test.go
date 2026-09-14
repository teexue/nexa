package i18n_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/i18n"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "zh-CN"},
		{"zh", "zh-CN"},
		{"zh-CN", "zh-CN"},
		{"zh_CN", "zh-CN"},
		{"en", "en"},
		{"en-US", "en"},
		{"fr", "zh-CN"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, i18n.Normalize(tt.in), tt.in)
	}
}

func TestResolveLocale(t *testing.T) {
	t.Setenv("NEXA_LOCALE", "")
	assert.Equal(t, "zh-CN", i18n.ResolveLocale("", ""))
	assert.Equal(t, "en", i18n.ResolveLocale("en", "zh-CN"))
	assert.Equal(t, "zh-CN", i18n.ResolveLocale("", "zh-CN"))

	t.Setenv("NEXA_LOCALE", "en")
	assert.Equal(t, "en", i18n.ResolveLocale("", "zh-CN"))
	assert.Equal(t, "zh-CN", i18n.ResolveLocale("zh-CN", "en"))
}

func TestParseAcceptLanguage(t *testing.T) {
	assert.Equal(t, "", i18n.ParseAcceptLanguage(""))
	assert.Equal(t, "en", i18n.ParseAcceptLanguage("en-US,en;q=0.9"))
	assert.Equal(t, "zh-CN", i18n.ParseAcceptLanguage("zh-CN,zh;q=0.9,en;q=0.8"))
}

func TestBundleT(t *testing.T) {
	zh, err := i18n.NewBundle("zh-CN")
	require.NoError(t, err)
	assert.Equal(t, "工具开始", zh.TLog("log.tool.start"))
	assert.Equal(t, "再见", zh.T("tui.chat.goodbye"))
	assert.Equal(t, "会话 abc 已保存", zh.T("tui.chat.session_saved", "id", "abc"))
	assert.Equal(t, "missing.key", zh.T("missing.key"))

	en, err := i18n.NewBundle("en")
	require.NoError(t, err)
	assert.Equal(t, "tool start", en.TLog("log.tool.start"))
	assert.Equal(t, "Goodbye", en.T("tui.chat.goodbye"))
}

func TestSlogHandler(t *testing.T) {
	zh, err := i18n.NewBundle("zh-CN")
	require.NoError(t, err)

	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(i18n.NewSlogHandler(inner, zh))
	logger.Info("log.tool.start", "tool", "echo")

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "工具开始", rec["msg"])
	assert.Equal(t, "log.tool.start", rec["msg_key"])
	assert.Equal(t, "echo", rec["tool"])
}

func TestContextBundle(t *testing.T) {
	en, err := i18n.NewBundle("en")
	require.NoError(t, err)
	ctx := i18n.WithLocale(context.Background(), en)
	assert.Equal(t, "Goodbye", i18n.TCtx(ctx, "tui.chat.goodbye"))
	assert.True(t, strings.Contains(i18n.FromContext(ctx).Locale(), "en"))
}
