package upload_test

import (
	"os/exec"
	"runtime"
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

	Measure("uploads fast", func(b Benchmarker) {
		r := b.Time("runtime", func() {
			r := strings.NewReader("bar")
			upload, err := upload.New("127.0.0.1:4443", r)
			Ω(err).ShouldNot(HaveOccurred())
			success := upload.WaitForCompletion()
			Expect(success).To(Equal(true))
		})
		Ω(r.Seconds()).Should(BeNumerically("<", 0.02), "Upload should not take too long")
	}, 100)

	Measure("uploads without memory leak", func(b Benchmarker) {
		m := &runtime.MemStats{}
		var startAlloc uint64

		runtime.GC()
		runtime.ReadMemStats(m)
		startAlloc = m.Alloc

		r := strings.NewReader("baz")
		upload, err := upload.New("127.0.0.1:4443", r)
		Ω(err).ShouldNot(HaveOccurred())
		success := upload.WaitForCompletion()
		Expect(success).To(Equal(true))
		upload = nil

		// time.Sleep(50 * time.Millisecond) // FIXME with a short sleep, mem stats seem to be more accurate
		runtime.GC()
		runtime.ReadMemStats(m)
		allocDiff := m.Alloc - startAlloc
		if m.Alloc < startAlloc { // fix negative mem consumption read stat issue
			allocDiff = 0
		}

		// TODO assert for mem leak
		// Ω(allocDiff).Should(BeNumerically("<", 20*1024), "Upload should not leak too much")

		b.RecordValue("bytes allocated and not yet freed", float64(allocDiff))
	}, 20)
})
