param(
  [switch]$Install,
  [switch]$StartServices,
  [switch]$InitDatabase,
  [string]$PostgresUser = "postgres",
  [string]$PostgresPassword = "postgres",
  [string]$DatabaseName = "socialops",
  [int]$PostgresPort = 5432,
  [int]$RedisPort = 6379
)

$ErrorActionPreference = "Stop"

function Find-Exe {
  param(
    [string]$Name,
    [string[]]$ExtraGlobs = @()
  )

  $cmd = Get-Command $Name -ErrorAction SilentlyContinue
  if ($cmd) {
    return $cmd.Source
  }

  foreach ($glob in $ExtraGlobs) {
    $match = Get-ChildItem -Path $glob -ErrorAction SilentlyContinue |
      Sort-Object FullName -Descending |
      Select-Object -First 1
    if ($match) {
      return $match.FullName
    }
  }

  return $null
}

function Test-PortOpen {
  param([int]$Port)

  $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  return $null -ne $conn
}

function Show-Status {
  param([string]$Name, [bool]$Ok, [string]$Detail)

  $mark = if ($Ok) { "OK" } else { "MISS" }
  Write-Host ("[{0}] {1} - {2}" -f $mark, $Name, $Detail)
}

function Find-PostgresServices {
  Get-Service -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -match "^postgresql" -or $_.DisplayName -match "PostgreSQL" }
}

function Find-RedisServices {
  Get-Service -ErrorAction SilentlyContinue |
    Where-Object {
      $_.Name -match "redis|memurai|valkey" -or
      $_.DisplayName -match "Redis|Memurai|Valkey"
    }
}

if ($Install) {
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw "winget is not available. Install PostgreSQL and Memurai manually, then rerun this script."
  }

  Write-Host "Installing PostgreSQL 18..."
  winget install --id PostgreSQL.PostgreSQL.18 -e --accept-package-agreements --accept-source-agreements

  Write-Host "Installing Memurai Developer (Redis-compatible)..."
  winget install --id Memurai.MemuraiDeveloper -e --accept-package-agreements --accept-source-agreements
}

$psql = Find-Exe "psql.exe" @(
  "C:\Program Files\PostgreSQL\*\bin\psql.exe"
)
$createdb = Find-Exe "createdb.exe" @(
  "C:\Program Files\PostgreSQL\*\bin\createdb.exe"
)
$redisCli = Find-Exe "redis-cli.exe" @(
  "C:\Program Files\Memurai\*\redis-cli.exe",
  "C:\Program Files\Redis\redis-cli.exe"
)

$postgresServices = @(Find-PostgresServices)
$redisServices = @(Find-RedisServices)

if ($StartServices) {
  foreach ($svc in $postgresServices + $redisServices) {
    if ($svc.Status -ne "Running") {
      Write-Host "Starting service $($svc.Name)..."
      Start-Service -Name $svc.Name
    }
  }
}

$postgresPortOpen = Test-PortOpen $PostgresPort
$redisPortOpen = Test-PortOpen $RedisPort

Show-Status "psql" ($null -ne $psql) ($(if ($psql) { $psql } else { "install PostgreSQL 18" }))
Show-Status "createdb" ($null -ne $createdb) ($(if ($createdb) { $createdb } else { "install PostgreSQL 18" }))
Show-Status "PostgreSQL service" ($postgresServices.Count -gt 0) ($(if ($postgresServices.Count -gt 0) { ($postgresServices | ForEach-Object { "$($_.Name):$($_.Status)" }) -join ", " } else { "not found" }))
Show-Status "PostgreSQL port $PostgresPort" $postgresPortOpen ($(if ($postgresPortOpen) { "listening" } else { "not listening" }))
Show-Status "Redis/Memurai service" ($redisServices.Count -gt 0) ($(if ($redisServices.Count -gt 0) { ($redisServices | ForEach-Object { "$($_.Name):$($_.Status)" }) -join ", " } else { "not found" }))
Show-Status "Redis port $RedisPort" $redisPortOpen ($(if ($redisPortOpen) { "listening" } else { "not listening" }))

if ($InitDatabase) {
  if (-not $psql -or -not $createdb) {
    throw "PostgreSQL client tools are missing. Install PostgreSQL first."
  }
  if (-not $postgresPortOpen) {
    throw "PostgreSQL is not listening on port $PostgresPort. Start the PostgreSQL service first."
  }

  $env:PGPASSWORD = $PostgresPassword
  try {
    $exists = & $psql -h 127.0.0.1 -p $PostgresPort -U $PostgresUser -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DatabaseName'"
    if ($LASTEXITCODE -ne 0) {
      throw "psql failed. Check the postgres user password. This project expects password '$PostgresPassword'."
    }
    if (($exists | Out-String).Trim() -eq "1") {
      Write-Host "Database '$DatabaseName' already exists."
    } else {
      & $createdb -h 127.0.0.1 -p $PostgresPort -U $PostgresUser $DatabaseName
      if ($LASTEXITCODE -ne 0) {
        throw "createdb failed."
      }
      Write-Host "Database '$DatabaseName' created."
    }
  } finally {
    Remove-Item Env:\PGPASSWORD -ErrorAction SilentlyContinue
  }
}

Write-Host ""
Write-Host "Native dependency target for GoLand:"
Write-Host "  PostgreSQL: 127.0.0.1:$PostgresPort user=$PostgresUser password=$PostgresPassword db=$DatabaseName"
Write-Host "  Redis:      127.0.0.1:$RedisPort"
Write-Host ""
Write-Host "Common first-run commands:"
Write-Host "  powershell -ExecutionPolicy Bypass -File tools/dev/native-deps.ps1 -Install"
Write-Host "  powershell -ExecutionPolicy Bypass -File tools/dev/native-deps.ps1 -StartServices -InitDatabase"
