package ui

import "testing"

func TestApplyOverlayCloseFeatureResets(t *testing.T) {
	m := Model{
		AddRemote: AddRemoteOverlayState{
			Active:         true,
			Connecting:     true,
			Error:          "e",
			OfferOverwrite: true,
		},
		RemoteAuth: RemoteAuthOverlayState{
			Connecting: true,
			Step:       "password",
			Target:     "t",
			Error:      "ae",
			Username:   "u",
		},
		AddSkill: AddSkillOverlayState{
			Active: true,
			Error:  "se",
		},
		UpdateSkill: UpdateSkillOverlayState{
			Active: true,
			Error:  "ue",
		},
		ConfigLLM: ConfigLLMOverlayState{
			Active:   true,
			Checking: true,
			Error:    "ce",
		},
	}
	m2 := applyTestOverlayCloseFeatureResets(m)
	if m2.AddRemote.Active || m2.AddRemote.Connecting || m2.AddRemote.Error != "" || m2.AddRemote.OfferOverwrite {
		t.Fatalf("remote fields not cleared: %+v", m2)
	}
	if m2.RemoteAuth.Connecting || m2.RemoteAuth.Step != "" || m2.RemoteAuth.Target != "" || m2.RemoteAuth.Error != "" || m2.RemoteAuth.Username != "" {
		t.Fatalf("remote auth fields not cleared: %+v", m2)
	}
	if m2.AddSkill.Active || m2.AddSkill.Error != "" || m2.UpdateSkill.Active || m2.UpdateSkill.Error != "" {
		t.Fatalf("skill fields not cleared: %+v", m2)
	}
	if m2.ConfigLLM.Active || m2.ConfigLLM.Checking || m2.ConfigLLM.Error != "" {
		t.Fatalf("configllm fields not cleared: %+v", m2)
	}
}
