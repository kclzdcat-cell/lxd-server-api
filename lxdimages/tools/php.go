package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigurePhp(containerName, distro, version string) error {
	fmt.Printf("     安装PHP...")
	
	time.Sleep(3 * time.Second)

	installCommands := getPhpInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装PHP失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置PHP...")
	configCommands := getPhpConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     安装Composer...")
	composerCommands := getComposerInstallCommands()
	for _, cmdStr := range composerCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getPhpCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getPhpInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq php php-fpm php-cli php-common php-mbstring php-xml php-curl php-zip php-mysql php-pgsql php-redis",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y php php-fpm php-cli php-common php-mbstring php-xml php-curl php-zip php-mysqlnd php-pgsql php-redis",
		}
	case "oracle":
		return []string{
			"yum install -y php php-fpm php-cli php-common php-mbstring php-xml php-curl php-zip php-mysqlnd php-pgsql",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q php php-fpm php-cli php-mbstring php-xml php-curl php-zip php-mysqli php-pgsql php-redis",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y php php-fpm php-cli php-mbstring php-curl php-zip php-mysql php-pgsql",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y php php-fpm php-cli php-common php-mbstring php-xml php-curl php-zip php-mysqlnd php-pgsql",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getPhpConfigCommands() []string {
	return []string{
		"mkdir -p /var/run/php",
		"sed -i 's/;cgi.fix_pathinfo=1/cgi.fix_pathinfo=0/' /etc/php*/fpm/php.ini 2>/dev/null || true",
		"sed -i 's/upload_max_filesize = 2M/upload_max_filesize = 64M/' /etc/php*/fpm/php.ini 2>/dev/null || true",
		"sed -i 's/post_max_size = 8M/post_max_size = 64M/' /etc/php*/fpm/php.ini 2>/dev/null || true",
	}
}

func getComposerInstallCommands() []string {
	return []string{
		"curl -sS https://getcomposer.org/installer | php",
		"mv composer.phar /usr/local/bin/composer",
		"chmod +x /usr/local/bin/composer",
	}
}

func getPhpCleanupCommands(distro string) []string {
	return []string{
		"composer clear-cache || true",
		"rm -rf /root/.composer/cache",
	}
}

