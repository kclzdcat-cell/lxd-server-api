package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureGolang(containerName, distro, version string) error {
	fmt.Printf("     安装Go语言...")
	
	time.Sleep(3 * time.Second)

	installCommands := getGolangInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Go语言失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Go环境...")
	configCommands := getGolangConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getGolangCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getGolangInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq golang",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y golang",
		}
	case "oracle":
		return []string{
			"yum install -y golang",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q go",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y go",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y golang",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getGolangConfigCommands() []string {
	return []string{
		"echo 'export GOPATH=/root/go' >> /root/.bashrc",
		"echo 'export PATH=$GOPATH/bin:$PATH' >> /root/.bashrc",
		"mkdir -p /root/go/{bin,src,pkg}",
	}
}

func getGolangCleanupCommands(distro string) []string {
	return []string{
		"go clean -cache -modcache || true",
	}
}

