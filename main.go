package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Input struct {
	Model struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Vim struct {
		Mode string `json:"mode"`
	} `json:"vim"`
	ContextWindow struct {
		RemainingPercentage *float64 `json:"remaining_percentage"`
		ContextWindowSize   *int     `json:"context_window_size"`
	} `json:"context_window"`
	RateLimits struct {
		FiveHour *struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       int64   `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay *struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       int64   `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
	Agent struct {
		Name string `json:"name"`
	} `json:"agent"`
	Worktree *struct {
		Name           string `json:"name"`
		Path           string `json:"path"`
		Branch         string `json:"branch"`
		OriginalCwd    string `json:"original_cwd"`
		OriginalBranch string `json:"original_branch"`
	} `json:"worktree"`
}

const (
	blue   = "\033[38;5;39m"
	orange = "\033[38;5;209m"
	green  = "\033[38;5;76m"
	yellow = "\033[38;5;11m"
	cyan   = "\033[38;5;14m"
	gray   = "\033[38;5;244m"
reset  = "\033[0m"
)

func gitInfo(cwd string) string {
	gitDir := findGitDir(cwd)
	if gitDir == "" {
		return ""
	}

	branch := branch(gitDir)
	if branch == "" {
		return ""
	}

	staged, unstaged, untracked, unknown := gitStatus(cwd)

	var color string
	var indicators string

	if unknown {
		color = gray
		indicators = "~"
	} else if staged || unstaged || untracked {
		if staged {
			indicators += "+"
		}
		if unstaged {
			indicators += "!"
		}
		if untracked {
			indicators += "?"
		}

		if untracked && !staged && !unstaged {
			color = cyan
		} else {
			color = yellow
		}
	} else {
		color = green
	}

	return fmt.Sprintf(" %s%s%s%s", color, branch, indicators, reset)
}

func findGitDir(cwd string) string {
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath
			}
			content, err := os.ReadFile(gitPath)
			if err == nil && strings.HasPrefix(string(content), "gitdir: ") {
				return strings.TrimSpace(strings.TrimPrefix(string(content), "gitdir: "))
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func branch(gitDir string) string {
	headPath := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}

	head := strings.TrimSpace(string(content))

	if after, ok := strings.CutPrefix(head, "ref: refs/heads/"); ok {
		return after
	}

	if len(head) >= 8 {
		return "@" + head[:8]
	}

	return ""
}

func gitStatus(cwd string) (staged, unstaged, untracked, unknown bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", cwd,
		"-c", "core.useBuiltinFSMonitor=false",
		"status", "--porcelain")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")

	output, err := cmd.Output()
	if err != nil {
		return false, false, false, true
	}

	staged, unstaged, untracked = parseGitStatus(string(output))
	return staged, unstaged, untracked, false
}

func parseGitStatus(output string) (staged, unstaged, untracked bool) {
	for line := range strings.SplitSeq(output, "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]

		// Staged: M=modified, A=added, D=deleted, R=renamed, C=copied, T=type change
		if x == 'M' || x == 'A' || x == 'D' || x == 'R' || x == 'C' || x == 'T' {
			staged = true
		}
		// Unstaged: M=modified, D=deleted, T=type change
		if y == 'M' || y == 'D' || y == 'T' {
			unstaged = true
		}
		// Conflict: U in either column, or DD/AA/AU/UA
		if x == 'U' || y == 'U' || (x == 'D' && y == 'D') || (x == 'A' && y == 'A') {
			staged = true
			unstaged = true
		}
		if x == '?' && y == '?' {
			untracked = true
		}
	}

	return
}

func formatWindowSize(size int) string {
	if size >= 1000000 {
		return fmt.Sprintf("%dM", size/1000000)
	}
	return fmt.Sprintf("%dk", size/1000)
}

func formatRateLimits(input *Input) string {
	fh := input.RateLimits.FiveHour
	sd := input.RateLimits.SevenDay

	if fh == nil && sd == nil {
		return ""
	}

	var parts []string
	if fh != nil {
		parts = append(parts, fmt.Sprintf("%s5h %s%.0f%%%s",
			gray, gray, fh.UsedPercentage, gray))
	}
	if sd != nil {
		parts = append(parts, fmt.Sprintf("%s7d %s%.0f%%%s",
			gray, gray, sd.UsedPercentage, gray))
	}

	return fmt.Sprintf(" · %s(%s%s)%s", gray, strings.Join(parts, gray+", "), gray, reset)
}

func formatContextWindow(input *Input) string {
	ctxPct := 0
	if input.ContextWindow.RemainingPercentage != nil {
		ctxPct = 100 - int(*input.ContextWindow.RemainingPercentage)
	}

	s := fmt.Sprintf("ctx %d%%", ctxPct)

	if input.ContextWindow.ContextWindowSize != nil {
		s += fmt.Sprintf(" %s[%s]%s", gray, formatWindowSize(*input.ContextWindow.ContextWindowSize), reset)
	}

	return s
}

func main() {
	var input Input
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "statusline: %v\n", err)
		return
	}

	cwd := input.Workspace.CurrentDir
	model := input.Model.DisplayName
	if model == "" {
		model = input.Model.ID
	}
	vimMode := input.Vim.Mode

	sym := "❯"
	if vimMode == "NORMAL" {
		sym = "❮"
	}

	home := os.Getenv("HOME")
	dirColor := orange
	shortDir := cwd
	if cwd == home {
		dirColor = blue
		shortDir = "~"
	} else if strings.HasPrefix(cwd, home+string(os.PathSeparator)) {
		dirColor = blue
		shortDir = "~" + strings.TrimPrefix(cwd, home)
	}

	gitStr := gitInfo(cwd)

	// Agent name before model
	agentStr := ""
	if input.Agent.Name != "" {
		agentStr = fmt.Sprintf("%s[%s]%s ", cyan, input.Agent.Name, reset)
	}

	ctxStr := formatContextWindow(&input)
	rlStr := formatRateLimits(&input)

	fmt.Printf("%s%s%s%s %s%s %s%s · %s%s%s\n",
		dirColor, shortDir, reset,
		gitStr,
		gray, sym,
		agentStr, model,
		ctxStr, rlStr, reset)
}
