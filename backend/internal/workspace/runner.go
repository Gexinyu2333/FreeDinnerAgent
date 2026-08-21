package workspace

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"freedinner/backend/internal/store"
)

var ErrUnsupportedSandbox = errors.New("unsupported workspace sandbox type")

type Runner interface {
	Execute(ctx context.Context, request CommandRequest) (CommandExecution, error)
}

type CommandRequest struct {
	Workspace       store.UserWorkspace
	Command         string
	Args            []string
	FullWorkingDir  string
	RelativeWorkDir string
	TimeoutSeconds  int
}

type CommandExecution struct {
	Status          string
	ExitCode        *int
	Stdout          string
	Stderr          string
	StdoutTruncated bool
	StderrTruncated bool
	DurationMS      int
	ErrorMessage    *string
	Metadata        map[string]any
}

type RunnerOptions struct {
	DockerBinary string
	PodmanBinary string
	NsJailBinary string
	SandboxImage string
}

type localRunner struct{}

func (r localRunner) Execute(ctx context.Context, request CommandRequest) (CommandExecution, error) {
	return executeProcess(ctx, exec.CommandContext, processSpec{
		Binary:         request.Command,
		Args:           request.Args,
		WorkingDir:     request.FullWorkingDir,
		TimeoutSeconds: request.TimeoutSeconds,
		Metadata: map[string]any{
			"runner":       "local_dir",
			"sandbox_type": request.Workspace.SandboxType,
		},
	}, request.Workspace)
}

type containerRunner struct {
	binary string
	image  string
	kind   string
}

func (r containerRunner) Execute(ctx context.Context, request CommandRequest) (CommandExecution, error) {
	args := buildContainerArgs(request, r.image)

	return executeProcess(ctx, exec.CommandContext, processSpec{
		Binary:         r.binary,
		Args:           args,
		WorkingDir:     filepath.Clean(request.Workspace.RootPath),
		TimeoutSeconds: request.TimeoutSeconds,
		Metadata: map[string]any{
			"runner":          r.kind,
			"sandbox_type":    request.Workspace.SandboxType,
			"container_image": r.image,
			"network_policy":  request.Workspace.NetworkPolicy,
		},
	}, request.Workspace)
}

type nsjailRunner struct {
	binary string
}

func (r nsjailRunner) Execute(ctx context.Context, request CommandRequest) (CommandExecution, error) {
	args := buildNsJailArgs(request)

	return executeProcess(ctx, exec.CommandContext, processSpec{
		Binary:         r.binary,
		Args:           args,
		WorkingDir:     filepath.Clean(request.Workspace.RootPath),
		TimeoutSeconds: request.TimeoutSeconds,
		Metadata: map[string]any{
			"runner":         "nsjail",
			"sandbox_type":   request.Workspace.SandboxType,
			"network_policy": request.Workspace.NetworkPolicy,
		},
	}, request.Workspace)
}

func buildContainerArgs(request CommandRequest, image string) []string {
	args := []string{
		"run",
		"--rm",
		"--workdir", containerWorkingDir(request.RelativeWorkDir),
		"--volume", filepath.Clean(request.Workspace.RootPath) + "/files:/workspace:rw",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--pids-limit", "64",
	}
	if request.Workspace.NetworkPolicy == "disabled" {
		args = append(args, "--network", "none")
	}
	if request.Workspace.MemoryLimitBytes != nil {
		args = append(args, "--memory", strconv.FormatInt(*request.Workspace.MemoryLimitBytes, 10))
	}
	if request.Workspace.CPULimit != nil && *request.Workspace.CPULimit != "" {
		args = append(args, "--cpus", *request.Workspace.CPULimit)
	}
	args = append(args, image, request.Command)
	args = append(args, request.Args...)
	return args
}

func buildNsJailArgs(request CommandRequest) []string {
	args := []string{
		"--quiet",
		"--mode", "o",
		"--time_limit", strconv.Itoa(request.TimeoutSeconds),
		"--cwd", containerWorkingDir(request.RelativeWorkDir),
		"--bindmount", filepath.Clean(request.Workspace.RootPath) + "/files:/workspace",
		"--rlimit_nproc", "64",
	}
	if request.Workspace.NetworkPolicy == "disabled" {
		args = append(args, "--clone_newnet")
	}
	if request.Workspace.MemoryLimitBytes != nil {
		args = append(args, "--rlimit_as", strconv.FormatInt(*request.Workspace.MemoryLimitBytes, 10))
	}
	args = append(args, "--", request.Command)
	args = append(args, request.Args...)
	return args
}

type commandFactory func(context.Context, string, ...string) *exec.Cmd

type processSpec struct {
	Binary         string
	Args           []string
	WorkingDir     string
	TimeoutSeconds int
	Metadata       map[string]any
}

func executeProcess(ctx context.Context, factory commandFactory, spec processSpec, workspace store.UserWorkspace) (CommandExecution, error) {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(spec.TimeoutSeconds)*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	startedAt := time.Now()
	cmd := factory(execCtx, spec.Binary, spec.Args...)
	cmd.Dir = spec.WorkingDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	duration := int(time.Since(startedAt).Milliseconds())

	status := "success"
	var exitCode *int
	var errorMessage *string
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		exitCode = &code
	}
	if execCtx.Err() == context.DeadlineExceeded {
		status = "timeout"
		message := "command timed out"
		errorMessage = &message
	} else if err != nil {
		status = "failed"
		message := err.Error()
		errorMessage = &message
	}

	stdoutText, stdoutTruncated := truncate(stdout.String(), workspace.MaxStdoutBytes)
	stderrText, stderrTruncated := truncate(stderr.String(), workspace.MaxStderrBytes)
	return CommandExecution{
		Status:          status,
		ExitCode:        exitCode,
		Stdout:          stdoutText,
		Stderr:          stderrText,
		StdoutTruncated: stdoutTruncated,
		StderrTruncated: stderrTruncated,
		DurationMS:      duration,
		ErrorMessage:    errorMessage,
		Metadata:        spec.Metadata,
	}, nil
}

func newRunner(sandboxType string, options RunnerOptions) (Runner, error) {
	switch sandboxType {
	case "", "local_dir":
		return localRunner{}, nil
	case "docker":
		return containerRunner{binary: valueOrDefault(options.DockerBinary, "docker"), image: valueOrDefault(options.SandboxImage, "freedinner-agent-sandbox:latest"), kind: "docker"}, nil
	case "podman":
		return containerRunner{binary: valueOrDefault(options.PodmanBinary, "podman"), image: valueOrDefault(options.SandboxImage, "freedinner-agent-sandbox:latest"), kind: "podman"}, nil
	case "nsjail":
		return nsjailRunner{binary: valueOrDefault(options.NsJailBinary, "nsjail")}, nil
	default:
		return nil, ErrUnsupportedSandbox
	}
}

func containerWorkingDir(relative string) string {
	if relative == "" || relative == "." {
		return "/workspace"
	}
	return "/workspace/" + filepath.ToSlash(relative)
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
