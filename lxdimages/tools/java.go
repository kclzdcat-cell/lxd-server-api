package tools

import (
	"fmt"
	"os/exec"
	"time"
)

func ConfigureJava(containerName, distro, version string) error {
	fmt.Printf("     安装Java...")
	
	time.Sleep(3 * time.Second)

	installCommands := getJavaInstallCommands(distro, version)
	for _, cmdStr := range installCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装Java失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置Java环境...")
	configCommands := getJavaConfigCommands()
	for _, cmdStr := range configCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     清理缓存...")
	cleanupCommands := getJavaCleanupCommands(distro)
	for _, cmdStr := range cleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getJavaInstallCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq default-jdk",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y java-latest-openjdk java-latest-openjdk-devel",
		}
	case "oracle":
		return []string{
			"yum install -y java-latest-openjdk java-latest-openjdk-devel",
		}
	case "alpine":
		return []string{
			"apk update -q",
			"apk add -q openjdk21",
		}
	case "opensuse":
		return []string{
			"zypper refresh",
			"zypper install -y java-openjdk java-openjdk-devel",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y java-21-amazon-corretto java-21-amazon-corretto-devel",
		}
	default:
		return []string{
			"echo '不支持的发行版' && exit 1",
		}
	}
}

func getJavaConfigCommands() []string {
	return []string{
		"JAVA_HOME=$(dirname $(dirname $(readlink -f $(which java)))) && echo \"export JAVA_HOME=$JAVA_HOME\" >> /root/.bashrc",
		"echo 'export PATH=$JAVA_HOME/bin:$PATH' >> /root/.bashrc",
	}
}

func getJavaCleanupCommands(distro string) []string {
	return []string{}
}

