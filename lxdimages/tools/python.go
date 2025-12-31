package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigurePython(containerName, distro, version string) error {
	fmt.Printf("     安装Python3...")
	
	time.Sleep(3 * time.Second)

	installCommands := getPythonInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Python3失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置pip...")
	configCommands := getPythonConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getPythonCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getPythonInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq python3 python3-pip python3-venv python3-dev",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y python3 python3-pip python3-devel",
		}
	case "oracle":
		return []string{
			"yum install -y python3 python3-pip python3-devel",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q python3 py3-pip python3-dev",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y python3 python3-pip python3-devel",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y python3 python3-pip python3-devel",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getPythonConfigCommands() []string {
	return []string{
		"python3 -m pip install --upgrade pip",
		"pip3 config set global.index-url https://pypi.org/simple",
		"mkdir -p /root/.local/bin",
		"echo 'export PATH=/root/.local/bin:$PATH' >> /root/.bashrc",
	}
}

func getPythonCleanupCommands(distro string) []string {
	return []string{
		"pip3 cache purge || true",
		"rm -rf /root/.cache/pip",
	}
}

