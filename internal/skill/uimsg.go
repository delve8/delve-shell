package skill

// AddRefsLoadedMsg is sent when git branch/tag list for add-skill URL has been loaded.
type AddRefsLoadedMsg struct {
	Refs []string
}

// AddPathsLoadedMsg is sent when directory paths in repo have been loaded for add-skill.
type AddPathsLoadedMsg struct {
	Paths []string
}

// OpenAddSkillOverlayMsg opens the add-skill overlay.
type OpenAddSkillOverlayMsg struct {
	URL  string
	Ref  string
	Path string
}

// OpenUpdateSkillOverlayMsg opens the update-skill overlay.
type OpenUpdateSkillOverlayMsg struct {
	Name string
}

// ApplyDelSkillMsg applies `/config del-skill ...`.
type ApplyDelSkillMsg struct {
	Name string
}
