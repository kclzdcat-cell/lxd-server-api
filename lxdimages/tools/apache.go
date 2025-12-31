package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureApache(containerName, distro, version string) error {
	fmt.Printf("     安装Apache...")
	
	time.Sleep(3 * time.Second)

	installCommands := getApacheInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Apache失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Apache...")
	configCommands := getApacheConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用Apache服务...")
	enableCommands := getApacheEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getApacheCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getApacheInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq apache2",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y httpd",
		}
	case "oracle":
		return []string{
			"yum install -y httpd",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q apache2",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y apache2",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y httpd",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getApacheConfigCommands() []string {
	return []string{
		"mkdir -p /var/www/html",
		"chown -R www-data:www-data /var/www/html 2>/dev/null || chown -R apache:apache /var/www/html 2>/dev/null || true",
	}
}

func getApacheEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"systemctl enable apache2",
		}
	case "centos", "fedora", "almalinux", "rockylinux", "oracle", "opensuse", "amazonlinux":
		return []string{
			"systemctl enable httpd",
		}
	case "alpine":
		return []string{
			"rc-update add apache2 default",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getApacheCleanupCommands(distro string) []string {
	return []string{}
}

