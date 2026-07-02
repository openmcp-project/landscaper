// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
	"github.com/openmcp-project/landscaper/controller-utils/pkg/logging"
	lsutils "github.com/openmcp-project/landscaper/pkg/utils/landscaper"
	"github.com/openmcp-project/landscaper/test/framework"
	"github.com/openmcp-project/landscaper/test/utils"
)

// RenderErrorTests exercises the error path from issue #174: a deploy
// execution template that renders to invalid YAML must surface the failing
// item name, template execution and rendered snippet in the installation's
// status.lastError so the user can locate the problem without hand-rendering
// the template.
func RenderErrorTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "installations", "testdata", "render-error")
	)

	Describe("Render Error Context (issue #174)", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		log, err := logging.GetLogger()
		if err != nil {
			f.Log().Logfln("Error fetching logger: %v", err)
			return
		}

		BeforeEach(func() {
			ctx = context.Background()
			ctx = logging.NewContext(ctx, log)
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should surface the failing item name and rendered YAML snippet in status.lastError when the deploy execution renders invalid YAML", func() {
			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation with a deploy execution that renders invalid YAML")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, filepath.Join(testdataDir, "installation.yaml")))

			By("Wait for the Installation to reach phase Failed")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))

			By("Assert status.lastError carries the enriched render context")
			Expect(inst.Status.LastError).ToNot(BeNil(), "expected status.lastError to be populated on render failure")
			msg := inst.Status.LastError.Message

			// Identify the offending deploy item extracted from the rendered
			// output by best-effort scan (issue #174).
			Expect(msg).To(ContainSubstring(`item "myfirstitem"`))
			// Identify the enclosing deploy execution from the blueprint.
			Expect(msg).To(ContainSubstring(`template execution "default"`))
			// Include the rendered document snippet with the marker on the
			// failing line - this is what makes "line N" resolvable.
			Expect(msg).To(ContainSubstring("templated output:"))
			Expect(msg).To(ContainSubstring("manifest broken line"))
			Expect(msg).To(ContainSubstring("ˆ≈≈≈≈≈≈≈"))
		})
	})
}
