package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// runCmd executes the command and formats output
func runCmd(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errMsg := ""
		if len(stderr.String()) != 0 {
			errMsg = fmt.Sprintf("stderr: %s", stderr.String())
		}
		if len(stdout.String()) != 0 {
			if len(errMsg) != 0 {
				errMsg = fmt.Sprintf("%s; stdout: %s", errMsg, stdout.String())
			} else {
				errMsg = fmt.Sprintf("stdout: %s", stdout.String())
			}
		}
		return errMsg, err
	}
	return stdout.String(), nil
}

// createShellCmd creates a shell command based on OS
func createShellCmd(ctx context.Context, cmdStr string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if ctx != nil {
			return exec.CommandContext(ctx, "powershell", "-Command", cmdStr)
		}
		return exec.Command("powershell", "-Command", cmdStr)
	}
	if ctx != nil {
		return exec.CommandContext(ctx, "bash", "-c", cmdStr)
	}
	return exec.Command("bash", "-c", cmdStr)
}

// Exec 本地执行命令 列如：("ls -ls")
func Exec(cmdStr string) (string, error) {
	cmd := createShellCmd(context.Background(), cmdStr)
	return runCmd(cmd)
}

// Execf 本地执行命令 列如：("ssh %s@%s", "user", "localhost")
func Execf(cmdStr string, a ...any) (string, error) {
	return Exec(fmt.Sprintf(cmdStr, a...))
}

// ExecWithTimeOut 本地执行命令超时退出 列如：("ssh user@localhost", 20*time.Second)
func ExecWithTimeOut(cmdStr string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := createShellCmd(ctx, cmdStr)
	out, err := runCmd(cmd)
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("errCmdTimeout %v", err)
	}
	return out, err
}

// ExecDirWithTimeOut 指定目录本地执行命令超时退出 列如：("ssh user@localhost", 20*time.Second)
func ExecDirWithTimeOut(workdir string, cmdStr string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := createShellCmd(ctx, cmdStr)
	cmd.Dir = workdir
	out, err := runCmd(cmd)
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("errCmdTimeout %v", err)
	}
	return out, err
}

// ExecDirScript 指定目录本地执行脚本文件, 默认超时10分钟 列如：("/tmp", "setup.sh")
func ExecDirScript(workDir, scriptPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-File", scriptPath)
	} else {
		cmd = exec.CommandContext(ctx, "bash", scriptPath)
	}

	cmd.Dir = workDir
	out, err := runCmd(cmd)
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("errCmdTimeout %v", err)
	}
	return out, err
}

// ExecCommand 执行命令程序带参数 例如：("ls", "-r", "-l", "-s")
func ExecCommand(name string, a ...string) (string, error) {
	cmd := exec.Command(name, a...)
	return runCmd(cmd)
}
