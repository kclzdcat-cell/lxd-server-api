package tools

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type SSHConfig struct {
	InstallCommands []string
	ConfigCommands  []string
	EnableCommands  []string
	CleanupCommands []string
}

type DistroVersion struct {
	Distro  string
	Version string
}

var SupportedDistros = map[string][]string{
	"ubuntu":      {"jammy", "noble", "plucky"},
	"debian":      {"bullseye", "bookworm", "trixie"},
	"centos":      {"9-Stream", "10-Stream"},
	"fedora":      {"41", "42"},
	"almalinux":   {"8", "9", "10"},
	"rockylinux":  {"8", "9", "10"},
	"oracle":      {"8", "9"},
	"opensuse":    {"15.5", "15.6", "tumbleweed"},
	"alpine":      {"3.19", "3.20", "3.21", "3.22", "edge"},
	"amazonlinux": {"2023"},
}

func ValidateDistroVersion(distro, version string) error {
	versions, exists := SupportedDistros[distro]
	if !exists {
		return fmt.Errorf("不支持的发行版: %s，支持的发行版: ubuntu, debian, centos, fedora, almalinux, rockylinux, oracle, opensuse, alpine, amazonlinux", distro)
	}

	for _, v := range versions {
		if v == version {
			return nil
		}
	}

	return fmt.Errorf("不支持的 %s 版本: %s，支持的版本: %s", distro, version, strings.Join(versions, ", "))
}

func getVersionSpecificOptimizations(distro, version string) []string {
	key := distro + ":" + version
	
	optimizations := map[string][]string{
		"centos:9-Stream": {
			"dnf config-manager --set-enabled crb 2>/dev/null || true",
		},
		"centos:10-Stream": {
			"dnf config-manager --set-enabled crb 2>/dev/null || true",
		},
		"almalinux:8": {
			"dnf config-manager --set-enabled powertools 2>/dev/null || true",
		},
		"almalinux:9": {
			"dnf config-manager --set-enabled crb 2>/dev/null || true",
		},
		"almalinux:10": {
			"dnf config-manager --set-enabled crb 2>/dev/null || true",
		},
		"rockylinux:8": {
			"dnf config-manager --set-enabled powertools 2>/dev/null || true",
		},
		"rockylinux:9": {
			"dnf config-manager --set-enabled crb 2>/dev/null || true",
		},
		"rockylinux:10": {
			"dnf config-manager --set-enabled crb 2>/dev/null || true",
		},
		"oracle:7": {
			"yum-config-manager --enable ol7_optional_latest 2>/dev/null || true",
		},
		"oracle:8": {
			"dnf config-manager --set-enabled ol8_codeready_builder 2>/dev/null || true",
		},
		"oracle:9": {
			"dnf config-manager --set-enabled ol9_codeready_builder 2>/dev/null || true",
		},
	}
	
	if opts, exists := optimizations[key]; exists {
		return opts
	}
	
	return []string{}
}

func GetSSHConfig(distro string, version string) SSHConfig {
	config := SSHConfig{}

	switch distro {
	case "ubuntu":
		config.InstallCommands = []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq openssh-server sudo ca-certificates",
		}
	case "debian":
		config.InstallCommands = []string{
			"apt-get update -qq",
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq openssh-server sudo ca-certificates",
		}
	case "centos":
		config.InstallCommands = []string{
			"dnf install -y openssh-server sudo ca-certificates",
		}
	case "fedora":
		config.InstallCommands = []string{
			"dnf install -y openssh-server sudo ca-certificates",
		}
	case "almalinux", "rockylinux":
		config.InstallCommands = []string{
			"dnf install -y openssh-server sudo ca-certificates",
		}
	case "oracle":
		config.InstallCommands = []string{
			"yum install -y oracle-epel-release-el8 || yum install -y oracle-epel-release-el9 || true",
			"yum install -y openssh-server sudo ca-certificates",
		}
	case "alpine":
		config.InstallCommands = []string{
			"apk update -q",
			"apk add -q openssh-server sudo bash ca-certificates",
		}
	case "opensuse":
		config.InstallCommands = []string{
			"zypper refresh",
			"zypper install -y openssh-server sudo ca-certificates",
		}
	case "amazonlinux":
		config.InstallCommands = []string{
			"dnf install -y openssh-server sudo ca-certificates shadow-utils",
		}
	default:
		config.InstallCommands = []string{
			"echo '不支持的发行版' && exit 1",
		}
	}

	baseConfigCommands := []string{
		"mkdir -p /run/sshd /var/run/sshd",
		"mkdir -p /root/.ssh && chmod 700 /root/.ssh",
		"echo 'root:password' | chpasswd",
	}

	var sshConfigCommands []string
	switch distro {
	case "ubuntu", "debian":
		sshConfigCommands = []string{
			"sed -i.bak 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	case "centos", "fedora":
		sshConfigCommands = []string{
			"sed -i.bak 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	case "almalinux", "rockylinux":
		sshConfigCommands = []string{
			"sed -i.bak 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	case "oracle":
		sshConfigCommands = []string{
			"sed -i.bak 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	case "alpine":
		sshConfigCommands = []string{
			"sed -i.bak 's/^GatewayPort/#GatewayPort/' /etc/ssh/sshd_config",
			"sed -i 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	case "opensuse":
		sshConfigCommands = []string{
			"sed -i.bak 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	case "amazonlinux":
		sshConfigCommands = []string{
			"sed -i.bak 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
			"sed -i 's/#*Port.*/Port 22/' /etc/ssh/sshd_config",
		}
	default:
		sshConfigCommands = []string{
			"echo '不支持的发行版配置' && exit 1",
		}
	}
	config.ConfigCommands = append(baseConfigCommands, sshConfigCommands...)

	switch distro {
	case "ubuntu", "debian":
		config.EnableCommands = []string{
			"ssh-keygen -A",
			"sshd -t",
			"systemctl enable ssh",
			"systemctl start ssh",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		config.EnableCommands = []string{
			"ssh-keygen -A",
			"sshd -t",
			"systemctl enable sshd",
			"systemctl start sshd",
		}
	case "oracle":
		config.EnableCommands = []string{
			"ssh-keygen -A",
			"sshd -t",
			"systemctl enable sshd",
			"systemctl start sshd",
		}
	case "alpine":
		config.EnableCommands = []string{
			"ssh-keygen -A",
			"sshd -t",
			"rc-update add sshd default",
			"rc-service sshd start",
		}
	case "opensuse":
		config.EnableCommands = []string{
			"ssh-keygen -A",
			"sshd -t",
			"systemctl enable sshd",
			"systemctl start sshd",
		}
	case "amazonlinux":
		config.EnableCommands = []string{
			"ssh-keygen -A",
			"sshd -t",
			"systemctl enable sshd",
			"systemctl start sshd",
		}
	default:
		config.EnableCommands = []string{
			"echo '不支持的发行版启用命令' && exit 1",
		}
	}

	baseCleanupCommands := []string{
		"history -c",
		"rm -f /root/.bash_history",
		"rm -rf /tmp/* /var/tmp/*",
	}

	config.CleanupCommands = baseCleanupCommands

	return config
}

func ConfigureSSH(containerName, distro, version string) error {
	fmt.Printf("     安装SSH服务...")
	
	time.Sleep(3 * time.Second)

	config := GetSSHConfig(distro, version)

	for _, cmdStr := range config.InstallCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装SSH服务失败: %v", err)
		}
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     配置SSH服务...")
	for _, cmdStr := range config.ConfigCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     启用SSH服务...")
	for _, cmdStr := range config.EnableCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	fmt.Printf("     安装基础工具...")
	toolCommands := getBaseToolsCommands(distro, version)
	for _, cmdStr := range toolCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	optimizations := getVersionSpecificOptimizations(distro, version)
	if len(optimizations) > 0 {
		fmt.Printf("     应用版本优化...")
		for _, cmdStr := range optimizations {
			cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.Run()
		}
		fmt.Printf(" OK\n")
	}

	fmt.Printf("     清理SSH临时文件...")
	for _, cmdStr := range config.CleanupCommands {
		cmd := exec.Command("lxc", "exec", containerName, "--", "sh", "-c", cmdStr)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
	fmt.Printf(" OK\n")

	return nil
}

func getBaseToolsCommands(distro string, version string) []string {
	switch distro {
	case "ubuntu", "debian":
		return []string{
			"DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl wget nano procps net-tools",
		}
	case "centos", "fedora", "almalinux", "rockylinux":
		return []string{
			"dnf install -y curl wget nano procps-ng net-tools",
		}
	case "oracle":
		return []string{
			"yum install -y curl wget nano procps-ng net-tools",
		}
	case "alpine":
		return []string{
			"apk add -q curl wget nano procps net-tools",
		}
	case "opensuse":
		return []string{
			"zypper install -y curl wget nano procps net-tools",
		}
	case "amazonlinux":
		return []string{
			"dnf install -y curl wget nano procps-ng net-tools",
		}
	default:
		return []string{
			"echo '不支持的发行版工具安装' && exit 1",
		}
	}
}
