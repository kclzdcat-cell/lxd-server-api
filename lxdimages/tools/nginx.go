package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureNginx(containerName, distro, version string) error {
	fmt.Printf("     安装Nginx...")
	
	time.Sleep(3 * time.Second)

	installCommands := getNginxInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Nginx失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Nginx...")
	configCommands := getNginxConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用Nginx服务...")
	enableCommands := getNginxEnableCommands(distro)
	for _, cmdStr := range enableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getNginxCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getNginxInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nginx",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y nginx",
		}
	case "oracle":
		return []string{
			"yum install -y nginx",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q nginx",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y nginx",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y nginx",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getNginxConfigCommands() []string {
	return []string{
		"mkdir -p /var/www/html",
		"mkdir -p /etc/nginx/sites-available",
		"mkdir -p /etc/nginx/sites-enabled",
		"chown -R nginx:nginx /var/www/html 2>/dev/null || chown -R www-data:www-data /var/www/html",
	}
}

func getNginxEnableCommands(distro string) []string {
	switch distro {
	case "ubuntu", "debian", "centos", "fedora", "almalinux", "rockylinux", "oracle", "opensuse", "amazonlinux":
		return []string{
			"systemctl enable nginx",
		}
	case "alpine":
		return []string{
			"rc-update add nginx default",
		}
	default:
		return []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}
}

func getNginxCleanupCommands(distro string) []string {
	return []string{}
}

