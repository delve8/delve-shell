package skill

import (
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
)

// AddSkillOverlayState stores overlay-only state for add-skill flow.
type AddSkillOverlayState struct {
	Active         bool
	URLInput       textinput.Model
	RefInput       textinput.Model
	PathInput      textinput.Model
	NameInput      textinput.Model
	FieldIndex     int // 0=url, 1=ref, 2=path, 3=name
	Error          string
	RefsFullList   []string // all refs from remote (for filtering)
	RefCandidates  []string // refs filtered by Ref input prefix
	RefIndex       int      // selection in ref dropdown
	PathsFullList  []string // paths from git after ListPaths; Path dropdown only uses this (no static placeholder list)
	PathCandidates []string // path options filtered by Path input prefix
	PathIndex      int      // selection in path dropdown
}

// UpdateSkillOverlayState stores overlay-only state for update-skill flow.
type UpdateSkillOverlayState struct {
	Active        bool
	Name          string
	URL           string
	Path          string
	CurrentCommit string
	Refs          []string
	RefIndex      int
	LatestCommit  string
	Error         string
}

type skillOverlayState struct {
	AddSkill    AddSkillOverlayState
	UpdateSkill UpdateSkillOverlayState
}

var currentSkillOverlayState struct {
	mu    sync.RWMutex
	state skillOverlayState
}

func getSkillOverlayState() skillOverlayState {
	currentSkillOverlayState.mu.RLock()
	defer currentSkillOverlayState.mu.RUnlock()
	return currentSkillOverlayState.state
}

func setSkillOverlayState(state skillOverlayState) {
	currentSkillOverlayState.mu.Lock()
	currentSkillOverlayState.state = state
	currentSkillOverlayState.mu.Unlock()
}

func resetSkillOverlayState() {
	setSkillOverlayState(skillOverlayState{})
}
