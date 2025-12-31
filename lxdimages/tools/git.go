package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureGit(containerName, distro, version string) error {
	fmt.Printf("     安装Git...")
	
	time.Sleep(3 * time.Second)

	installCommands := getGitInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Git失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Git...")
	configCommands := getGitConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getGitCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getGitInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq git",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y git",
		}
	case "oracle":
		return []string{
			"yum install -y git",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q git",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y git",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y git",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getGitConfigCommands() []string {
	return []string{
		"git config --global init.defaultBranch main",
		"git config --global color.ui auto",
	}
}

func getGitCleanupCommands(distro string) []string {
	return []string{}
}

