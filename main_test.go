package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBranch(t *testing.T) {
	tests := []struct {
		name     string
		head     string
		expected string
	}{
		{
			name:     "normal branch",
			head:     "ref: refs/heads/main\n",
			expected: "main",
		},
		{
			name:     "feature branch",
			head:     "ref: refs/heads/feature/foo-bar\n",
			expected: "feature/foo-bar",
		},
		{
			name:     "detached HEAD",
			head:     "abc123def456789\n",
			expected: "@abc123de",
		},
		{
			name:     "short hash",
			head:     "abc1234\n",
			expected: "",
		},
		{
			name:     "empty",
			head:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.head != "" {
				if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte(tt.head), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := branch(dir)
			if got != tt.expected {
				t.Errorf("branch() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFindGitDir(t *testing.T) {
	// Create a nested directory structure with .git
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "src", "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	// Should find .git from nested directory
	got := findGitDir(nested)
	if got != gitDir {
		t.Errorf("findGitDir(%q) = %q, want %q", nested, got, gitDir)
	}

	// Should find .git from root
	got = findGitDir(root)
	if got != gitDir {
		t.Errorf("findGitDir(%q) = %q, want %q", root, got, gitDir)
	}

	// Should return empty for non-git directory
	other := t.TempDir()
	got = findGitDir(other)
	if got != "" {
		t.Errorf("findGitDir(%q) = %q, want empty", other, got)
	}
}

func TestParseGitStatus(t *testing.T) {
	// git status --porcelain format: XY filename
	//   X = staged status (index)
	//   Y = unstaged status (working tree)
	//
	// Status codes:
	//   M = modified, A = added, D = deleted, R = renamed, C = copied
	//   ? = untracked (only valid as ??)
	//   ' ' (space) = unmodified
	tests := []struct {
		name      string
		output    string
		staged    bool
		unstaged  bool
		untracked bool
	}{
		{
			name:   "empty",
			output: "",
		},
		// X column (staged/index changes)
		{
			name:   "staged modified",
			output: "M  file.go\n",
			staged: true,
		},
		{
			name:   "staged added",
			output: "A  file.go\n",
			staged: true,
		},
		{
			name:   "staged deleted",
			output: "D  file.go\n",
			staged: true,
		},
		{
			name:   "staged renamed",
			output: "R  old.go -> new.go\n",
			staged: true,
		},
		{
			name:   "staged copied",
			output: "C  file.go\n",
			staged: true,
		},
		// Y column (unstaged/working tree changes)
		{
			name:     "unstaged modified",
			output:   " M file.go\n",
			unstaged: true,
		},
		{
			name:     "unstaged deleted",
			output:   " D file.go\n",
			unstaged: true,
		},
		// Untracked files
		{
			name:      "untracked",
			output:    "?? file.go\n",
			untracked: true,
		},
		// Combined statuses
		{
			name:     "staged and unstaged",
			output:   "MM file.go\n", // modified in index, then modified again in working tree
			staged:   true,
			unstaged: true,
		},
		{
			name:      "multiple files",
			output:    "M  staged.go\n M unstaged.go\n?? new.go\n",
			staged:    true,
			unstaged:  true,
			untracked: true,
		},
		{
			name:      "only untracked",
			output:    "?? one.go\n?? two.go\n",
			untracked: true,
		},
		// Type changes
		{
			name:   "staged type change",
			output: "T  file\n",
			staged: true,
		},
		{
			name:     "unstaged type change",
			output:   " T file\n",
			unstaged: true,
		},
		// Conflicts
		{
			name:     "conflict UU",
			output:   "UU file.go\n",
			staged:   true,
			unstaged: true,
		},
		{
			name:     "conflict DD",
			output:   "DD file.go\n",
			staged:   true,
			unstaged: true,
		},
		{
			name:     "conflict AA",
			output:   "AA file.go\n",
			staged:   true,
			unstaged: true,
		},
		{
			name:     "conflict AU",
			output:   "AU file.go\n",
			staged:   true,
			unstaged: true,
		},
		{
			name:     "conflict UA",
			output:   "UA file.go\n",
			staged:   true,
			unstaged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staged, unstaged, untracked := parseGitStatus(tt.output)
			if staged != tt.staged {
				t.Errorf("staged = %v, want %v", staged, tt.staged)
			}
			if unstaged != tt.unstaged {
				t.Errorf("unstaged = %v, want %v", unstaged, tt.unstaged)
			}
			if untracked != tt.untracked {
				t.Errorf("untracked = %v, want %v", untracked, tt.untracked)
			}
		})
	}
}

func TestParseInputFixture(t *testing.T) {
	data, err := os.ReadFile("fixtures/statusline_input.json")
	if err != nil {
		t.Fatal(err)
	}

	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	if input.Model.ID != "claude-opus-4-6" {
		t.Errorf("model.id = %q, want %q", input.Model.ID, "claude-opus-4-6")
	}
	if input.Model.DisplayName != "Opus 4.6" {
		t.Errorf("model.display_name = %q, want %q", input.Model.DisplayName, "Opus 4.6")
	}
	if input.Workspace.CurrentDir != "/home/user/my-project" {
		t.Errorf("workspace.current_dir = %q, want %q", input.Workspace.CurrentDir, "/home/user/my-project")
	}
	if input.Vim.Mode != "INSERT" {
		t.Errorf("vim.mode = %q, want %q", input.Vim.Mode, "INSERT")
	}
	if input.ContextWindow.RemainingPercentage == nil {
		t.Fatal("context_window.remaining_percentage is nil")
	}
	if *input.ContextWindow.RemainingPercentage != 80 {
		t.Errorf("context_window.remaining_percentage = %v, want %v", *input.ContextWindow.RemainingPercentage, 80.0)
	}
}

func TestParseInputFixtureNoVim(t *testing.T) {
	data, err := os.ReadFile("fixtures/statusline_input.json")
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "vim")

	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("failed to parse fixture without vim: %v", err)
	}

	if input.Vim.Mode != "" {
		t.Errorf("vim.mode = %q, want empty string", input.Vim.Mode)
	}
}

func TestFindGitDirWorktree(t *testing.T) {
	// Simulate a worktree where .git is a file pointing to the real git dir
	root := t.TempDir()
	realGitDir := filepath.Join(root, "real-git-dir")
	if err := os.Mkdir(realGitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}

	gitFile := filepath.Join(worktree, ".git")
	content := "gitdir: " + realGitDir + "\n"
	if err := os.WriteFile(gitFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := findGitDir(worktree)
	if got != realGitDir {
		t.Errorf("findGitDir(%q) = %q, want %q", worktree, got, realGitDir)
	}
}
