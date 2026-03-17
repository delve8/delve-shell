package git

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

// FetchRepoTree fetches the repo at url (ref optional, e.g. "main" or "v1.0") and writes
// the root tree (or subpath only if subpath is set) to destDir as plain files. No .git directory is created.
// subpath is a slash-separated path within the repo (e.g. "skills/foo"); empty means repo root.
// Returns the commit hash of the fetched ref.
func FetchRepoTree(ctx context.Context, url, ref, destDir, subpath string) (commitID string, err error) {
	auth := authFromURL(url)
	var refName plumbing.ReferenceName
	if ref != "" {
		refName, err = resolveRef(ctx, url, ref, auth)
		if err != nil {
			return "", err
		}
	}
	opts := &git.CloneOptions{
		URL:  url,
		Depth: 1,
		Auth:  auth,
	}
	if refName != "" {
		opts.ReferenceName = refName
		opts.SingleBranch = true
	}
	repo, err := git.CloneContext(ctx, memory.NewStorage(), nil, opts)
	if err != nil {
		return "", err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	commitID = head.Hash().String()
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", err
	}
	tree, err := repo.TreeObject(commit.TreeHash)
	if err != nil {
		return "", err
	}
	subpath = trimSlashPath(subpath)
	if subpath != "" {
		tree, err = treeAtPath(repo, tree, subpath)
		if err != nil {
			return "", err
		}
	}
	if err := writeTreeToDir(repo, tree, destDir, ""); err != nil {
		return "", err
	}
	return commitID, nil
}

func trimSlashPath(p string) string {
	return strings.Trim(strings.TrimSpace(p), "/")
}

// treeAtPath returns the tree at the given slash-separated path from root (e.g. "skills/foo").
func treeAtPath(repo *git.Repository, root *object.Tree, path string) (*object.Tree, error) {
	parts := strings.Split(path, "/")
	current := root
	for _, name := range parts {
		if name == "" {
			continue
		}
		entry, err := current.FindEntry(name)
		if err != nil {
			return nil, err
		}
		if entry.Mode != filemode.Dir {
			return nil, errors.New("not a directory: " + name)
		}
		current, err = repo.TreeObject(entry.Hash)
		if err != nil {
			return nil, err
		}
	}
	return current, nil
}

func resolveRef(ctx context.Context, repoURL, ref string, auth transport.AuthMethod) (plumbing.ReferenceName, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	refs, err := remote.ListContext(ctx, &git.ListOptions{Auth: auth})
	if err != nil {
		return "", err
	}
	for _, name := range []string{"refs/heads/" + ref, "refs/tags/" + ref} {
		want := plumbing.ReferenceName(name)
		for _, re := range refs {
			if re.Name() == want {
				return want, nil
			}
		}
	}
	return "", errRefNotFound
}

var errRefNotFound = errors.New("ref not found on remote")

// ListPaths returns directory paths in the repo at url/ref for the Path dropdown: root-level dirs plus "skills/xxx" if skills/ exists.
// ref can be empty (uses "main" or "master"). Uses authFromURL for private repos.
func ListPaths(ctx context.Context, repoURL, ref string) (paths []string, err error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil, nil
	}
	auth := authFromURL(repoURL)
	resolveRefName := ref
	if resolveRefName == "" {
		for _, try := range []string{"main", "master"} {
			if rn, e := resolveRef(ctx, repoURL, try, auth); e == nil && rn != "" {
				resolveRefName = try
				break
			}
		}
		if resolveRefName == "" {
			return nil, nil
		}
	}
	refName, err := resolveRef(ctx, repoURL, resolveRefName, auth)
	if err != nil {
		return nil, err
	}
	opts := &git.CloneOptions{URL: repoURL, Depth: 1, Auth: auth, ReferenceName: refName, SingleBranch: true}
	repo, err := git.CloneContext(ctx, memory.NewStorage(), nil, opts)
	if err != nil {
		return nil, err
	}
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := repo.TreeObject(commit.TreeHash)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			paths = append(paths, s)
		}
	}
	add(".")
	for _, e := range tree.Entries {
		if e.Mode != filemode.Dir {
			continue
		}
		add(e.Name)
		if e.Name == "skills" {
			sub, _ := repo.TreeObject(e.Hash)
			if sub != nil {
				for _, e2 := range sub.Entries {
					if e2.Mode == filemode.Dir {
						add("skills/" + e2.Name)
					}
				}
			}
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// ListRefs returns short names of branches and tags for the given repo URL (e.g. "main", "v1.0").
// Uses authFromURL for private repos. Returns nil slice on error or no refs.
func ListRefs(ctx context.Context, repoURL string) (refs []string) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil
	}
	auth := authFromURL(repoURL)
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	list, err := remote.ListContext(ctx, &git.ListOptions{Auth: auth})
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	for _, re := range list {
		name := re.Name()
		if !name.IsBranch() && !name.IsTag() {
			continue
		}
		short := name.Short()
		if short != "" && !seen[short] {
			seen[short] = true
			refs = append(refs, short)
		}
	}
	// stable order: sort by name
	if len(refs) > 1 {
		// simple sort
		for i := 0; i < len(refs); i++ {
			for j := i + 1; j < len(refs); j++ {
				if refs[j] < refs[i] {
					refs[i], refs[j] = refs[j], refs[i]
				}
			}
		}
	}
	return refs
}

func writeTreeToDir(repo *git.Repository, tree *object.Tree, destDir, subPath string) error {
	for _, entry := range tree.Entries {
		destPath := filepath.Join(destDir, subPath, entry.Name)
		switch entry.Mode {
		case filemode.Dir:
			subTree, err := repo.TreeObject(entry.Hash)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			if err := writeTreeToDir(repo, subTree, destDir, filepath.Join(subPath, entry.Name)); err != nil {
				return err
			}
		case filemode.Symlink:
			// skip or resolve; for simplicity skip to avoid security/portability issues
			continue
		default:
			// regular file or executable
			blob, err := repo.BlobObject(entry.Hash)
			if err != nil {
				return err
			}
			r, err := blob.Reader()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				r.Close()
				return err
			}
			f, err := os.Create(destPath)
			if err != nil {
				r.Close()
				return err
			}
			_, err = io.Copy(f, r)
			r.Close()
			f.Close()
			if err != nil {
				return err
			}
			if entry.Mode == filemode.Executable {
				_ = os.Chmod(destPath, 0755)
			}
		}
	}
	return nil
}
