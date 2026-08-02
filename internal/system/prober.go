package system

import (
	"os"
	"os/exec"
	"runtime"
)

type Prober interface {
	Env(string) string
	ReadFile(string) ([]byte, error)
	Stat(string) error
	LookPath(string) bool
	GOARCH() string
}

type RealProber struct{}

func (RealProber) Env(name string) string {
	return os.Getenv(name)
}

func (RealProber) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (RealProber) Stat(path string) error {
	_, err := os.Stat(path)
	return err
}

func (RealProber) LookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (RealProber) GOARCH() string {
	return runtime.GOARCH
}
