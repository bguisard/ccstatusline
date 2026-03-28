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

	// New fields
	if input.ContextWindow.ContextWindowSize == nil {
		t.Fatal("context_window.context_window_size is nil")
	}
	if *input.ContextWindow.ContextWindowSize != 200000 {
		t.Errorf("context_window.context_window_size = %v, want %v", *input.ContextWindow.ContextWindowSize, 200000)
	}
	if input.RateLimits.FiveHour == nil {
		t.Fatal("rate_limits.five_hour is nil")
	}
	if input.RateLimits.FiveHour.UsedPercentage != 19 {
		t.Errorf("rate_limits.five_hour.used_percentage = %v, want %v", input.RateLimits.FiveHour.UsedPercentage, 19.0)
	}
	if input.RateLimits.SevenDay == nil {
		t.Fatal("rate_limits.seven_day is nil")
	}
	if input.RateLimits.SevenDay.UsedPercentage != 39 {
		t.Errorf("rate_limits.seven_day.used_percentage = %v, want %v", input.RateLimits.SevenDay.UsedPercentage, 39.0)
	}
	if input.Agent.Name != "security-reviewer" {
		t.Errorf("agent.name = %q, want %q", input.Agent.Name, "security-reviewer")
	}
	if input.Worktree == nil {
		t.Fatal("worktree is nil")
	}
	if input.Worktree.Name != "my-feature" {
		t.Errorf("worktree.name = %q, want %q", input.Worktree.Name, "my-feature")
	}
	if input.Worktree.Branch != "worktree-my-feature" {
		t.Errorf("worktree.branch = %q, want %q", input.Worktree.Branch, "worktree-my-feature")
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

func TestFormatWindowSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		expected string
	}{
		{"200k", 200000, "200k"},
		{"128k", 128000, "128k"},
		{"1M", 1000000, "1M"},
		{"2M", 2000000, "2M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWindowSize(tt.size)
			if got != tt.expected {
				t.Errorf("formatWindowSize(%d) = %q, want %q", tt.size, got, tt.expected)
			}
		})
	}
}

func TestFormatContextWindow(t *testing.T) {
	pct80 := 80.0
	size200k := 200000
	size1M := 1000000

	tests := []struct {
		name     string
		input    Input
		expected string
	}{
		{
			name:     "with percentage and 200k window",
			input:    Input{ContextWindow: struct{ RemainingPercentage *float64 `json:"remaining_percentage"`; ContextWindowSize *int `json:"context_window_size"` }{&pct80, &size200k}},
			expected: "ctx 20% " + gray + "[200k]" + reset,
		},
		{
			name:     "with percentage and 1M window",
			input:    Input{ContextWindow: struct{ RemainingPercentage *float64 `json:"remaining_percentage"`; ContextWindowSize *int `json:"context_window_size"` }{&pct80, &size1M}},
			expected: "ctx 20% " + gray + "[1M]" + reset,
		},
		{
			name:     "with percentage no window size",
			input:    Input{ContextWindow: struct{ RemainingPercentage *float64 `json:"remaining_percentage"`; ContextWindowSize *int `json:"context_window_size"` }{&pct80, nil}},
			expected: "ctx 20%",
		},
		{
			name:     "no percentage no window size",
			input:    Input{},
			expected: "ctx 0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatContextWindow(&tt.input)
			if got != tt.expected {
				t.Errorf("formatContextWindow() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatRateLimits(t *testing.T) {
	tests := []struct {
		name     string
		input    Input
		expected string
	}{
		{
			name:     "both nil",
			input:    Input{},
			expected: "",
		},
		{
			name: "both present green",
			input: func() Input {
				var i Input
				i.RateLimits.FiveHour = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{19, 0}
				i.RateLimits.SevenDay = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{39, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "5h " + gray + "19%" + gray + gray + ", " + gray + "7d " + gray + "39%" + gray + gray + ")" + reset,
		},
		{
			name: "five_hour only",
			input: func() Input {
				var i Input
				i.RateLimits.FiveHour = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{55, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "5h " + gray + "55%" + gray + gray + ")" + reset,
		},
		{
			name: "seven_day only",
			input: func() Input {
				var i Input
				i.RateLimits.SevenDay = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{85, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "7d " + gray + "85%" + gray + gray + ")" + reset,
		},
		{
			name: "boundary 50",
			input: func() Input {
				var i Input
				i.RateLimits.FiveHour = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{50, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "5h " + gray + "50%" + gray + gray + ")" + reset,
		},
		{
			name: "boundary 80",
			input: func() Input {
				var i Input
				i.RateLimits.FiveHour = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{80, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "5h " + gray + "80%" + gray + gray + ")" + reset,
		},
		{
			name: "boundary 81",
			input: func() Input {
				var i Input
				i.RateLimits.FiveHour = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{81, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "5h " + gray + "81%" + gray + gray + ")" + reset,
		},
		{
			name: "boundary 49 green",
			input: func() Input {
				var i Input
				i.RateLimits.FiveHour = &struct {
					UsedPercentage float64 `json:"used_percentage"`
					ResetsAt       int64   `json:"resets_at"`
				}{49, 0}
				return i
			}(),
			expected: " · " + gray + "(" + gray + "5h " + gray + "49%" + gray + gray + ")" + reset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRateLimits(&tt.input)
			if got != tt.expected {
				t.Errorf("formatRateLimits() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseInputAgentPresent(t *testing.T) {
	data := []byte(`{"agent": {"name": "security-reviewer"}}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.Agent.Name != "security-reviewer" {
		t.Errorf("agent.name = %q, want %q", input.Agent.Name, "security-reviewer")
	}
}

func TestParseInputAgentAbsent(t *testing.T) {
	data := []byte(`{}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.Agent.Name != "" {
		t.Errorf("agent.name = %q, want empty", input.Agent.Name)
	}
}

func TestParseInputAgentEmpty(t *testing.T) {
	data := []byte(`{"agent": {"name": ""}}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.Agent.Name != "" {
		t.Errorf("agent.name = %q, want empty", input.Agent.Name)
	}
}

func TestParseInputWorktreePresent(t *testing.T) {
	data := []byte(`{"worktree": {"name": "my-feature", "path": "/tmp/wt", "branch": "wt-branch", "original_cwd": "/tmp/orig", "original_branch": "main"}}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.Worktree == nil {
		t.Fatal("worktree is nil")
	}
	if input.Worktree.Name != "my-feature" {
		t.Errorf("worktree.name = %q, want %q", input.Worktree.Name, "my-feature")
	}
	if input.Worktree.Path != "/tmp/wt" {
		t.Errorf("worktree.path = %q, want %q", input.Worktree.Path, "/tmp/wt")
	}
	if input.Worktree.Branch != "wt-branch" {
		t.Errorf("worktree.branch = %q, want %q", input.Worktree.Branch, "wt-branch")
	}
	if input.Worktree.OriginalCwd != "/tmp/orig" {
		t.Errorf("worktree.original_cwd = %q, want %q", input.Worktree.OriginalCwd, "/tmp/orig")
	}
	if input.Worktree.OriginalBranch != "main" {
		t.Errorf("worktree.original_branch = %q, want %q", input.Worktree.OriginalBranch, "main")
	}
}

func TestParseInputWorktreeAbsent(t *testing.T) {
	data := []byte(`{}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.Worktree != nil {
		t.Errorf("worktree = %+v, want nil", input.Worktree)
	}
}

func TestParseInputRateLimitsAbsent(t *testing.T) {
	data := []byte(`{}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.RateLimits.FiveHour != nil {
		t.Errorf("rate_limits.five_hour = %+v, want nil", input.RateLimits.FiveHour)
	}
	if input.RateLimits.SevenDay != nil {
		t.Errorf("rate_limits.seven_day = %+v, want nil", input.RateLimits.SevenDay)
	}
}

func TestParseInputRateLimitsPartial(t *testing.T) {
	data := []byte(`{"rate_limits": {"five_hour": {"used_percentage": 42, "resets_at": 100}}}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.RateLimits.FiveHour == nil {
		t.Fatal("rate_limits.five_hour is nil")
	}
	if input.RateLimits.FiveHour.UsedPercentage != 42 {
		t.Errorf("five_hour.used_percentage = %v, want %v", input.RateLimits.FiveHour.UsedPercentage, 42.0)
	}
	if input.RateLimits.SevenDay != nil {
		t.Errorf("rate_limits.seven_day = %+v, want nil", input.RateLimits.SevenDay)
	}
}

func TestParseInputContextWindowSize(t *testing.T) {
	data := []byte(`{"context_window": {"remaining_percentage": 80, "context_window_size": 200000}}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.ContextWindow.ContextWindowSize == nil {
		t.Fatal("context_window.context_window_size is nil")
	}
	if *input.ContextWindow.ContextWindowSize != 200000 {
		t.Errorf("context_window.context_window_size = %v, want %v", *input.ContextWindow.ContextWindowSize, 200000)
	}
}

func TestParseInputContextWindowSizeAbsent(t *testing.T) {
	data := []byte(`{"context_window": {"remaining_percentage": 80}}`)
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if input.ContextWindow.ContextWindowSize != nil {
		t.Errorf("context_window.context_window_size = %v, want nil", *input.ContextWindow.ContextWindowSize)
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
