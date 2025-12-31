package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureRedis(containerName, distro, version string) error {
	fmt.Printf("     安装Redis...")
	
	time.Sleep(3 * time.Second)

	installCommands := getRedisInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Redis失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Redis...")
	configCommands := getRedisConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用Redis服务...")
	enableCommands := getRedisEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getRedisCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getRedisInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq redis-server",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y redis",
		}
	case "oracle":
		return []string{
			"yum install -y redis",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q redis",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y redis",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y redis6",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getRedisConfigCommands() []string {
	return []string{
		"mkdir -p /var/lib/redis",
		"mkdir -p /var/run/redis",
		"sed -i 's/bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf 2>/dev/null || sed -i 's/bind 127.0.0.1/bind 0.0.0.0/' /etc/redis.conf",
		"sed -i 's/protected-mode yes/protected-mode no/' /etc/redis/redis.conf 2>/dev/null || sed -i 's/protected-mode yes/protected-mode no/' /etc/redis.conf",
	}
}

func getRedisEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"systemctl enable redis-server",
		}
	case "centos", "fedora", "almalinux", "rockylinux", "oracle", "opensuse", "amazonlinux":
		return []string{
			"systemctl enable redis",
		}
	case "alpine":
		return []string{
			"rc-update add redis default",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getRedisCleanupCommands(distro string) []string {
	return []string{}
}

