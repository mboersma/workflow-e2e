package tests

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("deis", func() {

	var appName string
	var appURL string

	Context("with a deployed app", func() {

		BeforeEach(func() {
			os.Chdir("example-go")
			appName = getRandAppName()
			appURL = strings.Replace(url, "deis", appName, 1)
			cmd := createApp(appName)
			Eventually(cmd).Should(SatisfyAll(
				Say("Git remote deis added"),
				Say("remote available at ")))
			Eventually(cmd).Should(Exit(0))
			cmd, err := start("GIT_SSH=%s git push deis master", gitSSH)
			Expect(err).NotTo(HaveOccurred())
			Eventually(cmd.Err, "2m").Should(Say("Done, %s:v2 deployed to Deis", appName))
			Eventually(cmd).Should(Exit(0))
		})

		AfterEach(func() {
			defer os.Chdir("..")
			destroyApp(appName)
		})

		It("can scale up and down", func() {

			// loop through several scale values
			for _, scaleTo := range []int{1, 3, 5, 1, 0, 1} {
				sess, err := start("deis ps:scale web=%d --app=%s", scaleTo, appName)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Scaling processes... but first,"))
				Eventually(sess).Should(Say(`done in \d+s`))
				Eventually(sess).Should(Say("=== %s Processes", appName))
				Eventually(sess).Should(Exit(0))

				// test that there are the right number of processes listed
				sess, err = start("deis ps:list --app=%s", appName)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("=== %s Processes", appName))
				Eventually(sess).Should(Exit(0))
				re := regexp.MustCompile(fmt.Sprintf(`%s-v\d-[\w-]+ up \(v\d\)`, appName))
				stdout := sess.Out.Contents()
				bytes := re.FindAll(stdout, -1)
				Expect(len(bytes)).To(Equal(scaleTo), string(stdout))

				if scaleTo > 0 {
					// TODO: curl app endpoint
				}
			}
		})

		XIt("can restart all processes", func() {
			// "deis ps:scale web=5 --app=%s"
			// "deis ps:list --app=%s"
			// curl app
			// "deis ps:restart web --app=%s"
			// "deis ps:list --app=%s"
			// curl app
		})

		XIt("can restart a specific process", func() {
			// "deis ps:restart web.1 --app=%s"
			// "deis ps:list --app=%s"
			// curl app
		})
	})
})
