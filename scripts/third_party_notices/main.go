package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-licenses/v2/licenses"
)

type moduleJSON struct {
	Path    string      `json:"Path"`
	Version string      `json:"Version"`
	Dir     string      `json:"Dir"`
	Replace *moduleJSON `json:"Replace"`
}

type packageJSON struct {
	ImportPath string      `json:"ImportPath"`
	Module     *moduleJSON `json:"Module"`
}

type moduleRecord struct {
	Path           string
	Version        string
	Dir            string
	ReplacePath    string
	ReplaceVersion string
	ReplaceDir     string
}

type moduleNotice struct {
	Module         moduleRecord
	LicenseNames   []string
	LicensePath    string
	LicenseText    string
	LicenseSource  string
	NoticeEntries  []textAsset
	OverrideReason string
}

type textAsset struct {
	Label  string
	Text   string
	Source string
}

type licenseOverride struct {
	Names   []string
	File    string
	Source  string
	Reason  string
	Notices []textAsset
}

type groupedText struct {
	Names        []string
	Modules      []string
	Text         string
	Sources      []string
	Attributions []moduleAttribution
}

type moduleAttribution struct {
	Module string
	Text   string
}

var noticeFilePattern = regexp.MustCompile(`(?i)^notice([._-].*)?$`)

var licenseOverrides = map[string]licenseOverride{
	"github.com/cloudwego/eino-ext/components/model/openai": {
		Names:  []string{"Apache-2.0"},
		File:   "overrides/cloudwego-eino-ext-LICENSE-APACHE.txt",
		Source: "https://github.com/cloudwego/eino-ext/blob/main/LICENSE-APACHE",
		Reason: "module archive does not include a top-level license file; use the upstream repository Apache-2.0 license text",
	},
	"github.com/cloudwego/eino-ext/libs/acl/openai": {
		Names:  []string{"Apache-2.0"},
		File:   "overrides/cloudwego-eino-ext-LICENSE-APACHE.txt",
		Source: "https://github.com/cloudwego/eino-ext/blob/main/LICENSE-APACHE",
		Reason: "module archive does not include a top-level license file; use the upstream repository Apache-2.0 license text",
	},
	"github.com/mattn/go-localereader": {
		Names:  []string{"MIT"},
		File:   "overrides/mattn-go-localereader-LICENSE.txt",
		Source: "https://github.com/mattn/go-localereader/blob/master/LICENSE",
		Reason: "module archive does not include a top-level license file; use the upstream repository MIT license text",
	},
}

func main() {
	root := flag.String("root", "", "repository root")
	outFile := flag.String("out", "", "output markdown file")
	platformsFlag := flag.String("platforms", "", "space-separated GOOS/GOARCH values")
	goBin := flag.String("go", "go", "Go binary")
	generatedOn := flag.String("generated-on", "", "generation date")
	flag.Parse()

	if *root == "" {
		fatalf("missing --root")
	}
	if *outFile == "" {
		fatalf("missing --out")
	}
	if *platformsFlag == "" {
		fatalf("missing --platforms")
	}
	if *generatedOn == "" {
		fatalf("missing --generated-on")
	}

	rootDir, err := filepath.Abs(*root)
	if err != nil {
		fatalf("resolve root: %v", err)
	}
	outPath, err := filepath.Abs(*outFile)
	if err != nil {
		fatalf("resolve output path: %v", err)
	}

	mainModulePath, err := loadMainModulePath(rootDir, *goBin)
	if err != nil {
		fatalf("load main module path: %v", err)
	}

	platforms := strings.Fields(*platformsFlag)
	modules, err := collectModules(rootDir, *goBin, mainModulePath, platforms)
	if err != nil {
		fatalf("collect modules: %v", err)
	}

	classifier, err := licenses.NewClassifier()
	if err != nil {
		fatalf("create go-licenses classifier: %v", err)
	}

	results, err := inspectModules(classifier, modules)
	if err != nil {
		fatalf("inspect module licenses: %v", err)
	}

	content, err := renderMarkdown(results, *generatedOn, strings.Join(platforms, " "))
	if err != nil {
		fatalf("render markdown: %v", err)
	}

	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		fatalf("write %s: %v", outPath, err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func loadMainModulePath(rootDir, goBin string) (string, error) {
	cmd := exec.Command(goBin, "list", "-m", "-json")
	cmd.Dir = rootDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		return "", commandError(cmd, err)
	}

	var mod moduleJSON
	if err := json.Unmarshal(out, &mod); err != nil {
		return "", err
	}
	if mod.Path == "" {
		return "", fmt.Errorf("main module path is empty")
	}
	return mod.Path, nil
}

func collectModules(rootDir, goBin, mainModulePath string, platforms []string) ([]moduleRecord, error) {
	modulesByPath := map[string]moduleRecord{}
	for _, platform := range platforms {
		parts := strings.SplitN(platform, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid platform %q", platform)
		}

		cmd := exec.Command(goBin, "list", "-deps", "-json", "./...")
		cmd.Dir = rootDir
		cmd.Env = append(os.Environ(),
			"GOWORK=off",
			"GOOS="+parts[0],
			"GOARCH="+parts[1],
		)
		out, err := cmd.Output()
		if err != nil {
			return nil, commandError(cmd, err)
		}

		dec := json.NewDecoder(bytes.NewReader(out))
		for {
			var pkg packageJSON
			if err := dec.Decode(&pkg); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("decode go list output for %s: %w", platform, err)
			}
			if pkg.Module == nil {
				continue
			}
			if pkg.Module.Path == "" || pkg.Module.Path == mainModulePath {
				continue
			}
			rec := moduleFromJSON(pkg.Module)
			current, ok := modulesByPath[rec.Path]
			if !ok {
				modulesByPath[rec.Path] = rec
				continue
			}
			modulesByPath[rec.Path] = mergeModuleRecord(current, rec)
		}
	}

	modules := make([]moduleRecord, 0, len(modulesByPath))
	for _, mod := range modulesByPath {
		modules = append(modules, mod)
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Path < modules[j].Path
	})
	return modules, nil
}

func moduleFromJSON(mod *moduleJSON) moduleRecord {
	rec := moduleRecord{
		Path:    mod.Path,
		Version: mod.Version,
		Dir:     mod.Dir,
	}
	if mod.Replace != nil {
		rec.ReplacePath = mod.Replace.Path
		rec.ReplaceVersion = mod.Replace.Version
		rec.ReplaceDir = mod.Replace.Dir
	}
	return rec
}

func mergeModuleRecord(current, next moduleRecord) moduleRecord {
	if current.Version == "" {
		current.Version = next.Version
	}
	if current.Dir == "" {
		current.Dir = next.Dir
	}
	if current.ReplacePath == "" {
		current.ReplacePath = next.ReplacePath
	}
	if current.ReplaceVersion == "" {
		current.ReplaceVersion = next.ReplaceVersion
	}
	if current.ReplaceDir == "" {
		current.ReplaceDir = next.ReplaceDir
	}
	return current
}

func (m moduleRecord) sourceDir() string {
	if m.ReplaceDir != "" {
		return m.ReplaceDir
	}
	return m.Dir
}

func (m moduleRecord) displayVersion() string {
	version := m.Version
	if version == "" {
		version = "Unknown"
	}
	if m.ReplacePath == "" {
		return version
	}
	replacement := m.ReplacePath
	if m.ReplaceVersion != "" {
		replacement += " " + m.ReplaceVersion
	}
	return fmt.Sprintf("%s (replaced by %s)", version, replacement)
}

func inspectModules(classifier licenses.Classifier, modules []moduleRecord) ([]moduleNotice, error) {
	results := make([]moduleNotice, 0, len(modules))
	var missing []string

	for _, mod := range modules {
		result, err := inspectModule(classifier, mod)
		if err != nil {
			return nil, err
		}
		if len(result.LicenseNames) == 0 || strings.TrimSpace(result.LicenseText) == "" {
			missing = append(missing, mod.Path)
			continue
		}
		results = append(results, result)
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing full license text for: %s", strings.Join(missing, ", "))
	}

	return results, nil
}

func inspectModule(classifier licenses.Classifier, mod moduleRecord) (moduleNotice, error) {
	result := moduleNotice{Module: mod}
	sourceDir := mod.sourceDir()
	if sourceDir == "" {
		return result, fmt.Errorf("module %s has no source dir", mod.Path)
	}

	candidates, err := licenses.FindCandidates(sourceDir, sourceDir)
	if err != nil {
		return result, fmt.Errorf("find license candidates for %s: %w", mod.Path, err)
	}

	for _, candidate := range candidates {
		identified, err := classifier.Identify(candidate)
		if err != nil {
			continue
		}
		names := uniqueSortedLicenseNames(identified)
		if len(names) == 0 {
			continue
		}

		text, err := os.ReadFile(candidate)
		if err != nil {
			return result, fmt.Errorf("read %s for %s: %w", candidate, mod.Path, err)
		}

		result.LicenseNames = names
		result.LicensePath = candidate
		result.LicenseText = strings.TrimRight(string(text), "\n")
		break
	}

	notices, err := loadNoticeFiles(sourceDir, result.LicensePath)
	if err != nil {
		return result, fmt.Errorf("load notices for %s: %w", mod.Path, err)
	}
	result.NoticeEntries = notices

	if len(result.LicenseNames) > 0 && strings.TrimSpace(result.LicenseText) != "" {
		return result, nil
	}

	override, ok := licenseOverrides[mod.Path]
	if !ok {
		return result, nil
	}

	text, err := os.ReadFile(override.File)
	if err != nil {
		return result, fmt.Errorf("read override license for %s: %w", mod.Path, err)
	}

	result.LicenseNames = append([]string(nil), override.Names...)
	sort.Strings(result.LicenseNames)
	result.LicenseSource = override.Source
	result.LicenseText = strings.TrimRight(string(text), "\n")
	result.OverrideReason = override.Reason
	if len(result.NoticeEntries) == 0 && len(override.Notices) > 0 {
		result.NoticeEntries = append([]textAsset(nil), override.Notices...)
	}
	return result, nil
}

func uniqueSortedLicenseNames(items []licenses.License) []string {
	set := map[string]struct{}{}
	for _, item := range items {
		if item.Name == "" {
			continue
		}
		set[item.Name] = struct{}{}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func loadNoticeFiles(dir, licensePath string) ([]textAsset, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var notices []textAsset
	for _, entry := range entries {
		if entry.IsDir() || !noticeFilePattern.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if licensePath != "" && filepath.Clean(path) == filepath.Clean(licensePath) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			continue
		}
		notices = append(notices, textAsset{
			Label: entry.Name(),
			Text:  strings.TrimRight(string(data), "\n"),
		})
	}

	sort.Slice(notices, func(i, j int) bool {
		return notices[i].Label < notices[j].Label
	})
	return notices, nil
}

func renderMarkdown(results []moduleNotice, generatedOn, platforms string) ([]byte, error) {
	var buf bytes.Buffer

	counts := map[string]int{}
	hasOverride := false
	for _, result := range results {
		for _, name := range result.LicenseNames {
			counts[name]++
		}
		if result.OverrideReason != "" {
			hasOverride = true
		}
	}

	fmt.Fprintln(&buf, "# Third-Party Notices")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "`delve-shell` is licensed under Apache-2.0.")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "This file is generated by `scripts/update-third-party-notices.sh`.")
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, "Dependency data in this file was derived from the union of `go list -deps -json ./...` across these target platforms on %s: `%s`.\n", generatedOn, platforms)
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "License classification and candidate discovery use Google's `go-licenses` library.")
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, "The table below covers %d third-party Go modules currently reachable from the build targets.\n", len(results))
	fmt.Fprintln(&buf)

	if len(counts) > 0 {
		fmt.Fprintln(&buf, "License summary:")
		fmt.Fprintln(&buf)
		names := make([]string, 0, len(counts))
		for name := range counts {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(&buf, "- `%s`: %d modules\n", name, counts[name])
		}
		fmt.Fprintln(&buf)
	}

	if hasOverride {
		fmt.Fprintln(&buf, "Notes:")
		fmt.Fprintln(&buf)
		fmt.Fprintln(&buf, "- Some module archives do not include a top-level license file. Those entries use an explicit upstream license override, and the source is listed before the full text block.")
		fmt.Fprintln(&buf)
	}

	fmt.Fprintln(&buf, "## Go Modules")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "| Module | Version | License |")
	fmt.Fprintln(&buf, "|---|---:|---|")
	for _, result := range results {
		fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` |\n", result.Module.Path, result.Module.displayVersion(), strings.Join(result.LicenseNames, ", "))
	}
	fmt.Fprintln(&buf)

	licenseGroups := groupLicenseTexts(results)
	if len(licenseGroups) > 0 {
		fmt.Fprintln(&buf, "## Full License Texts")
		fmt.Fprintln(&buf)
		writeGroupedTexts(&buf, licenseGroups, "license")
	}

	noticeGroups := groupNoticeTexts(results)
	if len(noticeGroups) > 0 {
		fmt.Fprintln(&buf, "## Additional NOTICE Texts")
		fmt.Fprintln(&buf)
		writeGroupedTexts(&buf, noticeGroups, "notice")
	}

	return buf.Bytes(), nil
}

func groupLicenseTexts(results []moduleNotice) []groupedText {
	groupsByText := map[string]*groupedText{}
	for _, result := range results {
		text := normalizeLicenseText(result.LicenseText)
		if text == "" {
			continue
		}
		groupKey, groupText, attribution := groupableLicenseText(result.LicenseNames, text)
		group, ok := groupsByText[groupKey]
		if !ok {
			group = &groupedText{
				Names: append([]string(nil), result.LicenseNames...),
				Text:  groupText,
			}
			groupsByText[groupKey] = group
		}
		group.Modules = append(group.Modules, result.Module.Path)
		if result.LicenseSource != "" {
			group.Sources = append(group.Sources, result.LicenseSource)
		}
		if attribution != "" {
			group.Attributions = append(group.Attributions, moduleAttribution{
				Module: result.Module.Path,
				Text:   attribution,
			})
		}
	}
	return sortedGroupedTexts(groupsByText)
}

func groupNoticeTexts(results []moduleNotice) []groupedText {
	groupsByText := map[string]*groupedText{}
	for _, result := range results {
		for _, notice := range result.NoticeEntries {
			text := strings.TrimSpace(notice.Text)
			if text == "" {
				continue
			}
			group, ok := groupsByText[text]
			if !ok {
				group = &groupedText{
					Names: []string{"NOTICE"},
					Text:  notice.Text,
				}
				groupsByText[text] = group
			}
			group.Modules = append(group.Modules, result.Module.Path)
			if notice.Source != "" {
				group.Sources = append(group.Sources, notice.Source)
			}
		}
	}
	return sortedGroupedTexts(groupsByText)
}

func sortedGroupedTexts(groupsByText map[string]*groupedText) []groupedText {
	groups := make([]groupedText, 0, len(groupsByText))
	for _, group := range groupsByText {
		group.Modules = uniqueSortedStrings(group.Modules)
		group.Sources = uniqueSortedStrings(group.Sources)
		group.Names = uniqueSortedStrings(group.Names)
		group.Attributions = sortedAttributions(group.Attributions)
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		leftName := strings.Join(groups[i].Names, ", ")
		rightName := strings.Join(groups[j].Names, ", ")
		if leftName != rightName {
			return leftName < rightName
		}
		return groups[i].Modules[0] < groups[j].Modules[0]
	})
	return groups
}

func writeGroupedTexts(buf *bytes.Buffer, groups []groupedText, kind string) {
	titleCounts := map[string]int{}
	for _, group := range groups {
		titleCounts[strings.Join(group.Names, ", ")]++
	}
	titleIndexes := map[string]int{}

	for _, group := range groups {
		baseTitle := strings.Join(group.Names, ", ")
		titleIndexes[baseTitle]++
		title := baseTitle
		if titleCounts[baseTitle] > 1 {
			title = fmt.Sprintf("%s #%d", baseTitle, titleIndexes[baseTitle])
		}

		fmt.Fprintf(buf, "### %s\n\n", title)
		fmt.Fprintln(buf, "Used by:")
		fmt.Fprintln(buf)
		for _, module := range group.Modules {
			fmt.Fprintf(buf, "- `%s`\n", module)
		}
		fmt.Fprintln(buf)
		if len(group.Sources) > 0 {
			label := "Sources"
			if len(group.Sources) == 1 {
				label = "Source"
			}
			fmt.Fprintf(buf, "%s:\n\n", label)
			for _, source := range group.Sources {
				fmt.Fprintf(buf, "- `%s`\n", source)
			}
			fmt.Fprintln(buf)
		}
		if len(group.Attributions) > 0 {
			fmt.Fprintln(buf, "Project-specific attributions:")
			fmt.Fprintln(buf)
			for _, attribution := range group.Attributions {
				fmt.Fprintf(buf, "#### `%s`\n\n", attribution.Module)
				fmt.Fprintf(buf, "```text\n%s\n```\n\n", attribution.Text)
			}
		}
		fmt.Fprintf(buf, "```text\n%s\n```\n\n", group.Text)
	}

	if kind == "notice" && len(groups) == 0 {
		fmt.Fprintln(buf)
	}
}

func uniqueSortedStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedAttributions(values []moduleAttribution) []moduleAttribution {
	if len(values) == 0 {
		return nil
	}
	merged := map[string][]string{}
	for _, value := range values {
		text := strings.TrimSpace(value.Text)
		if value.Module == "" || text == "" {
			continue
		}
		merged[value.Module] = append(merged[value.Module], text)
	}
	modules := make([]string, 0, len(merged))
	for module := range merged {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	out := make([]moduleAttribution, 0, len(modules))
	for _, module := range modules {
		out = append(out, moduleAttribution{
			Module: module,
			Text:   strings.Join(uniqueSortedStrings(merged[module]), "\n\n"),
		})
	}
	return out
}

func normalizeLicenseText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func groupableLicenseText(names []string, text string) (key, groupedText, attribution string) {
	licenseName := strings.Join(uniqueSortedStrings(names), ", ")
	normalized := normalizeLicenseText(text)

	switch licenseName {
	case "Apache-2.0":
		if body, ok := canonicalApacheLicense(normalized); ok {
			return licenseName, body, ""
		}
		return licenseName, normalized, ""
	case "MIT":
		const anchor = "Permission is hereby granted, free of charge, to any person obtaining a copy"
		if body, preamble, ok := splitLicensePreamble(normalized, anchor); ok {
			return licenseName + "\n" + body, "MIT License\n\n" + body, cleanMITAttribution(preamble)
		}
	case "ISC":
		const anchor = "Permission to use, copy, modify, and/or distribute this software"
		if body, preamble, ok := splitLicensePreamble(normalized, anchor); ok {
			return licenseName + "\n" + body, "ISC License\n\n" + body, cleanISCAttribution(preamble)
		}
	}

	return licenseName + "\n" + normalized, normalized, ""
}

func splitLicensePreamble(text, anchor string) (body, preamble string, ok bool) {
	idx := strings.Index(text, anchor)
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(text[idx:]), strings.TrimSpace(text[:idx]), true
}

func cleanMITAttribution(preamble string) string {
	return cleanLicenseAttribution(
		preamble,
		[]string{
			"mit license",
			"the mit license (mit)",
		},
	)
}

func cleanISCAttribution(preamble string) string {
	return cleanLicenseAttribution(
		preamble,
		[]string{
			"isc license",
		},
	)
}

func cleanLicenseAttribution(preamble string, dropTitles []string) string {
	if preamble == "" {
		return ""
	}
	titleSet := map[string]struct{}{}
	for _, title := range dropTitles {
		titleSet[strings.ToLower(title)] = struct{}{}
	}
	lines := strings.Split(preamble, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if _, ok := titleSet[strings.ToLower(trimmed)]; ok {
			continue
		}
		kept = append(kept, trimmed)
	}
	return strings.Join(kept, "\n")
}

func canonicalApacheLicense(text string) (string, bool) {
	start := strings.Index(text, "Apache License")
	if start < 0 {
		return "", false
	}
	const endMarker = "END OF TERMS AND CONDITIONS"
	end := strings.Index(text[start:], endMarker)
	if end < 0 {
		return "", false
	}
	end += start + len(endMarker)
	return strings.TrimSpace(text[start:end]), true
}

func commandError(cmd *exec.Cmd, err error) error {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}
	stderr := strings.TrimSpace(string(exitErr.Stderr))
	if stderr == "" {
		return fmt.Errorf("%v", err)
	}
	return fmt.Errorf("%v: %s", err, stderr)
}
