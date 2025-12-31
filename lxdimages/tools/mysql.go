package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureMysql(containerName, distro, version string) error {
	fmt.Printf("     安装MySQL...")
	
	time.Sleep(3 * time.Second)

	installCommands := getMysqlInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装MySQL失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置MySQL...")
	configCommands := getMysqlConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用MySQL服务...")
	enableCommands := getMysqlEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getMysqlCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getMysqlInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq mysql-server",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y mysql-server",
		}
	case "oracle":
		return []string{
			"yum install -y mysql-server",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q mysql mysql-client",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y mysql mysql-server",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y mariadb105-server mariadb105",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getMysqlConfigCommands() []string {
	return []string{
		"mkdir -p /var/lib/mysql",
		"mkdir -p /var/run/mysqld",
		"chown -R mysql:mysql /var/lib/mysql /var/run/mysqld 2>/dev/null || true",
	}
}

func getMysqlEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"systemctl enable mysql",
		}
	case "centos", "fedora", "almalinux", "rockylinux", "oracle", "opensuse":
		return []string{
			"systemctl enable mysqld",
		}
	case "alpine":
		return []string{
			"rc-update add mysql default",
		}
	case "amazonlinux":
		return []string{
			"systemctl enable mariadb",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getMysqlCleanupCommands(distro string) []string {
	return []string{}
}

