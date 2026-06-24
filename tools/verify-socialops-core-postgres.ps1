param(
  [string]$Phase = "phase77",
  [string]$BackendDir = "backend",
  [string]$OutputPath = "output/phase77-socialops-core-postgres-readiness.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Test-CommandExists {
  param([string]$Name)

  $cmd = Get-Command $Name -ErrorAction SilentlyContinue
  if ($null -eq $cmd) {
    return @{
      available = $false
      source = $null
    }
  }

  return @{
    available = $true
    source = $cmd.Source
  }
}

function Test-LocalPort {
  param([int]$Port)

  try {
    $listening = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($null -ne $listening) {
      return $true
    }

    return [bool](Test-NetConnection -ComputerName 127.0.0.1 -Port $Port -InformationLevel Quiet -WarningAction SilentlyContinue)
  } catch {
    return $false
  }
}

function Get-EnvPresence {
  param([string]$Name)

  $value = [Environment]::GetEnvironmentVariable($Name)
  return @{
    name = $Name
    present = -not [string]::IsNullOrWhiteSpace($value)
  }
}

function Read-HostDevPort {
  param(
    [string]$Name,
    [int]$Default
  )

  $envPath = Join-Path (Get-Location) "deploy/.env.host-dev"
  if (-not (Test-Path -LiteralPath $envPath)) {
    return $Default
  }

  $line = Get-Content -LiteralPath $envPath |
    Where-Object { $_ -match ("^\s*" + [regex]::Escape($Name) + "\s*=") } |
    Select-Object -First 1
  if ([string]::IsNullOrWhiteSpace($line)) {
    return $Default
  }

  $raw = ($line -split "=", 2)[1].Trim()
  $parsed = 0
  if ([int]::TryParse($raw, [ref]$parsed)) {
    return $parsed
  }

  return $Default
}

$startedAt = (Get-Date).ToUniversalTime().ToString("o")
$backendPath = Resolve-Path -LiteralPath $BackendDir
$postgresDSN = [Environment]::GetEnvironmentVariable("SOCIALOPS_INTEGRATION_POSTGRES_DSN")
$postgresDSNTarget = $null
if (-not [string]::IsNullOrWhiteSpace($postgresDSN)) {
  try {
    if ($postgresDSN -match "^\s*postgres(ql)?://") {
      $parsedDSN = [uri]$postgresDSN
      $dsnPort = $parsedDSN.Port
      if ($dsnPort -lt 0) {
        $dsnPort = 5432
      }
      $postgresDSNTarget = @{
        host = $parsedDSN.Host
        port = $dsnPort
        database = $parsedDSN.AbsolutePath.TrimStart("/")
      }
    } else {
      $dsnFields = @{}
      foreach ($match in [regex]::Matches($postgresDSN, "(\w+)=('[^']*'|""[^""]*""|\S+)")) {
        $value = $match.Groups[2].Value.Trim()
        if (($value.StartsWith("'") -and $value.EndsWith("'")) -or ($value.StartsWith('"') -and $value.EndsWith('"'))) {
          $value = $value.Substring(1, $value.Length - 2)
        }
        $dsnFields[$match.Groups[1].Value.ToLowerInvariant()] = $value
      }
      $dsnPort = 5432
      if ($dsnFields.Contains("port")) {
        [void][int]::TryParse([string]$dsnFields["port"], [ref]$dsnPort)
      }
      $postgresDSNTarget = @{
        host = if ($dsnFields.Contains("host")) { $dsnFields["host"] } else { "localhost" }
        port = $dsnPort
        database = if ($dsnFields.Contains("dbname")) { $dsnFields["dbname"] } else { $null }
      }
    }
  } catch {
    $postgresDSNTarget = @{
      parse_error = $_.Exception.Message
    }
  }
}
$postgresHostDevPort = Read-HostDevPort -Name "POSTGRES_HOST_PORT" -Default 5433
$databaseHostDevPort = Read-HostDevPort -Name "DATABASE_PORT" -Default $postgresHostDevPort
$redisHostDevPort = Read-HostDevPort -Name "REDIS_HOST_PORT" -Default 6380

$commands = [ordered]@{}
foreach ($name in @("docker", "docker-compose", "podman", "psql", "postgres", "pg_ctl", "initdb", "pg_isready")) {
  $commands[$name] = Test-CommandExists -Name $name
}

$ports = [ordered]@{}
$portsToCheck = @($postgresHostDevPort, $databaseHostDevPort, 5432, $redisHostDevPort, 6379)
if ($null -ne $postgresDSNTarget -and $postgresDSNTarget.Contains("port")) {
  $portsToCheck += [int]$postgresDSNTarget.port
}
foreach ($port in $portsToCheck | Sort-Object -Unique) {
  $ports["127.0.0.1:$port"] = Test-LocalPort -Port $port
}

$envVars = @(
  "SOCIALOPS_INTEGRATION_POSTGRES_DSN",
  "SOCIALOPS_INTEGRATION_REDIS_ADDR",
  "DATABASE_URL",
  "DATABASE_HOST",
  "DATABASE_PORT",
  "DATABASE_USER",
  "DATABASE_DBNAME",
  "PGHOST",
  "PGPORT",
  "PGUSER",
  "PGDATABASE"
) | ForEach-Object { Get-EnvPresence -Name $_ }

$testOutput = @()
$exitCode = $null
$commandText = "go test -tags=integration ./internal/repository -run TestSocialOpsCoreMigrations -count=1 -v"
Push-Location -LiteralPath $backendPath
try {
  $testOutput = & go test -tags=integration ./internal/repository -run TestSocialOpsCoreMigrations -count=1 -v 2>&1
  $exitCode = $LASTEXITCODE
} finally {
  Pop-Location
}

$joinedOutput = ($testOutput | ForEach-Object { [string]$_ }) -join "`n"
$skipped = $joinedOutput -match "skipping integration tests"
$proved = ($exitCode -eq 0) -and (-not $skipped) -and ($joinedOutput -match "(?m)^(--- PASS:|PASS\b|ok\s+)")

$missing = @()
$hasExternalDSN = ($envVars | Where-Object { $_.name -eq "SOCIALOPS_INTEGRATION_POSTGRES_DSN" }).present
$hasDockerFallback = $commands["docker"].available
$hasCommonListeningPostgres = [bool]($ports.GetEnumerator() | Where-Object { $_.Name -match "127\.0\.0\.1:(5432|5433)$" -and $_.Value })
$hasDSNListeningPostgres = $false
if ($null -ne $postgresDSNTarget -and $postgresDSNTarget.Contains("port")) {
  $dsnPortKey = "127.0.0.1:$($postgresDSNTarget.port)"
  $hasDSNListeningPostgres = $ports.Contains($dsnPortKey) -and [bool]$ports[$dsnPortKey]
}
if (-not $hasExternalDSN) {
  $missing += "SOCIALOPS_INTEGRATION_POSTGRES_DSN"
}
if (-not $hasExternalDSN -and -not $hasDockerFallback) {
  $missing += "docker"
}
if (-not $hasExternalDSN -and -not $hasCommonListeningPostgres) {
  $missing += "listening PostgreSQL port"
}
if ($hasExternalDSN -and -not $hasDSNListeningPostgres -and -not $proved) {
  $missing += "listening PostgreSQL DSN target"
}

$status = "failed"
if ($proved) {
  $status = "verified"
} elseif ($skipped) {
  $status = "not_verified_missing_postgres"
}

$result = [ordered]@{
  phase = $Phase
  checked_at = $startedAt
  status = $status
  proved_real_postgres_integration = $proved
  integration_test_skipped = $skipped
  missing_capabilities = $missing | Select-Object -Unique
  backend_dir = $backendPath.Path
  command = $commandText
  exit_code = $exitCode
  postgres_dsn_target = $postgresDSNTarget
  env = $envVars
  commands = $commands
  local_ports = $ports
  output = ($testOutput | ForEach-Object { [string]$_ })
}

$outputFile = Join-Path (Get-Location) $OutputPath
$outputDir = Split-Path -Parent $outputFile
if (-not [string]::IsNullOrWhiteSpace($outputDir)) {
  New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$json = $result | ConvertTo-Json -Depth 8
Set-Content -LiteralPath $outputFile -Value $json -Encoding UTF8
Write-Output $json
