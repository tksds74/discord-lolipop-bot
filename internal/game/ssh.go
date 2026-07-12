package game

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHConfig struct {
	Host           string
	Port           string
	User           string
	KeyPath        string
	PrivateKeyPEM  string
	KeyPassphrase  string
	KnownHostsPath string
	ConnectTimeout time.Duration
}

func (config SSHConfig) validate() error {
	if config.Host == "" {
		return errors.New("ssh host is required")
	}
	if config.User == "" {
		return errors.New("ssh user is required")
	}
	if config.KeyPath == "" && config.PrivateKeyPEM == "" {
		return errors.New("either ssh key path or private key content is required")
	}
	return nil
}

func (config SSHConfig) loadSigner() (ssh.Signer, error) {
	var keyBytes []byte
	if config.PrivateKeyPEM != "" {
		keyBytes = []byte(config.PrivateKeyPEM)
	} else {
		bytes, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read ssh key file: %w", err)
		}
		keyBytes = bytes
	}

	if config.KeyPassphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(config.KeyPassphrase))
	}
	return ssh.ParsePrivateKey(keyBytes)
}

func (config SSHConfig) clientConfig() (*ssh.ClientConfig, error) {
	signer, err := config.loadSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to load ssh private key: %w", err)
	}

	timeout := config.ConnectTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	hostKeyCallback, err := config.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:            config.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}, nil
}

func (config SSHConfig) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if config.KnownHostsPath == "" {
		log.Printf("[SSH] WARNING: SSH_KNOWN_HOSTS not set, host key verification is disabled")
		return ssh.InsecureIgnoreHostKey(), nil
	}

	callback, err := knownhosts.New(config.KnownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts file: %w", err)
	}
	return callback, nil
}

func (config SSHConfig) address() string {
	port := config.Port
	if port == "" {
		port = "22"
	}
	return net.JoinHostPort(config.Host, port)
}
