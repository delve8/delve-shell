package ui

import "testing"

func TestApplyOverlayCloseFeatureResets(t *testing.T) {
	m := Model{
		AddRemoteActive:         true,
		AddRemoteConnecting:     true,
		AddRemoteError:          "e",
		AddRemoteOfferOverwrite: true,
		RemoteAuthConnecting:    true,
		RemoteAuthStep:          "password",
		RemoteAuthTarget:        "t",
		RemoteAuthError:         "ae",
		RemoteAuthUsername:      "u",
		AddSkillActive:          true,
		AddSkillError:           "se",
		UpdateSkillActive:       true,
		UpdateSkillError:        "ue",
		ConfigLLMActive:         true,
		ConfigLLMChecking:       true,
		ConfigLLMError:          "ce",
	}
	m2 := ApplyOverlayCloseFeatureResets(m)
	if m2.AddRemoteActive || m2.AddRemoteConnecting || m2.AddRemoteError != "" || m2.AddRemoteOfferOverwrite {
		t.Fatalf("remote fields not cleared: %+v", m2)
	}
	if m2.RemoteAuthConnecting || m2.RemoteAuthStep != "" || m2.RemoteAuthTarget != "" || m2.RemoteAuthError != "" || m2.RemoteAuthUsername != "" {
		t.Fatalf("remote auth fields not cleared: %+v", m2)
	}
	if m2.AddSkillActive || m2.AddSkillError != "" || m2.UpdateSkillActive || m2.UpdateSkillError != "" {
		t.Fatalf("skill fields not cleared: %+v", m2)
	}
	if m2.ConfigLLMActive || m2.ConfigLLMChecking || m2.ConfigLLMError != "" {
		t.Fatalf("configllm fields not cleared: %+v", m2)
	}
}
