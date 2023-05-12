package ooobot_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOoobot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ooobot Suite")
}
