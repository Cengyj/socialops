package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRootForDeployContractTest(t *testing.T) string {
	t.Helper()

	return filepath.Clean(filepath.Join("..", "..", ".."))
}

func readRepoFileForDeployContractTest(t *testing.T, parts ...string) string {
	t.Helper()

	pathParts := append([]string{repoRootForDeployContractTest(t)}, parts...)
	data, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read repo file %v: %v", parts, err)
	}
	return string(data)
}

func TestDockerComposePassesDocumentedRuntimeEnv(t *testing.T) {
	requiredRuntimeEnv := []string{
		"LOG_LEVEL",
		"LOG_FORMAT",
		"LOG_SERVICE_NAME",
		"LOG_ENV",
		"LOG_CALLER",
		"LOG_STACKTRACE_LEVEL",
		"LOG_OUTPUT_TO_STDOUT",
		"LOG_OUTPUT_TO_FILE",
		"LOG_OUTPUT_FILE_PATH",
		"LOG_ROTATION_MAX_SIZE_MB",
		"LOG_ROTATION_MAX_BACKUPS",
		"LOG_ROTATION_MAX_AGE_DAYS",
		"LOG_ROTATION_COMPRESS",
		"LOG_ROTATION_LOCAL_TIME",
		"LOG_SAMPLING_ENABLED",
		"LOG_SAMPLING_INITIAL",
		"LOG_SAMPLING_THEREAFTER",
		"SERVER_MAX_REQUEST_BODY_SIZE",
		"SERVER_H2C_ENABLED",
		"SERVER_H2C_MAX_CONCURRENT_STREAMS",
		"SERVER_H2C_IDLE_TIMEOUT",
		"SERVER_H2C_MAX_READ_FRAME_SIZE",
		"SERVER_H2C_MAX_UPLOAD_BUFFER_PER_CONNECTION",
		"SERVER_H2C_MAX_UPLOAD_BUFFER_PER_STREAM",
		"JWT_ACCESS_TOKEN_EXPIRE_MINUTES",
		"TOTP_ENCRYPTION_KEY",
		"UPDATE_PROXY_URL",
	}

	envExample := readRepoFileForDeployContractTest(t, "deploy", ".env.example")
	for _, key := range requiredRuntimeEnv {
		if !strings.Contains(envExample, key+"=") {
			t.Fatalf("deploy/.env.example must document %s", key)
		}
	}

	for _, composeFile := range []string{
		"docker-compose.yml",
		"docker-compose.local.yml",
		"docker-compose.standalone.yml",
	} {
		compose := readRepoFileForDeployContractTest(t, "deploy", composeFile)
		for _, key := range requiredRuntimeEnv {
			if !strings.Contains(compose, "- "+key+"=${"+key+":-") {
				t.Errorf("%s does not pass documented runtime env %s to socialops", composeFile, key)
			}
		}
	}
}

func TestDeployDocsDoNotExposeRemovedDashboardAggregationConfig(t *testing.T) {
	removed := []string{
		"DASHBOARD_AGGREGATION_ENABLED",
		"DASHBOARD_AGGREGATION_INTERVAL_SECONDS",
		"DASHBOARD_AGGREGATION_LOOKBACK_SECONDS",
		"DASHBOARD_AGGREGATION_BACKFILL_ENABLED",
		"DASHBOARD_AGGREGATION_BACKFILL_MAX_DAYS",
		"DASHBOARD_AGGREGATION_RECOMPUTE_DAYS",
		"DASHBOARD_AGGREGATION_RETENTION_USAGE_LOGS_DAYS",
		"DASHBOARD_AGGREGATION_RETENTION_HOURLY_DAYS",
		"DASHBOARD_AGGREGATION_RETENTION_DAILY_DAYS",
		"Dashboard Aggregation",
		"dashboard_aggregation",
	}
	for _, pathParts := range [][]string{
		{"deploy", ".env.example"},
		{"deploy", "docker-compose.yml"},
		{"deploy", "docker-compose.local.yml"},
		{"deploy", "docker-compose.standalone.yml"},
		{"deploy", "config.example.yaml"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)
		for _, marker := range removed {
			if strings.Contains(content, marker) {
				t.Fatalf("%s still documents removed dashboard aggregation config %q", name, marker)
			}
		}
	}
}

func TestDeployDocsDoNotExposeUnusedURLAllowlistConfig(t *testing.T) {
	removedEnv := []string{
		"SECURITY_URL_ALLOWLIST_ENABLED",
		"SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP",
		"SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS",
		"SECURITY_URL_ALLOWLIST_UPSTREAM_HOSTS",
	}
	for _, pathParts := range [][]string{
		{"deploy", ".env.example"},
		{"deploy", "docker-compose.yml"},
		{"deploy", "docker-compose.local.yml"},
		{"deploy", "docker-compose.standalone.yml"},
		{"deploy", "config.example.yaml"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)
		if strings.Contains(content, "url_allowlist") {
			t.Fatalf("%s still documents removed security.url_allowlist config", name)
		}
		for _, key := range removedEnv {
			if strings.Contains(content, key) {
				t.Fatalf("%s still documents removed runtime env %s", name, key)
			}
		}
	}
}

func TestDeployDocsDoNotExposeRemovedUserAPIKeyCacheConfig(t *testing.T) {
	removed := removedUserAPIKeyAuthCacheMarkersForDeployContractTest()
	for _, pathParts := range [][]string{
		{"deploy", ".env.example"},
		{"deploy", "docker-compose.yml"},
		{"deploy", "docker-compose.local.yml"},
		{"deploy", "docker-compose.standalone.yml"},
		{"deploy", "config.example.yaml"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)
		for _, marker := range removed {
			if strings.Contains(content, marker) {
				t.Fatalf("%s still documents removed user API Key auth cache config %q", name, marker)
			}
		}
	}
}

func removedUserAPIKeyAuthCacheMarkersForDeployContractTest() []string {
	envParts := []string{"api", "key", "auth", "cache"}
	return []string{
		strings.Join(envParts, "_"),
		strings.ToUpper(strings.Join(envParts, "_")),
		strings.Join([]string{"API", "Key", "Auth", "Cache"}, " "),
		strings.Join([]string{"API", "Key", "认证缓存"}, " "),
	}
}

func TestManagedPostgresComposePassesDocumentedServerTuning(t *testing.T) {
	requiredPostgresTuning := map[string]string{
		"POSTGRES_MAX_CONNECTIONS":      "max_connections",
		"POSTGRES_SHARED_BUFFERS":       "shared_buffers",
		"POSTGRES_EFFECTIVE_CACHE_SIZE": "effective_cache_size",
		"POSTGRES_MAINTENANCE_WORK_MEM": "maintenance_work_mem",
	}

	envExample := readRepoFileForDeployContractTest(t, "deploy", ".env.example")
	for key := range requiredPostgresTuning {
		if !strings.Contains(envExample, key+"=") {
			t.Fatalf("deploy/.env.example must document %s", key)
		}
	}

	for _, composeFile := range []string{
		"docker-compose.yml",
		"docker-compose.local.yml",
		"docker-compose.dev.yml",
	} {
		compose := readRepoFileForDeployContractTest(t, "deploy", composeFile)
		for envKey, pgSetting := range requiredPostgresTuning {
			if !strings.Contains(compose, pgSetting+"=${"+envKey+":-") {
				t.Errorf("%s does not pass documented %s to postgres %s", composeFile, envKey, pgSetting)
			}
		}
	}
}

func TestManagedRedisComposePassesDocumentedServerTuning(t *testing.T) {
	envExample := readRepoFileForDeployContractTest(t, "deploy", ".env.example")
	if !strings.Contains(envExample, "REDIS_MAXCLIENTS=") {
		t.Fatal("deploy/.env.example must document REDIS_MAXCLIENTS")
	}

	for _, composeFile := range []string{
		"docker-compose.yml",
		"docker-compose.local.yml",
		"docker-compose.dev.yml",
	} {
		compose := readRepoFileForDeployContractTest(t, "deploy", composeFile)
		if !strings.Contains(compose, "--maxclients ${REDIS_MAXCLIENTS:-") {
			t.Errorf("%s does not pass documented REDIS_MAXCLIENTS to redis-server maxclients", composeFile)
		}
	}
}

func TestLocalComposeOverrideKeepsCurrentManagedServiceImages(t *testing.T) {
	content := readRepoFileForDeployContractTest(t, "deploy", "docker-compose.local-override.yml")
	for _, current := range []string{
		"image: postgres:18-alpine",
		"image: redis:8-alpine",
	} {
		if !strings.Contains(content, current) {
			t.Fatalf("deploy/docker-compose.local-override.yml must keep current managed service image %q", current)
		}
	}
	for _, stale := range []string{
		"image: postgres:15",
		"image: redis:7-alpine",
	} {
		if strings.Contains(content, stale) {
			t.Fatalf("deploy/docker-compose.local-override.yml still pins stale managed service image %q", stale)
		}
	}
}

func TestReleaseMetadataUsesCurrentProductPositioning(t *testing.T) {
	for _, pathParts := range [][]string{
		{".goreleaser.yaml"},
		{".goreleaser.simple.yaml"},
		{"Dockerfile.goreleaser"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)

		for _, removedPositioning := range []string{
			"AI API Gateway",
			"AI 订阅配额",
		} {
			if strings.Contains(content, removedPositioning) {
				t.Fatalf("%s still contains removed product positioning %q", name, removedPositioning)
			}
		}
	}
}

func TestRootReadmesDocumentCurrentTwitterExecutorBoundary(t *testing.T) {
	checks := []struct {
		pathParts []string
		required  string
		stale     []string
	}{
		{
			pathParts: []string{"README.md"},
			required:  "Twitter/X execution currently uses the existing submit, validate, queue, executor, log, and billing flow",
			stale: []string{
				"real social executors are attached later",
				"future backend execution workers",
				"does not implement account purchase flow first, real social-platform executors",
			},
		},
		{
			pathParts: []string{"README_CN.md"},
			required:  "当前 Twitter/X 执行链路已经接入提交、校验、排队、执行器、日志和计费流程",
			stale: []string{
				"真实执行器后续接入",
				"当前阶段不实现真实社交平台执行器",
			},
		},
		{
			pathParts: []string{"README_JA.md"},
			required:  "現在の Twitter/X 実行フローは",
			stale: []string{
				"実行ワーカーは後続で接続します",
				"実際のソーシャルプラットフォーム実行器",
			},
		},
	}

	for _, check := range checks {
		content := readRepoFileForDeployContractTest(t, check.pathParts...)
		name := filepath.Join(check.pathParts...)
		if !strings.Contains(content, check.required) {
			t.Fatalf("%s must document the current Twitter/X executor boundary", name)
		}
		for _, stale := range check.stale {
			if strings.Contains(content, stale) {
				t.Fatalf("%s still documents stale executor boundary %q", name, stale)
			}
		}
	}
}

func TestRepositoryRootDoesNotShipDefaultNpmScaffold(t *testing.T) {
	rootPackage := filepath.Join(repoRootForDeployContractTest(t), "package.json")
	content, err := os.ReadFile(rootPackage)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read root package.json: %v", err)
	}
	for _, stale := range []string{
		`"name": "socialops-main"`,
		`"test": "echo \"Error: no test specified\" && exit 1"`,
		`"license": "ISC"`,
	} {
		if strings.Contains(string(content), stale) {
			t.Fatalf("root package.json still contains default npm scaffold %q; use frontend/package.json for the Vue app", stale)
		}
	}
}

func TestLocalDeployRuntimeArtifactsStayIgnored(t *testing.T) {
	patterns := []string{
		"deploy/dev-*.log",
		"deploy/dev-data/",
		"deploy/dev-postgres_data/",
	}
	for _, ignoreFile := range []string{".gitignore", ".ignore"} {
		content := readRepoFileForDeployContractTest(t, ignoreFile)
		for _, pattern := range patterns {
			if !strings.Contains(content, pattern) {
				t.Fatalf("%s must keep local deploy runtime artifact pattern %q", ignoreFile, pattern)
			}
		}
	}
}

func TestRootMakefileDoesNotExposeTargetsForMissingModules(t *testing.T) {
	root := repoRootForDeployContractTest(t)
	if _, err := os.Stat(filepath.Join(root, "datamanagement")); !os.IsNotExist(err) {
		return
	}

	content := readRepoFileForDeployContractTest(t, "Makefile")
	for _, removedTarget := range []string{
		"build-datamanagementd",
		"test-datamanagementd",
		"cd datamanagement",
	} {
		if strings.Contains(content, removedTarget) {
			t.Fatalf("root Makefile exposes %q even though datamanagement module is absent", removedTarget)
		}
	}
}

func TestDeployDoesNotShipDatamanagementdInstallPathWhenModuleIsAbsent(t *testing.T) {
	root := repoRootForDeployContractTest(t)
	if _, err := os.Stat(filepath.Join(root, "datamanagement")); !os.IsNotExist(err) {
		return
	}

	for _, pathParts := range [][]string{
		{"deploy", "install-datamanagementd.sh"},
		{"deploy", "socialops-datamanagementd.service"},
	} {
		path := filepath.Join(append([]string{root}, pathParts...)...)
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("%s is a stale datamanagementd deployment artifact; the datamanagement module is absent", filepath.Join(pathParts...))
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", filepath.Join(pathParts...), err)
		}
	}

	readme := readRepoFileForDeployContractTest(t, "deploy", "README.md")
	for _, stale := range []string{
		"install-datamanagementd.sh",
		"socialops-datamanagementd.service",
		"datamanagementd（数据管理）联动",
		"额外部署宿主机 `datamanagementd`",
	} {
		if strings.Contains(readme, stale) {
			t.Fatalf("deploy/README.md documents stale datamanagementd artifact or setup flow %q while the module is absent", stale)
		}
	}

	doc := readRepoFileForDeployContractTest(t, "deploy", "DATAMANAGEMENTD_CN.md")
	for _, required := range []string{
		"当前开源仓库不随附 datamanagementd 源码模块",
		"不要从当前仓库执行 datamanagementd 源码构建",
	} {
		if !strings.Contains(doc, required) {
			t.Fatalf("deploy/DATAMANAGEMENTD_CN.md must document current datamanagementd deployment boundary %q", required)
		}
	}
	for _, stale := range []string{
		"go build -o /opt/socialops/datamanagementd ./cmd/datamanagementd",
		"install-datamanagementd.sh",
		"sudo ./deploy/install-datamanagementd.sh",
	} {
		if strings.Contains(doc, stale) {
			t.Fatalf("deploy/DATAMANAGEMENTD_CN.md still documents stale datamanagementd source/install flow %q", stale)
		}
	}
}

func TestDeployMakefileTargetsBackendModuleFromDeployDirectory(t *testing.T) {
	root := repoRootForDeployContractTest(t)
	if _, err := os.Stat(filepath.Join(root, "deploy", "cmd", "server")); err == nil {
		return
	}

	content := readRepoFileForDeployContractTest(t, "deploy", "Makefile")
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, broken := range []string{
			"cd cmd/server",
			"go build -o bin/server ./cmd/server",
			"go build -tags embed -o bin/server ./cmd/server",
			"go test -tags unit ./... -count=1",
			"go test -tags integration ./... -count=1",
			"go test -tags e2e ./internal/integration/...",
		} {
			if strings.Contains(trimmed, broken) && !strings.Contains(trimmed, "BACKEND_DIR") {
				t.Fatalf("deploy/Makefile uses backend-relative command %q without targeting BACKEND_DIR", broken)
			}
		}
	}
	if !strings.Contains(content, "BACKEND_DIR") || !strings.Contains(content, "../backend") {
		t.Fatal("deploy/Makefile must explicitly target the backend module from the deploy directory")
	}
}

func TestRootDockerfileDoesNotCopyMissingRemovedResources(t *testing.T) {
	root := repoRootForDeployContractTest(t)
	if _, err := os.Stat(filepath.Join(root, "backend", "resources")); err == nil {
		return
	}

	content := readRepoFileForDeployContractTest(t, "Dockerfile")
	if strings.Contains(content, "/app/backend/resources") || strings.Contains(content, "/app/resources") {
		t.Fatal("root Dockerfile copies backend/resources even though SocialOps no longer ships that removed resource directory")
	}
}

func TestDockerfilesUseCurrentBuildAndRuntimeContract(t *testing.T) {
	required := []string{
		"ARG ALPINE_IMAGE=alpine:3.21",
		"ARG POSTGRES_IMAGE=postgres:18-alpine",
		"corepack prepare pnpm@9 --activate",
		"VERSION_VALUE=\"${VERSION}\"",
		"-buildvcs=false",
		"-trimpath",
		"-X main.Version=${VERSION_VALUE}",
		"FROM ${POSTGRES_IMAGE} AS pg-client",
		"COPY --from=pg-client /usr/local/bin/pg_dump /usr/local/bin/pg_dump",
		"COPY --from=pg-client /usr/local/bin/psql /usr/local/bin/psql",
		"LABEL description=\"SocialOps - Website Account Pool Social Operations Platform\"",
	}
	forbidden := []string{
		"corepack prepare pnpm@latest --activate",
		"ARG VERSION=docker",
		"Social Account Rental & Task Distribution Platform",
	}

	for _, pathParts := range [][]string{
		{"Dockerfile"},
		{"deploy", "Dockerfile"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)

		for _, current := range required {
			if !strings.Contains(content, current) {
				t.Errorf("%s must include current Docker build/runtime contract %q", name, current)
			}
		}
		for _, staleContract := range forbidden {
			if strings.Contains(content, staleContract) {
				t.Errorf("%s still contains stale Docker build/runtime contract %q", name, staleContract)
			}
		}
	}
}

func TestRepositoryDoesNotShipUnsupportedBackendOnlyDockerfile(t *testing.T) {
	root := repoRootForDeployContractTest(t)
	path := filepath.Join(root, "backend", "Dockerfile")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("backend/Dockerfile is an unsupported stale backend-only image path; use the root Dockerfile or deploy/Dockerfile")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat backend/Dockerfile: %v", err)
	}
}

func TestDockerignoreExcludesLocalGeneratedArtifacts(t *testing.T) {
	content := readRepoFileForDeployContractTest(t, ".dockerignore")
	for _, pattern := range []string{
		".codex-artifacts/",
		".codex-logs/",
		".codex-qa/",
		".codex-screenshots/",
		"artifacts/",
		"codex-screenshots/",
		"qa-artifacts/",
		"qa-screenshots/",
	} {
		if !strings.Contains(content, pattern) {
			t.Errorf(".dockerignore must exclude local generated artifact directory %q", pattern)
		}
	}
}

func TestDeployEnvExampleDoesNotDocumentUnsupportedRuntimeEnv(t *testing.T) {
	content := readRepoFileForDeployContractTest(t, "deploy", ".env.example")
	for _, unsupported := range []string{
		"RATE_LIMIT_OVERLOAD_COOLDOWN_MINUTES",
		"upstream returns 529",
		"上游返回 529",
	} {
		if strings.Contains(content, unsupported) {
			t.Fatalf("deploy/.env.example documents unsupported stale runtime env or wording %q", unsupported)
		}
	}
}

func TestDockerHubDocsUseCurrentConfigEnv(t *testing.T) {
	docs := readRepoFileForDeployContractTest(t, "deploy", "DOCKER.md")

	for _, unsupportedEnv := range []string{"DATABASE_URL", "REDIS_URL", "GIN_MODE"} {
		if strings.Contains(docs, unsupportedEnv) {
			t.Fatalf("deploy/DOCKER.md still documents unsupported %s configuration", unsupportedEnv)
		}
	}

	for _, current := range []string{"DATABASE_HOST", "DATABASE_PASSWORD", "REDIS_HOST", "SERVER_PORT", "SERVER_MODE"} {
		if !strings.Contains(docs, current) {
			t.Fatalf("deploy/DOCKER.md must document current %s configuration", current)
		}
	}
}

func TestDockerDeployReadmeMatchesGeneratedComposeName(t *testing.T) {
	script := readRepoFileForDeployContractTest(t, "deploy", "docker-deploy.sh")
	if !strings.Contains(script, "-o docker-compose.yml") {
		t.Fatal("deploy/docker-deploy.sh must continue writing the generated compose file to docker-compose.yml")
	}

	readme := readRepoFileForDeployContractTest(t, "deploy", "README.md")
	start := strings.Index(readme, "**After running the script:**")
	if start == -1 {
		t.Fatal("deploy/README.md must document what to do after running docker-deploy.sh")
	}
	end := strings.Index(readme[start:], "### Method 2:")
	if end == -1 {
		t.Fatal("deploy/README.md must keep Method 2 marker after one-click deployment instructions")
	}
	section := readme[start : start+end]
	if strings.Contains(section, "docker-compose.local.yml") {
		t.Fatal("docker-deploy.sh writes docker-compose.yml, so one-click README steps must not reference docker-compose.local.yml")
	}
	if !strings.Contains(section, "docker compose up -d") {
		t.Fatal("one-click README steps must start the generated docker-compose.yml without an explicit -f file")
	}
}

func TestDockerDeployDoesNotPrintGeneratedSystemSecrets(t *testing.T) {
	script := readRepoFileForDeployContractTest(t, "deploy", "docker-deploy.sh")
	for lineNo, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		isOutput := strings.HasPrefix(trimmed, "echo ") || strings.HasPrefix(trimmed, "print_")
		if !isOutput {
			continue
		}
		for _, secret := range []string{"POSTGRES_PASSWORD", "JWT_SECRET", "TOTP_ENCRYPTION_KEY"} {
			if strings.Contains(trimmed, "${"+secret+"}") {
				t.Fatalf("deploy/docker-deploy.sh prints generated system secret %s on line %d; write it to .env without echoing it", secret, lineNo+1)
			}
		}
	}

	readme := readRepoFileForDeployContractTest(t, "deploy", "README.md")
	for _, stale := range []string{
		"Displays generated credentials",
		"POSTGRES_PASSWORD, JWT_SECRET, etc.",
	} {
		if strings.Contains(readme, stale) {
			t.Fatalf("deploy/README.md still documents terminal disclosure of generated system secrets: %q", stale)
		}
	}
}

func TestDeployEnvExampleDoesNotShipReusablePostgresPassword(t *testing.T) {
	content := readRepoFileForDeployContractTest(t, "deploy", ".env.example")

	if strings.Contains(content, "POSTGRES_PASSWORD=change_this_secure_password") {
		t.Fatal("deploy/.env.example must not ship a reusable POSTGRES_PASSWORD that satisfies compose required-variable checks")
	}
	if !strings.Contains(content, "POSTGRES_PASSWORD=") {
		t.Fatal("deploy/.env.example must keep documenting POSTGRES_PASSWORD for manual deployments")
	}

	script := readRepoFileForDeployContractTest(t, "deploy", "docker-deploy.sh")
	if !strings.Contains(script, "POSTGRES_PASSWORD=$(generate_secret)") ||
		!strings.Contains(script, "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=${POSTGRES_PASSWORD}/") {
		t.Fatal("deploy/docker-deploy.sh must continue generating POSTGRES_PASSWORD for one-click deployments")
	}
}

func TestDockerHubDocsDoNotRecommendReusableDatabasePasswords(t *testing.T) {
	docs := readRepoFileForDeployContractTest(t, "deploy", "DOCKER.md")
	for _, reusable := range []string{
		`DATABASE_PASSWORD="change_this_secure_password"`,
		"DATABASE_PASSWORD=postgres",
		"POSTGRES_PASSWORD=postgres",
	} {
		if strings.Contains(docs, reusable) {
			t.Fatalf("deploy/DOCKER.md must not recommend reusable database password example %q", reusable)
		}
	}
	for _, required := range []string{
		"DATABASE_PASSWORD=\"${DATABASE_PASSWORD}\"",
		"POSTGRES_PASSWORD=${DATABASE_PASSWORD}",
		"DATABASE_PASSWORD=$(openssl rand -hex 32)",
	} {
		if !strings.Contains(docs, required) {
			t.Fatalf("deploy/DOCKER.md must show generated database password usage %q", required)
		}
	}
}

func TestDeployComposeCommentsUseSocialOpsRuntimeTerms(t *testing.T) {
	for _, deployFile := range []string{
		".env.example",
		"docker-compose.yml",
		"docker-compose.local.yml",
		"docker-compose.standalone.yml",
		"docker-compose.dev.yml",
	} {
		content := readRepoFileForDeployContractTest(t, "deploy", deployFile)
		for _, staleWording := range []string{
			"upstream/pricing/CRS",
			"pricing data",
			"定价数据",
		} {
			if strings.Contains(content, staleWording) {
				t.Fatalf("deploy/%s still contains stale deployment wording %q", deployFile, staleWording)
			}
		}
	}
}

func TestSystemdDeployUsesCurrentServerModeEnv(t *testing.T) {
	for _, pathParts := range [][]string{
		{"deploy", "socialops.service"},
		{"deploy", "install.sh"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)

		if strings.Contains(content, "Environment=GIN_MODE") {
			t.Fatalf("%s still writes stale GIN_MODE instead of SERVER_MODE", name)
		}
		if !strings.Contains(content, "Environment=SERVER_MODE=release") {
			t.Fatalf("%s must set current SERVER_MODE for release deployments", name)
		}
	}
}

func TestSystemdDeployUsesCurrentProductDescription(t *testing.T) {
	const current = "SocialOps - Website Account Pool Social Operations Platform"
	const stale = "SocialOps - Social Account Rental & Task Distribution Platform"

	for _, pathParts := range [][]string{
		{"deploy", "socialops.service"},
		{"deploy", "install.sh"},
	} {
		content := readRepoFileForDeployContractTest(t, pathParts...)
		name := filepath.Join(pathParts...)

		if strings.Contains(content, stale) {
			t.Fatalf("%s still uses stale deployment description %q", name, stale)
		}
		if !strings.Contains(content, current) {
			t.Fatalf("%s must use current deployment description %q", name, current)
		}
	}
}
