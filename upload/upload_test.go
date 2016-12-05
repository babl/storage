package upload_test

import (
	"os/exec"
	"strings"
	"time"

	"github.com/larskluge/babl-storage/upload"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var pathToBablStorage string
var session *gexec.Session

var _ = BeforeSuite(func() {
	var err error
	pathToBablStorage, err = gexec.Build("github.com/larskluge/babl-storage")
	Ω(err).ShouldNot(HaveOccurred())

	command := exec.Command(pathToBablStorage, "-debug")
	session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	time.Sleep(1 * time.Second)
	Ω(session.ExitCode()).Should(Equal(-1)) // -1 to check if the process runs
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	session.Terminate().Wait()
})

var _ = Describe("Upload", func() {
	It("uploads", func() {
		r := strings.NewReader("foo")
		upload, err := upload.New("127.0.0.1:4443", r)
		Ω(err).ShouldNot(HaveOccurred())
		success := upload.WaitForCompletion()
		Expect(success).To(Equal(true))
	})
})
