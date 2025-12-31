package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigurePostgresql(containerName, distro, version string) error {
	fmt.Printf("     安装PostgreSQL...")
	
	time.Sleep(3 * time.Second)

	installCommands := getPostgresqlInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装PostgreSQL失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置PostgreSQL...")
	configCommands := getPostgresqlConfigCommands(distro)
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用PostgreSQL服务...")
	enableCommands := getPostgresqlEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getPostgresqlCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getPostgresqlInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq postgresql postgresql-contrib",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y postgresql-server postgresql-contrib",
		}
	case "oracle":
		return []string{
			"yum install -y postgresql-server postgresql-contrib",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q postgresql postgresql-contrib",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y postgresql-server postgresql-contrib",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y postgresql15-server postgresql15-contrib",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getPostgresqlConfigCommands(distro string) []string {
	switch distro {
	case "centos", "fedora", "almalinux", "rockylinux", "oracle", "amazonlinux":
		return []string{
			"postgresql-setup --initdb 2>/dev/null || postgresql-setup initdb 2>/dev/null || true",
		}
	case "alpine":
		return []string{
			"mkdir -p /var/lib/postgresql/data",
			"chown -R postgres:postgres /var/lib/postgresql",
			"su - postgres -c 'initdb -D /var/lib/postgresql/data' 2>/dev/null || true",
		}
	default:
		return []string{}
	}
}

func getPostgresqlEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian", "centos", "fedora", "almalinux", "rockylinux", "oracle", "opensuse", "amazonlinux":
		return []string{
			"systemctl enable postgresql",
		}
	case "alpine":
		return []string{
			"rc-update add postgresql default",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getPostgresqlCleanupCommands(distro string) []string {
	return []string{}
}

