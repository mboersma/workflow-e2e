package tests

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

// scrapeProcs returns the sorted process names for an app from the given output.
// It matches the current "deis ps" output for a healthy container:
//   earthy-vocalist-v2-cmd-1d73e up (v2)
//   myapp-v16-web-bujlq up (v16)
func scrapeProcs(app string, output []byte) []string {
	re := regexp.MustCompile(fmt.Sprintf(`%s-v\d+-[\w-]+ up \(v\d+\)`, app))
	found := re.FindAll(output, -1)
	procs := make([]string, len(found))
	for i := range found {
		procs[i] = string(found[i])
	}
	sort.Strings(procs)
	return procs
}

var _ = FDescribe("Processes", func() {

	Context("with a deployed app", func() {

		appName := getRandAppName()
		appURL := strings.Replace(url, "deis", appName, 1)
		once := &sync.Once{}

		BeforeEach(func() {
			// Set up the Processes test app only once and assume the suite will clean up.
			once.Do(func() {
				os.Chdir("example-go")
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
		})

		DescribeTable("can scale up and down",

			func(scaleTo, respCode int) {
				// TODO: need some way to choose between "web" and "cmd" here!
				// scale the app's processes to the desired number
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

				// curl the app's root URL and print just the HTTP response code
				sess, err = start(`curl -sL -w "%{http_code}\\n" "%s" -o /dev/null`, appURL)
				Eventually(sess).Should(Say(strconv.Itoa(respCode)))
				Eventually(sess).Should(Exit(0))
			},

			Entry("scale to 1", 1, 200),
			Entry("scale to 3", 3, 200),
			Entry("scale to 0", 0, 502),
			Entry("scale to 5", 5, 200),
			Entry("scale to 0", 0, 502),
			Entry("scale to 1", 1, 200),
		)

		DescribeTable("can restart processes",

			func(restart string, scaleTo int, success bool, respCode int) {
				// TODO: need some way to choose between "web" and "cmd" here!
				// scale the app's processes to the desired number
				sess, err := start("deis ps:scale web=%d --app=%s", scaleTo, appName)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Scaling processes... but first,"))
				// Eventually(sess).Should(Say(`done in \d+s`))
				// Eventually(sess).Should(Say("=== %s Processes", appName))
				Eventually(sess).Should(Exit(0))

				// capture the process names
				beforeProcs := scrapeProcs(appName, sess.Out.Contents())

				// restart the app's process(es)
				var arg string
				switch restart {
				case "all":
					arg = ""
				case "by type":
					// TODO: need some way to choose between "web" and "cmd" here!
					arg = "web"
				case "by wrong type":
					// TODO: need some way to choose between "web" and "cmd" here!
					arg = "cmd"
				case "one":
					arg = beforeProcs[rand.Intn(len(beforeProcs))]
				}
				sess, err = start("deis ps:restart %s --app=%s", arg, appName)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Restarting processes... but first,"))
				// Eventually(sess).Should(Say(`done in \d+s`))
				// Eventually(sess).Should(Say("=== %s Processes", appName))
				Eventually(sess).Should(Exit(0))

				// capture the process names
				afterProcs := scrapeProcs(appName, sess.Out.Contents())

				// compare the before and after sets of process names
				Expect(beforeProcs).NotTo(Equal(afterProcs))

				// curl the app's root URL and print just the HTTP response code
				sess, err = start(`curl -sL -w "%{http_code}\\n" "%s" -o /dev/null`, appURL)
				Eventually(sess).Should(Say(strconv.Itoa(respCode)))
				Eventually(sess).Should(Exit(0))
			},

			Entry("restart one of 1", "one", 1, true, 200),
			Entry("restart all of 1", "all", 1, true, 200),
			Entry("restart all of 1 by type", "by type", 1, true, 200),
			Entry("restart all of 1 by wrong type", "by wrong type", 1, true, 200),
			Entry("restart one of 6", "one", 6, true, 200),
			Entry("restart all of 6", "all", 6, true, 200),
			Entry("restart all of 6 by type", "by type", 6, true, 200),
			Entry("restart all of 6 by wrong type", "by wrong type", 6, true, 200),
			Entry("restart all of 0", "all", 0, true, 502),
			Entry("restart all of 0 by type", "by type", 0, true, 502),
			Entry("restart all of 0 by wrong type", "by wrong type", 0, true, 502),
		)
	})
})
