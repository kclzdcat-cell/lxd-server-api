package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureNodejs(containerName, distro, version string) error {
	fmt.Printf("     安装Node.js...")
	
	time.Sleep(3 * time.Second)

	installCommands := getNodejsInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Node.js失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置npm...")
	configCommands := getNodejsConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     安装pnpm...")
	pnpmCommands := []string{
		"npm install -g pnpm",
	}
	for _, cmdStr := range pnpmCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getNodejsCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getNodejsInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nodejs npm",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y nodejs npm",
		}
	case "oracle":
		return []string{
			"yum install -y nodejs npm",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q nodejs npm",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y nodejs npm",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y nodejs npm",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getNodejsConfigCommands() []string {
	return []string{
		"npm config set registry https://registry.npmjs.org/",
		"mkdir -p /root/.npm-global",
		"npm config set prefix /root/.npm-global",
		"echo 'export PATH=/root/.npm-global/bin:$PATH' >> /root/.bashrc",
	}
}

func getNodejsCleanupCommands(distro string) []string {
	return []string{
		"npm cache clean --force",
		"rm -rf /root/.npm",
	}
}

