package gannoy

import (
	"testing"
)

func TestVersionCheckSemverKernel(t *testing.T) {
	bytes := []byte("Linux 4.12.2-1")
	if validateKernel(bytes) != true {
		t.Errorf("Kernel Version is not less than 3.15.0")
	}
}

func TestVersionCheckElrepoKernel(t *testing.T) {
	bytes := []byte("Linux 4.12.2-1.el7.elrepo.x86_64")
	if validateKernel(bytes) != true {
		t.Errorf("Kernel Version is not less than 3.15.0")
	}
}

func TestVersionCheckKernel(t *testing.T) {
	bytes := []byte("Linux 3.10.0-327.36.1.el7.x86_64")
	if validateKernel(bytes) != false {
		t.Errorf("Kernel Version is less than 3.15.0")
	}
}

