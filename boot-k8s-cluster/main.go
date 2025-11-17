package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/leosykes117/leonomano-doks-ops-pipelines/boot-k8s-cluster/pkg/config"
	exechelper "github.com/leosykes117/leonomano-doks-ops-pipelines/boot-k8s-cluster/pkg/exec-helper"
	"github.com/leosykes117/leonomano-doks-ops-pipelines/boot-k8s-cluster/pkg/logger"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Error("Failed to run pipeline", logger.Err(r.(error)))
		}
	}()
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}
	logger.Init(
		logger.WithLevel(logger.Level(cfg.Log.Level)),
		logger.WithFormat(cfg.Log.Format),
		logger.WithColor(cfg.Log.Color),
	)
	defer func() {
		_ = logger.L().Sync()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	CreateDOKubeCluster(ctx, cfg)
	ConfigExternalSecret(ctx, cfg)
}

func CreateDOKubeCluster(ctx context.Context, cfg *config.Config) {
	tfModulePath := "k8s-cluster"
	tgClusterModuleWorkDir := fmt.Sprintf("%s/%s/%s", cfg.Terragrunt.RootPath, cfg.Environment, tfModulePath)
	tgSourceDir := ""
	if cfg.Terragrunt.BaseModulesPath != "" {
		tgSourceDir = fmt.Sprintf("%s//k8s-cluster", cfg.Terragrunt.BaseModulesPath)
	}
	planOutPath := ""
	if !cfg.DryRun {
		planName := strings.Replace(tfModulePath, "/", "_", -1)
		planOutPath = fmt.Sprintf("%s.plan", planName)
	}
	extraArgs := make([]string, 0, 1)
	if !cfg.Log.Color || !cfg.Terragrunt.Log.Color {
		extraArgs = append(extraArgs, "--no-color")
	}
	executeTerragruntPlan(ctx, tgClusterModuleWorkDir, tgSourceDir, planOutPath, extraArgs)
	executeTerragruntApply(ctx, tgClusterModuleWorkDir, tgSourceDir, planOutPath, extraArgs, cfg.DryRun)
}

func ConfigExternalSecret(ctx context.Context, cfg *config.Config) {
	tfModulePath := "cluster-configuration/external-secrets-operator"
	tgClusterModuleWorkDir := fmt.Sprintf("%s/%s/%s", cfg.Terragrunt.RootPath, cfg.Environment, tfModulePath)
	tgSourceDir := ""
	if cfg.Terragrunt.BaseModulesPath != "" {
		tgSourceDir = fmt.Sprintf("%s//%s", cfg.Terragrunt.BaseModulesPath, tfModulePath)
	}
	planOutPath := ""
	if !cfg.DryRun {
		planName := strings.Replace(tfModulePath, "/", "_", -1)
		planOutPath = fmt.Sprintf("%s.plan", planName)
	}
	extraArgs := []string{
		"--feature=state_path_prefix=do-k8s",
		"--feature=kube_ctx=do-sfo2-dev-leonomano-projects",
	}
	executeTerragruntPlan(ctx, tgClusterModuleWorkDir, tgSourceDir, planOutPath, extraArgs)
	executeTerragruntApply(ctx, tgClusterModuleWorkDir, tgSourceDir, planOutPath, extraArgs, cfg.DryRun)
	HelmTemplateExternalSecret(ctx)
	HelmUpgradeExternalSecret(ctx)
}

func HelmTemplateExternalSecret(ctx context.Context) {
	helmBinPath, err := exec.LookPath("helm")
	if err != nil {
		log.Fatalf("not found helm binary: %s", err)
	}
	helmUpgradeArgs := []string{
		"template", "external-secrets", "external-secrets/external-secrets",
		"-n", "external-secrets",
		"--set", "installCRDs=true",
	}
	f, err := os.Create("./eso-generated-template.yaml")
	if err != nil {
		logger.L().Error("failed to create the yaml file for ESO helm char", logger.Err(err))
		panic(err)
	}
	defer f.Close()

	err = exechelper.Run(helmBinPath,
		exechelper.WithContext(ctx),
		exechelper.WithArgs(helmUpgradeArgs...),
		exechelper.WithStdout(f),
		exechelper.WithStderr(os.Stderr),
	)
	//exitCode, err := RunCmd(ctx, helmBinPath, "", helmUpgradeArgs)
	if err != nil {
		logger.L().Error(`Failed to run "helm template" command`, logger.Err(err))
		panic(err)
	}
	logger.L().Info("helm template finished" /* logger.Int("exitCode", exitCode) */)
}

func HelmUpgradeExternalSecret(ctx context.Context) {
	helmBinPath, err := exec.LookPath("helm")
	if err != nil {
		log.Fatalf("not found helm binary: %s", err)
	}
	// Archivo donde guardarás el YAML desde el marcador en adelante

	helmUpgradeArgs := []string{
		"upgrade", "external-secrets", "external-secrets/external-secrets", "--install",
		"-n", "external-secrets", "--create-namespace",
		"--set", "installCRDs=true",
		"--debug", "--wait",
	}
	helmUpgradeStream, helmUpgradeStout, err := os.Pipe()
	if err != nil {
		logger.L().Error("error generating io writer and reade for helm upgrade", logger.Err(err))
		panic(err)
	}

	f, err := os.Create("./eso-supplied-values.yaml")
	if err != nil {
		logger.L().Error("failed to create the yaml file for ESO applied values", logger.Err(err))
		panic(err)
	}
	defer f.Close()

	go func(r io.Reader, file io.Writer) {
		scanner := bufio.NewScanner(r)
		switchWriter := false
		for scanner.Scan() {
			line := scanner.Text()
			if switchWriter {
				_, err := f.WriteString(line)
				if err != nil {
					logger.L().Error("writting user supplied values to file fail", logger.Err(err))
				}
				continue
			}
			if strings.Contains(line, "USER-SUPPLIED VALUES:") {
				switchWriter = true
				continue
			}
			_, err := os.Stdout.WriteString(line)
			if err != nil {
				logger.L().Error("writting helm upgrade debug logs fail", logger.Err(err))
			}
		}
		if err := scanner.Err(); err != nil {
			logger.L().Error("scanning helm upgrade stdout fail", logger.Err(err))
		}
	}(helmUpgradeStream, f)

	err = exechelper.Run(helmBinPath,
		exechelper.WithContext(ctx),
		exechelper.WithArgs(helmUpgradeArgs...),
		exechelper.WithStdout(helmUpgradeStout),
		exechelper.WithStderr(os.Stderr),
	)
	if err != nil {
		logger.L().Error(`Failed to run "helm upgrade" command`, logger.Err(err))
		panic(err)
	}
	logger.L().Info("helm upgrade finished")
}

func executeTerragruntPlan(ctx context.Context, workDir, tgSource, planOutPath string, extraArgs []string) {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Debug("recovery from terragrunt plan fail")
		}
	}()
	terragruntBinPath, err := exec.LookPath("terragrunt")
	if err != nil {
		log.Fatalf("not found terragrunt binary: %s", err)
	}
	tgPlanArgs := []string{"plan"}
	logger.L().Info(fmt.Sprintf("tgSource=%s", tgSource))
	if tgSource != "" {
		tgPlanArgs = append(tgPlanArgs, "--source", tgSource)
	}
	if planOutPath != "" {
		tgPlanArgs = append(tgPlanArgs, "-out", planOutPath)
	}
	tgPlanArgs = append(tgPlanArgs, extraArgs...)
	err = exechelper.Run(terragruntBinPath,
		exechelper.WithContext(ctx),
		exechelper.WithDir(workDir),
		exechelper.WithArgs(tgPlanArgs...),
		exechelper.WithStdout(os.Stdout),
		exechelper.WithStderr(os.Stderr),
	)
	//exitCode, err := RunCmd(ctx, terragruntBinPath, workDir, tgPlanArgs)
	if err != nil {
		logger.L().Error(`Failed to run "terragrunt plan" command`, logger.Err(err))
		panic(err)
	}
	logger.L().Info("terragrunt plan finished" /* logger.Int("exitCode", exitCode) */)
}

func executeTerragruntApply(ctx context.Context, workDir, tgSource, planOutPath string, extraArgs []string, dryRun bool) {
	/* defer func() {
		if r := recover(); r != nil {
			logger.L().Debug("recovery from terragrunt apply fail")
		}
	}() */
	terragruntBinPath, err := exec.LookPath("terragrunt")
	if err != nil {
		log.Fatalf("not found terragrunt binary: %s", err)
	}
	//tgApplyArgs := []string{"apply", "-auto-approve"}
	argsSize := 2
	argsCap := argsSize + len(extraArgs)
	tgApplyArgs := make([]string, argsSize, argsCap)
	tgApplyArgs[0], tgApplyArgs[1] = "apply", "-auto-approve"
	tgApplyArgs = append(tgApplyArgs, extraArgs...)
	logger.L().Info(fmt.Sprintf("tgSource=%s", tgSource))
	if planOutPath != "" {
		tgApplyArgs = append(tgApplyArgs, planOutPath)
	}
	if tgSource != "" {
		tgApplyArgs = append(tgApplyArgs, "--source", tgSource)
	}
	if !dryRun {
		err = exechelper.Run(terragruntBinPath,
			exechelper.WithContext(ctx),
			exechelper.WithDir(workDir),
			exechelper.WithArgs(tgApplyArgs...),
			exechelper.WithStdout(os.Stdout),
			exechelper.WithStderr(os.Stderr),
		)
		//exitCode, err := RunCmd(ctx, terragruntBinPath, workDir, tgApplyArgs)
		if err != nil {
			logger.L().Error(`Failed to run "terragrunt apply" command`, logger.Err(err))
			panic(err)
		}
		logger.L().Info("terragrunt apply finished" /*logger.Int("exitCode", exitCode)*/)
	} else {
		logger.L().Debug("dry mode enabled, skipping apply execution",
			logger.String("command", fmt.Sprintf("%s %s", terragruntBinPath, strings.Join(tgApplyArgs, " "))),
		)
	}
}

func RunCmd(ctx context.Context, cmdName, workDir string, args []string) (int, error) {
	cmd := exec.CommandContext(ctx, cmdName, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logger.L().Error("cannot generate stdout pipe for command", logger.Err(err))
		return 5, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.L().Error("cannot generate stderr pipe for command", logger.Err(err))
		return 5, err
	}

	go func() {
		if err := streamCommandOutput(stdoutPipe); err != nil {
			logger.L().Error("stdout error", logger.Err(err))
		}
	}()
	go func() {
		if err := streamCommandOutput(stderrPipe); err != nil {
			logger.L().Error("stderr err", logger.Err(err))
		}
	}()

	logger.L().Info("Running command", logger.String("command", strings.Join(cmd.Args, " ")))
	if err = cmd.Start(); err != nil {
		logger.L().Error("failed to start the command", logger.Err(err))
		return 5, err
	}

	err = cmd.Wait()
	if errors.Is(err, context.DeadlineExceeded) {
		logger.L().Error("command timed out", logger.String("cmd", strings.Join(cmd.Args, " ")))
		return 124, err // 124 timeout standard code
	}

	return getCmdExitCode(cmd, err), err
}

func streamCommandOutput(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if scanner.Err() != nil {
		return fmt.Errorf("failed to scan from reader: %w", scanner.Err())
	}
	return nil
}

func getCmdExitCode(cmd *exec.Cmd, err error) int {
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		return -1
	}
	logger.L().Debug("No errors after running comand")
	if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}
	return 0
}
