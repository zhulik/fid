package elect_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestElect(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elect Suite")
}
