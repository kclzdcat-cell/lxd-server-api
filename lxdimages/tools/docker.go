package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureDocker(containerName, distro, version string) error {
	fmt.Printf("     安装Docker...")
	
	time.Sleep(3 * time.Second)

	installCommands := getDockerInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Docker失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Docker...")
	configCommands := getDockerConfigCommands(distro)
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用Docker服务...")
	enableCommands := getDockerEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getDockerCleanupCommands(distro, version)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getDockerInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq docker.io docker-compose",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y docker docker-compose",
		}
	case "oracle":
		return []string{
			"yum install -y docker-engine docker-compose",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q docker docker-cli-compose",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y docker docker-compose",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y docker",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getDockerConfigCommands(distro string) []string {
	return []string{
		"mkdir -p /etc/docker",
		"echo '{\"storage-driver\": \"overlay2\"}' > /etc/docker/daemon.json",
	}
}

func getDockerEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian", "centos", "fedora", "almalinux", "rockylinux", "oracle", "opensuse", "amazonlinux":
		return []string{
			"systemctl enable docker",
			"systemctl start docker",
		}
	case "alpine":
		return []string{
			"rc-update add docker default",
			"rc-service docker start",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getDockerCleanupCommands(distro string, version string) []string {
	return []string{
		"docker system prune -af || true",
	}
}

