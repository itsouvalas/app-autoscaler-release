package app_test

import (
	"acceptance/helpers"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	cfh "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"time"
)

var _ = Describe("AutoScaler dynamic policy", func() {
	var (
		appName string
		appGUID string
		policy  string

		doneChan       chan bool
		doneAcceptChan chan bool
		ticker         *time.Ticker
	)

	BeforeEach(func() {

	})

	JustBeforeEach(func() {
		appName = helpers.CreateTestApp(cfg, "dynamic-policy", initialInstanceCount)
		appGUID = helpers.GetAppGuid(cfg, appName)
		helpers.StartApp(appName, cfg.CfPushTimeoutDuration())
		helpers.WaitForNInstancesRunning(appGUID, initialInstanceCount, cfg.DefaultTimeoutDuration())
		instanceName = helpers.CreatePolicy(cfg, appName, appGUID, policy)
	})

	AfterEach(func() {
		if os.Getenv("SKIP_TEARDOWN") == "true" {
			fmt.Println("Skipping Teardown...")
		} else {
			DeletePolicy(appName, appGUID)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
		}
	})

	Context("when scaling by memoryused", func() {

		Context("when memory used is greater than scaling out threshold", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "memoryused", 30)
				initialInstanceCount = 1
			})

			It("should scale out", func() {
				helpers.WaitForNInstancesRunning(appGUID, 2, 3*time.Minute)
			})

		})

		Context("when  memory used is lower than scaling in threshold", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleInPolicy(1, 2, "memoryused", 80)
				initialInstanceCount = 2
			})
			It("should scale in", func() {
				helpers.WaitForNInstancesRunning(appGUID, 1, 3*time.Minute)
			})
		})

	})

	Context("when scaling by memoryutil", func() {

		Context("when memoryutil is greater than scaling out threshold", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "memoryutil", 20)
				initialInstanceCount = 1
			})

			It("should scale out", func() {
				helpers.WaitForNInstancesRunning(appGUID, 2, 3*time.Minute)
			})

		})

		Context("when  memoryutil is lower than scaling in threshold", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleInPolicy(1, 2, "memoryutil", 80)
				initialInstanceCount = 2
			})
			It("should scale in", func() {
				helpers.WaitForNInstancesRunning(appGUID, 1, 3*time.Minute)
			})
		})

	})

	Context("when scaling by responsetime", func() {

		JustBeforeEach(func() {
			doneChan = make(chan bool)
			doneAcceptChan = make(chan bool)
		})

		AfterEach(func() {
			close(doneChan)
			Eventually(doneAcceptChan, 10*time.Second).Should(Receive())
		})

		Context("when responsetime is greater than scaling out threshold", func() {

			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "responsetime", 2000)
				initialInstanceCount = 1
			})

			JustBeforeEach(func() {
				ticker = time.NewTicker(10 * time.Second)
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							Eventually(func() string {
								response := cfh.CurlApp(cfg, appName, "/slow/3000")
								if response == "" {
									return "dummy application with slow response"
								}
								return response
							}, 10*time.Second, 1*time.Second).Should(ContainSubstring("dummy application with slow response"))
						}
					}
				}(doneChan)
			})

			It("should scale out", func() {
				helpers.WaitForNInstancesRunning(appGUID, 2, 5*time.Minute)
			})
		})

		Context("when responsetime is less than scaling in threshold", func() {

			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleInPolicy(1, 2, "responsetime", 1000)
				initialInstanceCount = 2
			})

			JustBeforeEach(func() {
				ticker = time.NewTicker(2 * time.Second)
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							Eventually(func() string {
								response := cfh.CurlApp(cfg, appName, "/fast")
								if response == "" {
									return "dummy application with fast response"
								}
								return response
							}, 10*time.Second, 1*time.Second).Should(ContainSubstring("dummy application with fast response"))
						}
					}
				}(doneChan)
			})

			It("should scale in", func() {
				helpers.WaitForNInstancesRunning(appGUID, 1, 5*time.Minute)
			})
		})

	})

	Context("when scaling by throughput", func() {

		JustBeforeEach(func() {
			doneChan = make(chan bool)
			doneAcceptChan = make(chan bool)
		})

		AfterEach(func() {
			close(doneChan)
			Eventually(doneAcceptChan, 10*time.Second).Should(Receive())
		})

		Context("when throughput is greater than scaling out threshold", func() {

			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "throughput", 1)
				initialInstanceCount = 1
			})

			JustBeforeEach(func() {
				ticker = time.NewTicker(25 * time.Millisecond)
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							Eventually(func() string {
								response := cfh.CurlApp(cfg, appName, "/fast")
								if response == "" {
									return "dummy application with fast response"
								}
								return response
							}, 10*time.Second, 25*time.Millisecond).Should(ContainSubstring("dummy application with fast response"))
						}
					}
				}(doneChan)
			})

			It("should scale out", func() {
				helpers.WaitForNInstancesRunning(appGUID, 2, 5*time.Minute)
			})

		})

		Context("when throughput is less than scaling in threshold", func() {

			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleInPolicy(1, 2, "throughput", 100)
				initialInstanceCount = 2
			})

			JustBeforeEach(func() {
				ticker = time.NewTicker(10 * time.Second)
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							Eventually(func() string {
								response := cfh.CurlApp(cfg, appName, "/fast")
								if response == "" {
									return "dummy application with fast response"
								}
								return response
							}, 10*time.Second, 1*time.Second).Should(ContainSubstring("dummy application with fast response"))
						}
					}
				}(doneChan)
			})
			It("should scale in", func() {
				helpers.WaitForNInstancesRunning(appGUID, 1, 4*time.Minute)
			})
		})

	})

	// To check existing aggregated cpu metrics do: cf asm APP_NAME cpu
	// See: https://www.ibm.com/docs/de/cloud-private/3.2.0?topic=SSBS6K_3.2.0/cloud_foundry/integrating/cfee_autoscaler.html
	Context("when scaling by cpu", func() {

		BeforeEach(func() {
			policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "cpu", 5, 10)
			initialInstanceCount = 1
		})

		It("when cpu is greater than scaling out threshold", func() {

			By("should scale out to 2 instances")
			helpers.AppSetCpuUsage(cfg, appName, 50, 5)
			helpers.WaitForNInstancesRunning(appGUID, 2, 5*time.Minute)

			By("should scale in to 1 instance after cpu usage is reduced")
			for i := 0; i < 2; i++ {
				helpers.AppEndCpuTest(cfg, appName, i)
			}
			helpers.WaitForNInstancesRunning(appGUID, 1, 5*time.Minute)
		})
	})
})
