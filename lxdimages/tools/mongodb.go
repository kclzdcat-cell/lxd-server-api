package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureMongodb(containerName, distro, version string) error {
	fmt.Printf("     安装MongoDB...")
	
	time.Sleep(3 * time.Second)

	installCommands := getMongodbInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装MongoDB失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置MongoDB...")
	configCommands := getMongodbConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用MongoDB服务...")
	enableCommands := getMongodbEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getMongodbCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getMongodbInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq mongodb",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y mongodb mongodb-server",
		}
	case "oracle":
		return []string{
			"yum install -y mongodb mongodb-server",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q mongodb mongodb-tools",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y mongodb mongodb-server",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y mongodb mongodb-server",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getMongodbConfigCommands() []string {
	return []string{
		"mkdir -p /var/lib/mongodb",
		"mkdir -p /var/log/mongodb",
		"chown -R mongod:mongod /var/lib/mongodb /var/log/mongodb 2>/dev/null || chown -R mongodb:mongodb /var/lib/mongodb /var/log/mongodb 2>/dev/null || true",
	}
}

func getMongodbEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian", "centos", "fedora", "almalinux", "rockylinux", "oracle", "amazonlinux":
		return []string{
			"systemctl enable mongod",
		}
	case "alpine":
		return []string{
			"rc-update add mongodb default",
		}
	case "opensuse":
		return []string{
			"systemctl enable mongodb",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getMongodbCleanupCommands(distro string) []string {
	return []string{}
}

