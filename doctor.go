package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type doctorReport struct {
	Version versionInfo      `json:"version"`
	Tools   []doctorToolInfo `json:"tools"`
}

type doctorToolInfo struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Path    string `json:"path"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

type doctorToolSpec struct {
	Name    string
	Command string
	Args    []string
}

func runDoctorText(opt Options) string {
	report := buildDoctorReport(opt)
	var b strings.Builder
	b.WriteString(report.Version.FullVersion)
	b.WriteString("\n\n工具检测:\n")
	for _, tool := range report.Tools {
		status := "可用"
		if tool.Status != "ok" {
			status = "不可用"
		}
		b.WriteString("- ")
		b.WriteString(tool.Name)
		b.WriteString(": ")
		b.WriteString(status)
		if tool.Path != "" {
			b.WriteString(" | ")
			b.WriteString(tool.Path)
		}
		if tool.Version != "" {
			b.WriteString(" | ")
			b.WriteString(tool.Version)
		}
		if tool.Error != "" {
			b.WriteString(" | ")
			b.WriteString(tool.Error)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func runDoctorJSON(opt Options) string {
	b, _ := json.MarshalIndent(buildDoctorReport(opt), "", "  ")
	return string(b) + "\n"
}

func buildDoctorReport(opt Options) doctorReport {
	return doctorReport{
		Version: currentVersionInfo(),
		Tools:   probeDoctorTools(doctorToolSpecs(opt), exec.LookPath, runToolVersion),
	}
}

func doctorToolSpecs(opt Options) []doctorToolSpec {
	ffmpeg := strings.TrimSpace(opt.FFmpegBinaryPath)
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	specs := []doctorToolSpec{
		{Name: "ffmpeg", Command: ffmpeg, Args: []string{"-version"}},
		{Name: "ffprobe", Command: ffprobeCommandForDoctor(ffmpeg), Args: []string{"-version"}},
		{Name: "mkvmerge", Command: "mkvmerge", Args: []string{"--version"}},
		{Name: "mp4decrypt", Command: "mp4decrypt", Args: []string{"--version"}},
		{Name: "shaka-packager", Command: "shaka-packager", Args: []string{"--version"}},
	}
	if value := strings.TrimSpace(opt.DecryptionBinaryPath); value != "" {
		specs = append(specs, doctorToolSpec{Name: "decryption-binary-path", Command: value, Args: []string{"--version"}})
	}
	return specs
}

func ffprobeCommandForDoctor(ffmpeg string) string {
	dir := filepath.Dir(ffmpeg)
	base := filepath.Base(ffmpeg)
	if strings.HasPrefix(base, "ffmpeg") && dir != "." {
		return filepath.Join(dir, strings.Replace(base, "ffmpeg", "ffprobe", 1))
	}
	return "ffprobe"
}

func probeDoctorTools(specs []doctorToolSpec, lookPath func(string) (string, error), runVersion func(context.Context, string, []string) (string, error)) []doctorToolInfo {
	tools := make([]doctorToolInfo, 0, len(specs))
	for _, spec := range specs {
		tool := doctorToolInfo{
			Name:    spec.Name,
			Command: spec.Command,
			Status:  "missing",
		}
		path, err := lookPath(spec.Command)
		if err != nil {
			tool.Error = err.Error()
			tools = append(tools, tool)
			continue
		}
		tool.Path = path
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		versionLine, err := runVersion(ctx, path, spec.Args)
		cancel()
		if err != nil {
			tool.Status = "error"
			tool.Error = err.Error()
			tools = append(tools, tool)
			continue
		}
		tool.Status = "ok"
		tool.Version = versionLine
		tools = append(tools, tool)
	}
	return tools
}

func runToolVersion(ctx context.Context, path string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return "", errors.New("读取版本超时")
	}
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return "", fmt.Errorf("%w: %s", err, firstNonEmptyLine(text))
		}
		return "", err
	}
	return firstNonEmptyLine(text), nil
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if value := strings.TrimSpace(line); value != "" {
			return value
		}
	}
	return ""
}
