param(
  [ValidateSet("launch", "backend", "frontend", "mock")]
  [string]$Role = "launch",
  [switch]$SkipInfra,
  [switch]$SkipBackend,
  [switch]$SkipFrontend,
  [switch]$UseMockApi,
  [switch]$InstallFrontendDeps,
  [bool]$BackendWatch = $true,
  [int]$BackendPort = 8080,
  [int]$FrontendPort = 3000,
  [int]$DatabasePort = 5432,
  [int]$RedisPort = 6379,
  [string]$DatabaseUser = "socialops",
  [string]$DatabasePassword = "socialops_dev_password",
  [string]$DatabaseName = "socialops",
  [string]$RedisPassword = "",
  [string]$AdminEmail = "3081794680@qq.com",
  [string]$AdminPassword = "668435li"
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$BackendDir = Join-Path $Root "backend"
$FrontendDir = Join-Path $Root "frontend"
$DeployDir = Join-Path $Root "deploy"
$DataDir = Join-Path $DeployDir "dev-data"
$ScriptPath = $PSCommandPath

function Test-Command {
  param([string]$Name)
  return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Quote-PS {
  param([string]$Value)
  return "'" + ($Value -replace "'", "''") + "'"
}

function Set-BackendEnvironment {
  New-Item -ItemType Directory -Path $DataDir -Force | Out-Null

  $env:DATA_DIR = $DataDir
  $env:AUTO_SETUP = "true"
  $env:SERVER_HOST = "0.0.0.0"
  $env:SERVER_PORT = [string]$BackendPort
  $env:SERVER_MODE = "debug"
  $env:RUN_MODE = "standard"
  $env:DATABASE_HOST = "127.0.0.1"
  $env:DATABASE_PORT = [string]$DatabasePort
  $env:DATABASE_USER = $DatabaseUser
  $env:DATABASE_PASSWORD = $DatabasePassword
  $env:DATABASE_DBNAME = $DatabaseName
  $env:DATABASE_SSLMODE = "disable"
  $env:REDIS_HOST = "127.0.0.1"
  $env:REDIS_PORT = [string]$RedisPort
  $env:REDIS_PASSWORD = $RedisPassword
  $env:REDIS_DB = "0"
  $env:ADMIN_EMAIL = $AdminEmail
  $env:ADMIN_PASSWORD = $AdminPassword
  $env:JWT_SECRET = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  $env:TOTP_ENCRYPTION_KEY = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  $env:TZ = "Asia/Shanghai"
}

function Get-StringHash {
  param([string]$Text)
  $sha = [System.Security.Cryptography.SHA256]::Create()
  try {
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($Text)
    $hash = $sha.ComputeHash($bytes)
    return ([BitConverter]::ToString($hash) -replace "-", "").ToLowerInvariant()
  } finally {
    $sha.Dispose()
  }
}

function Get-BackendFingerprint {
  $items = Get-ChildItem -Path $BackendDir -Recurse -File -Include "*.go", "go.mod", "go.sum" |
    Where-Object {
      $_.FullName -notmatch "\\(bin|tmp|\.cache|vendor)\\"
    } |
    Sort-Object FullName |
    ForEach-Object {
      "$($_.FullName)|$($_.Length)|$($_.LastWriteTimeUtc.Ticks)"
    }

  return Get-StringHash ($items -join "`n")
}

function Stop-ProcessTree {
  param([System.Diagnostics.Process]$Process)
  if ($null -eq $Process -or $Process.HasExited) {
    return
  }
  & taskkill.exe /PID $Process.Id /T /F | Out-Null
}

function Start-BackendProcess {
  Write-Host "[backend] starting go server on http://localhost:$BackendPort"
  return Start-Process `
    -FilePath "go" `
    -ArgumentList @("run", "./cmd/server") `
    -WorkingDirectory $BackendDir `
    -NoNewWindow `
    -PassThru
}

function Start-BackendRole {
  if (-not (Test-Command "go")) {
    throw "Go is not available on PATH."
  }

  Set-BackendEnvironment
  Set-Location $BackendDir

  if (-not $BackendWatch) {
    Write-Host "[backend] running without watcher"
    go run ./cmd/server
    return
  }

  Write-Host "[backend] watcher enabled; edits to backend/*.go restart the server"
  $fingerprint = Get-BackendFingerprint
  $proc = Start-BackendProcess

  try {
    while ($true) {
      Start-Sleep -Seconds 2
      $next = Get-BackendFingerprint
      if ($next -ne $fingerprint) {
        Write-Host "[backend] change detected; restarting"
        Stop-ProcessTree $proc
        $fingerprint = $next
        $proc = Start-BackendProcess
        continue
      }
      if ($proc.HasExited) {
        Write-Host "[backend] process exited with code $($proc.ExitCode); waiting for changes"
      }
    }
  } finally {
    Stop-ProcessTree $proc
  }
}

function Invoke-Pnpm {
  param([string[]]$Arguments)

  if (Test-Command "pnpm") {
    & pnpm @Arguments
    return
  }
  if (Test-Command "corepack") {
    & corepack pnpm @Arguments
    return
  }
  throw "pnpm is not available, and corepack is not available to provide it."
}

function Start-FrontendRole {
  if (-not (Test-Command "node")) {
    throw "Node.js is not available on PATH."
  }
  if (-not (Test-Command "pnpm") -and (Test-Command "corepack")) {
    Write-Host "[frontend] enabling pnpm through corepack"
    corepack prepare pnpm@9 --activate
  }

  $env:VITE_DEV_PROXY_TARGET = "http://localhost:$BackendPort"
  $env:VITE_DEV_PORT = [string]$FrontendPort

  if ($InstallFrontendDeps -or -not (Test-Path (Join-Path $FrontendDir "node_modules"))) {
    Write-Host "[frontend] installing dependencies"
    Invoke-Pnpm @("--dir", $FrontendDir, "install")
  }

  Write-Host "[frontend] starting Vite on http://localhost:$FrontendPort"
  Invoke-Pnpm @("--dir", $FrontendDir, "run", "dev")
}

function Start-MockRole {
  if (-not (Test-Command "node")) {
    throw "Node.js is not available on PATH."
  }
  $env:MOCK_API_PORT = [string]$BackendPort
  $env:ADMIN_EMAIL = $AdminEmail
  $env:ADMIN_PASSWORD = $AdminPassword
  Write-Host "[mock-api] starting on http://localhost:$BackendPort"
  node (Join-Path $Root "tools\mock-api.mjs")
}

function Start-Infra {
  if ($SkipInfra) {
    return $true
  }
  if (-not (Test-Command "docker")) {
    Write-Warning "Docker is not available. Start PostgreSQL and Redis yourself, or install Docker Desktop."
    Write-Warning "Expected PostgreSQL: 127.0.0.1:$DatabasePort user=$DatabaseUser db=$DatabaseName"
    Write-Warning "Expected Redis:      127.0.0.1:$RedisPort"
    return $false
  }

  $env:BIND_HOST = "127.0.0.1"
  $env:POSTGRES_USER = $DatabaseUser
  $env:POSTGRES_PASSWORD = $DatabasePassword
  $env:POSTGRES_DB = $DatabaseName
  $env:DATABASE_PORT = [string]$DatabasePort
  $env:REDIS_PORT = [string]$RedisPort
  $env:REDIS_PASSWORD = $RedisPassword
  $env:REDIS_DB = "0"
  $env:SERVER_PORT = [string]$BackendPort
  $env:ADMIN_EMAIL = $AdminEmail
  $env:ADMIN_PASSWORD = $AdminPassword
  $env:JWT_SECRET = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  $env:TOTP_ENCRYPTION_KEY = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

  Write-Host "[infra] starting PostgreSQL and Redis containers"
  docker compose -p socialops-hot -f (Join-Path $DeployDir "docker-compose.dev.yml") up -d postgres redis
  return $true
}

function Start-DevWindow {
  param([string]$Title, [string]$ChildRole)

  $args = @(
    "-NoExit",
    "-ExecutionPolicy", "Bypass",
    "-Command",
    "& { `$Host.UI.RawUI.WindowTitle = $(Quote-PS $Title); & $(Quote-PS $ScriptPath) -Role $ChildRole -SkipInfra -BackendPort $BackendPort -FrontendPort $FrontendPort -DatabasePort $DatabasePort -RedisPort $RedisPort -DatabaseUser $(Quote-PS $DatabaseUser) -DatabasePassword $(Quote-PS $DatabasePassword) -DatabaseName $(Quote-PS $DatabaseName) -RedisPassword $(Quote-PS $RedisPassword) -AdminEmail $(Quote-PS $AdminEmail) -AdminPassword $(Quote-PS $AdminPassword) -BackendWatch:`$$BackendWatch -InstallFrontendDeps:`$$InstallFrontendDeps }"
  )

  Start-Process -FilePath "powershell.exe" -ArgumentList $args -WorkingDirectory $Root | Out-Null
}

if ($Role -eq "backend") {
  Start-BackendRole
  return
}

if ($Role -eq "frontend") {
  Start-FrontendRole
  return
}

if ($Role -eq "mock") {
  Start-MockRole
  return
}

$infraReady = Start-Infra

if ($UseMockApi -and -not $SkipBackend) {
  Start-DevWindow "SocialOps mock API" "mock"
} elseif (-not $SkipBackend) {
  if (-not $infraReady) {
    Write-Warning "Backend may fail until PostgreSQL and Redis are running."
  }
  Start-DevWindow "SocialOps backend" "backend"
}

if (-not $SkipFrontend) {
  Start-DevWindow "SocialOps frontend" "frontend"
}

Write-Host ""
Write-Host "SocialOps local dev launched."
Write-Host "Frontend: http://localhost:$FrontendPort"
Write-Host "Backend:  http://localhost:$BackendPort"
Write-Host "Admin:    $AdminEmail / $AdminPassword"
Write-Host "Data dir: $DataDir"
Write-Host ""
Write-Host "Notes:"
Write-Host "- Frontend changes hot reload through Vite."
Write-Host "- Backend Go changes restart the backend watcher."
Write-Host "- If Docker is missing, install Docker Desktop or run PostgreSQL/Redis locally before using the backend."
