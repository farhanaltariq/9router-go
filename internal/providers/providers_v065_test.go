package providers

import (
	"testing"
)

func TestGemini38_Capabilities(t *testing.T) {
	caps := GetCapabilitiesForModel("", "gemini-3.8-flash-high")
	if !caps.Vision || !caps.Reasoning || !caps.Search || !caps.Tools {
		t.Errorf("gemini-3.8-flash-high should have Vision+Reasoning+Search+Tools, got %+v", caps)
	}
	caps2 := GetCapabilitiesForModel("antigravity", "gemini-3.8-flash")
	if !caps2.Vision || !caps2.AudioInput || !caps2.VideoInput {
		t.Errorf("gemini-3.8-flash should have Vision+AudioInput+VideoInput, got %+v", caps2)
	}
	caps3 := GetCapabilitiesForModel("", "gemini-3.8-flash-low")
	if !caps3.Reasoning {
		t.Errorf("gemini-3.8-flash-low should have Reasoning, got %+v", caps3)
	}
}

func TestCodeBuddyCN_Catalog(t *testing.T) {
	for _, m := range []string{"hy3", "hy3-x", "hy4-preview", "hy4-preview-x", "glm-5.3", "glm-5.3-flash", "kimi-k3-1"} {
		caps := GetCapabilitiesForModel("codebuddy-cn", m)
		if !caps.Reasoning {
			t.Errorf("codebuddy-cn %s should have Reasoning, got %+v", m, caps)
		}
	}
}

func TestModelTokenLimits_Gemini38(t *testing.T) {
	cw, maxOut := GetModelTokenLimits("gemini-3.8-flash-high")
	if cw != 1048576 || maxOut != 65536 {
		t.Errorf("expected 1048576/65536 for gemini-3.8, got %d/%d", cw, maxOut)
	}
}
