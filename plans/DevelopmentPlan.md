$START_DOC_NAME

**PURPOSE:** Реализовать production-ready `pgkernel` v0.1.0 как однобинарный Go CLI с проверками Linux kernel + PostgreSQL, пригодный для локального запуска и CI.
**SCOPE:** Архитектура приложения, правила проверки, policy-режим, форматы вывода, тестирование и документы для open-source запуска.
**KEYWORDS:** DOMAIN(PostgreSQL): config audit; DOMAIN(Linux): kernel/sysctl/filesystem; CONCEPT(CLI): policy-first UX; TECH(Go): single static binary.

$START_DOCUMENT_PLAN
### Document Plan

**SECTION_GOALS:**
- GOAL [Спроектировать минимально-зависимую, расширяемую архитектуру модулей проверки] => [GOAL_ARCH]
- GOAL [Обеспечить machine-readable контракт через стабильный JSON schema] => [GOAL_JSON]
- GOAL [Сделать CI-ready политику через fail-on/baseline/compare] => [GOAL_POLICY]

**SECTION_USE_CASES:**
- USE_CASE [SRE запускает pre-upgrade диагностику и получает actionable remediation] => [UC_PREUPGRADE]
- USE_CASE [CI pipeline валидирует infra drift и блокирует регрессии] => [UC_CI]
- USE_CASE [DBA делится markdown-отчетом в GitHub/Slack] => [UC_SHARE]

$END_DOCUMENT_PLAN

$START_SECTION_DEVELOPMENT_PLAN
### Development Plan

$START_DEV_PLAN

**PURPOSE:** Построить конкурентный open-source инструмент, который за <=2s выдает доверяемую диагностику kernel+postgres и готовый remediation plan.

---

### 1. Draft Code Graph

```xml
<DraftCodeGraph>
  <cmd_pgkernel_main_go FILE="cmd/pgkernel/main.go" TYPE="ENTRY_POINT">
    <annotation>CLI parsing and orchestration of detect/check/policy/output.</annotation>
    <cmd_pgkernel_runCheck_FUNCTION NAME="runCheck" TYPE="IS_FUNCTION_OF_MODULE">
      <annotation>Build runtime context, execute checks, apply policy, print report.</annotation>
      <CrossLinks>
        <Link TARGET="internal_detect_collectstate_FUNCTION" TYPE="CALLS_FUNCTION" />
        <Link TARGET="internal_checker_runner_run_FUNCTION" TYPE="CALLS_FUNCTION" />
        <Link TARGET="internal_policy_applyfilter_FUNCTION" TYPE="CALLS_FUNCTION" />
        <Link TARGET="internal_output_render_FUNCTION" TYPE="CALLS_FUNCTION" />
      </CrossLinks>
    </cmd_pgkernel_runCheck_FUNCTION>
  </cmd_pgkernel_main_go>

  <internal_checker_result_go FILE="internal/checker/result.go" TYPE="DOMAIN_MODEL">
    <annotation>Stable report schema entities and compatibility metadata.</annotation>
  </internal_checker_result_go>

  <internal_checker_checker_go FILE="internal/checker/checker.go" TYPE="RUNNER">
    <annotation>Check interface, execution engine, summary counters.</annotation>
    <internal_checker_runner_run_FUNCTION NAME="Run" TYPE="IS_METHOD_OF_CLASS">
      <annotation>Runs checks and computes normalized summary.</annotation>
    </internal_checker_runner_run_FUNCTION>
  </internal_checker_checker_go>

  <internal_detect_kernel_go FILE="internal/detect/kernel.go" TYPE="DETECTOR">
    <annotation>Kernel version and preemption model detection with fallbacks.</annotation>
    <internal_detect_collectstate_FUNCTION NAME="CollectKernelState" TYPE="IS_FUNCTION_OF_MODULE">
      <annotation>Reads debugfs, uname and boot config hints.</annotation>
    </internal_detect_collectstate_FUNCTION>
  </internal_detect_kernel_go>

  <internal_detect_system_go FILE="internal/detect/system.go" TYPE="DETECTOR">
    <annotation>RAM, CPU, distro and sysctl state extraction.</annotation>
  </internal_detect_system_go>

  <internal_detect_postgres_go FILE="internal/detect/postgres.go" TYPE="DETECTOR">
    <annotation>postgresql.conf discovery and parser.</annotation>
  </internal_detect_postgres_go>

  <internal_detect_profile_go FILE="internal/detect/profile.go" TYPE="DETECTOR">
    <annotation>Runtime environment profile auto-detection.</annotation>
  </internal_detect_profile_go>

  <internal_checks_kern_preemption_go FILE="internal/checks/kern_preemption.go" TYPE="CHECK_MODULE">
    <annotation>KERN-001/002/003 evaluation logic.</annotation>
  </internal_checks_kern_preemption_go>

  <internal_checks_mem_hugepages_go FILE="internal/checks/mem_hugepages.go" TYPE="CHECK_MODULE">
    <annotation>MEM-001/002 evaluation logic.</annotation>
  </internal_checks_mem_hugepages_go>

  <internal_checks_mem_swap_go FILE="internal/checks/mem_swap.go" TYPE="CHECK_MODULE">
    <annotation>MEM-003/004/005 evaluation logic.</annotation>
  </internal_checks_mem_swap_go>

  <internal_checks_io_scheduler_go FILE="internal/checks/io_scheduler.go" TYPE="CHECK_MODULE">
    <annotation>IO-001/002 evaluation logic.</annotation>
  </internal_checks_io_scheduler_go>

  <internal_checks_net_tcp_go FILE="internal/checks/net_tcp.go" TYPE="CHECK_MODULE">
    <annotation>NET-001 evaluation logic.</annotation>
  </internal_checks_net_tcp_go>

  <internal_checks_pg_config_go FILE="internal/checks/pg_config.go" TYPE="CHECK_MODULE">
    <annotation>PG-001/002/003 evaluation logic.</annotation>
  </internal_checks_pg_config_go>

  <internal_policy_filter_go FILE="internal/policy/filter.go" TYPE="POLICY_MODULE">
    <annotation>--only/--exclude rule application by ID and category.</annotation>
    <internal_policy_applyfilter_FUNCTION NAME="ApplyFilter" TYPE="IS_FUNCTION_OF_MODULE">
      <annotation>Returns selected checks according to include/exclude constraints.</annotation>
    </internal_policy_applyfilter_FUNCTION>
  </internal_policy_filter_go>

  <internal_policy_fail_go FILE="internal/policy/fail.go" TYPE="POLICY_MODULE">
    <annotation>Exit-code precedence and fail-on threshold logic.</annotation>
  </internal_policy_fail_go>

  <internal_policy_baseline_go FILE="internal/policy/baseline.go" TYPE="POLICY_MODULE">
    <annotation>Regression detection against baseline/compare report.</annotation>
  </internal_policy_baseline_go>

  <internal_output_pretty_go FILE="internal/output/pretty.go" TYPE="RENDERER">
    <annotation>Human-first colored output with grouped categories.</annotation>
    <CrossLinks>
      <Link TARGET="internal_output_share_go" TYPE="CALLS_MODULE" />
    </CrossLinks>
  </internal_output_pretty_go>

  <internal_output_json_go FILE="internal/output/json.go" TYPE="RENDERER">
    <annotation>Stable JSON serialization with schema_version.</annotation>
  </internal_output_json_go>

  <internal_output_markdown_go FILE="internal/output/markdown.go" TYPE="RENDERER">
    <annotation>Markdown report + optional share block.</annotation>
  </internal_output_markdown_go>

  <internal_output_share_go FILE="internal/output/share.go" TYPE="RENDERER">
    <annotation>Top-risks snippet optimized for GitHub/Slack virality.</annotation>
  </internal_output_share_go>
</DraftCodeGraph>
```

---

### 2. Step-by-step Data Flow

1. **Step 1:** CLI `check` command validates flags and prepares run options (format/profile/fail policy/filter controls).
2. **Step 2:** Detect layer reads system facts (`/proc`, `/sys`, uname, os-release), postgres facts (config discovery + parser), and runtime profile.
3. **Step 3:** Runner executes all checks over normalized `RuntimeState`; each check emits structured `CheckResult` with confidence, impact, remediation safety.
4. **Step 4:** Policy layer applies `--only/--exclude` filtering, computes regressions from baseline/compare reports, and derives final exit code with precedence rules.
5. **Step 5:** Output layer renders `pretty|json|markdown`; optional `--share` adds concise “top risks + fixes” block.
6. **Step 6:** In `--quiet`, tool suppresses full report and prints machine-friendly non-zero summary to stderr while preserving deterministic exit code.

---

### 3. Acceptance Criteria

- [ ] **Criterion 1:** Команда `pgkernel check` компилируется и выполняется без panic на Linux/macOS (kernel checks on non-Linux gracefully degrade to `skip/info`).
- [ ] **Criterion 2:** Реализованы все 13 check IDs из спецификации с expected/current/fix/reference/impact/confidence/safety.
- [ ] **Criterion 3:** Работают форматы `pretty`, `json`, `markdown`; JSON содержит `schema_version`, `summary`, `compatibility`.
- [ ] **Criterion 4:** Реализованы `--only`, `--exclude`, `--fail-on`, `--baseline`, `--compare-with`, `--quiet`.
- [ ] **Criterion 5:** Exit code приоритизация соответствует спецификации (3 > 2 > 1 > 0).
- [ ] **Criterion 6:** Есть тесты на критические policy/check rules и они проходят через `go test ./...`.
- [ ] **Criterion 7:** README и `docs/checks/*.md` содержат материалы для конкурентной дифференциации и вирусного распространения.

$END_DEV_PLAN

$END_SECTION_DEVELOPMENT_PLAN

$END_DOC_NAME
