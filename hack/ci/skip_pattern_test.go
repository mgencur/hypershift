package ci

import (
	"regexp"
	"testing"
)

const skipPattern = `(^(\.tekton|\.github|\.claude|docs|examples|enhancements|contrib|\.cursor|test/envtest)/)|(\.md$)|((^|/)OWNERS$)|(/overrides\.yaml$)|(^renovate\.json$)|(/\.testcoverage\.yml$)|(^\.gitlint$)|(^\.gitignore$)|(^\.coderabbit\.yaml$)|(^\.dockerignore$)|(^codecov\.yml$)|(^(api|availability-prober|client|cmd|contrib|control-plane-operator|control-plane-pki-operator|dnsresolver|etcd-backup|etcd-defrag|etcd-recovery|etcd-upload|hack|hypershift-ci-python|hypershift-operator|ignition-server|karpenter-operator|kas-bootstrap|konnectivity-https-proxy|konnectivity-socks5-proxy|kubevirtexternalinfra|kubernetes-default-proxy|oadp-debug|pkg|product-cli|shared-ingress|sharedingress-config-generator|support|sync-fg-configmap|sync-global-pullsecret|token-minter)/.*_test\.go$)`

func TestSkipPattern(t *testing.T) {
	re := regexp.MustCompile(skipPattern)

	tests := []struct {
		name  string
		path  string
		match bool
	}{
		// --- Files that SHOULD be skipped (match=true) ---

		// Documentation and config files
		{"top-level markdown", "README.md", true},
		{"nested markdown", "docs/design/proposal.md", true},
		{"OWNERS at root", "OWNERS", true},
		{"nested OWNERS", "hypershift-operator/controllers/OWNERS", true},
		{"docs directory", "docs/architecture/overview.txt", true},
		{"examples directory", "examples/cluster.yaml", true},
		{"enhancements directory", "enhancements/proposal.md", true},
		{"contrib directory", "contrib/scripts/helper.sh", true},
		{"tekton directory", ".tekton/pipeline.yaml", true},
		{"github directory", ".github/workflows/ci.yaml", true},
		{"claude directory", ".claude/settings.json", true},
		{"cursor directory", ".cursor/rules.json", true},
		{"test/envtest directory", "test/envtest/some_test.go", true},
		{"overrides yaml", "some/path/overrides.yaml", true},
		{"renovate json", "renovate.json", true},
		{"testcoverage yml", "some/.testcoverage.yml", true},
		{"gitlint", ".gitlint", true},
		{"gitignore", ".gitignore", true},
		{"coderabbit yaml", ".coderabbit.yaml", true},
		{"dockerignore", ".dockerignore", true},
		{"codecov yml", "codecov.yml", true},

		// Unit tests in non-E2E directories (the new clause)
		{"unit test in cmd", "cmd/install/install_test.go", true},
		{"unit test in api", "api/hypershift/v1beta1/types_test.go", true},
		{"unit test in client", "client/applyconfiguration/util_test.go", true},
		{"unit test in support", "support/util/visibility_test.go", true},
		{"unit test in support deep", "support/controlplane-component/defaults_test.go", true},
		{"unit test in hack", "hack/tools/generate_test.go", true},
		{"unit test in hypershift-operator", "hypershift-operator/controllers/hostedcluster/hostedcluster_controller_test.go", true},
		{"unit test in control-plane-operator", "control-plane-operator/controllers/hostedcontrolplane/v2/kas/deployment_test.go", true},
		{"unit test in karpenter-operator", "karpenter-operator/controllers/karpenter/karpenter_test.go", true},
		{"unit test in ignition-server", "ignition-server/controllers/local_test.go", true},
		{"unit test in availability-prober", "availability-prober/prober_test.go", true},
		{"unit test in token-minter", "token-minter/cmd/token_minter_test.go", true},
		{"unit test in control-plane-pki-operator", "control-plane-pki-operator/controllers/controller_test.go", true},
		{"unit test in pkg", "pkg/something/util_test.go", true},
		{"unit test in shared-ingress", "shared-ingress/controller_test.go", true},
		{"unit test in dnsresolver", "dnsresolver/resolver_test.go", true},
		{"unit test in kas-bootstrap", "kas-bootstrap/bootstrap_test.go", true},
		{"unit test in konnectivity-https-proxy", "konnectivity-https-proxy/proxy_test.go", true},
		{"unit test in product-cli", "product-cli/cmd/root_test.go", true},

		// --- Files that SHOULD NOT be skipped (match=false) ---

		// Source code changes
		{"go source in support", "support/util/visibility.go", false},
		{"go source in CPO", "control-plane-operator/controllers/hostedcontrolplane/v2/kas/deployment.go", false},
		{"go source in HO", "hypershift-operator/controllers/hostedcluster/hostedcluster_controller.go", false},
		{"go source in cmd", "cmd/install/install.go", false},
		{"yaml manifest", "control-plane-operator/controllers/hostedcontrolplane/v2/assets/kas/deployment.yaml", false},

		// E2E test files (must NOT be skipped)
		{"e2e test file", "test/e2e/util/util_test.go", false},
		{"e2e v2 test file", "test/e2e/v2/tests/multi_hop_upgrade_test.go", false},
		{"e2e v2 internal test", "test/e2e/v2/internal/env_vars_test.go", false},
		{"e2e v2 lifecycle test", "test/e2e/v2/lifecycle/azure_test.go", false},
		{"e2e integration test", "test/integration/control_plane_test.go", false},

		// Non-test Go files should never be skipped
		{"go source in api", "api/hypershift/v1beta1/types.go", false},
		{"go source in karpenter-operator", "karpenter-operator/controllers/karpenter/karpenter.go", false},

		// Edge cases
		{"test.go not _test.go", "support/util/test.go", false},
		{"_test.go at root", "_test.go", false},
		{"partial dir name match", "supportive/util_test.go", false},
		{"md in filename not extension", "cmd/markdown_parser.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := re.MatchString(tt.path)
			if got != tt.match {
				if tt.match {
					t.Errorf("path %q should be skipped but was not matched", tt.path)
				} else {
					t.Errorf("path %q should NOT be skipped but was matched", tt.path)
				}
			}
		})
	}
}
