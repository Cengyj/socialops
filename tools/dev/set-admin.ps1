param(
  [string]$Email = "3081794680@qq.com",
  [string]$Password = "668435li",
  [string]$PostgresUser = "postgres",
  [string]$PostgresPassword = "postgres",
  [string]$DatabaseName = "socialops",
  [int]$PostgresPort = 5432
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$psql = Get-Command psql.exe -ErrorAction SilentlyContinue
if (-not $psql) {
  $psqlPath = Get-ChildItem -Path "C:\Program Files\PostgreSQL\*\bin\psql.exe" -ErrorAction SilentlyContinue |
    Sort-Object FullName -Descending |
    Select-Object -First 1
  if ($psqlPath) {
    $psql = @{ Source = $psqlPath.FullName }
  }
}
if (-not $psql) {
  throw "psql.exe not found. Install PostgreSQL or add psql.exe to PATH."
}

$backendDir = Join-Path $repoRoot "backend"
Push-Location $backendDir
try {
  $hash = (& go run (Join-Path $repoRoot "tools\dev\hash-password.go") $Password).Trim()
  if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($hash)) {
    throw "failed to generate bcrypt password hash"
  }
} finally {
  Pop-Location
}

$sql = @"
BEGIN;

WITH target AS (
  SELECT id
  FROM users
  WHERE deleted_at IS NULL
    AND (role = 'admin' OR lower(email) = lower(:'admin_email'))
  ORDER BY CASE WHEN role = 'admin' THEN 0 ELSE 1 END, id
  LIMIT 1
),
updated AS (
  UPDATE users
  SET email = :'admin_email',
      password_hash = :'password_hash',
      role = 'admin',
      status = 'active',
      username = CASE WHEN username = '' THEN :'admin_email' ELSE username END,
      signup_source = 'email',
      token_version = token_version + 1,
      updated_at = NOW()
  WHERE id IN (SELECT id FROM target)
  RETURNING id
),
inserted AS (
  INSERT INTO users (
    email,
    password_hash,
    role,
    balance,
    concurrency,
    status,
    username,
    signup_source,
    created_at,
    updated_at
  )
  SELECT
    :'admin_email',
    :'password_hash',
    'admin',
    0,
    30,
    'active',
    :'admin_email',
    'email',
    NOW(),
    NOW()
  WHERE NOT EXISTS (SELECT 1 FROM updated)
  RETURNING id
),
admin_user AS (
  SELECT id FROM updated
  UNION ALL
  SELECT id FROM inserted
)
INSERT INTO auth_identities (
  user_id,
  provider_type,
  provider_key,
  provider_subject,
  verified_at,
  metadata,
  created_at,
  updated_at
)
SELECT
  id,
  'email',
  'email',
  lower(:'admin_email'),
  NOW(),
  jsonb_build_object('source', 'dev_set_admin'),
  NOW(),
  NOW()
FROM admin_user
ON CONFLICT (provider_type, provider_key, provider_subject)
DO UPDATE SET
  user_id = EXCLUDED.user_id,
  verified_at = COALESCE(auth_identities.verified_at, EXCLUDED.verified_at),
  metadata = auth_identities.metadata || EXCLUDED.metadata,
  updated_at = NOW();

COMMIT;

SELECT id, email, role, status, concurrency, token_version
FROM users
WHERE deleted_at IS NULL AND lower(email) = lower(:'admin_email');
"@

$tmp = New-TemporaryFile
try {
  Set-Content -Path $tmp -Value $sql -Encoding UTF8
  $env:PGPASSWORD = $PostgresPassword
  & $psql.Source `
    -h 127.0.0.1 `
    -p $PostgresPort `
    -U $PostgresUser `
    -d $DatabaseName `
    -v "admin_email=$Email" `
    -v "password_hash=$hash" `
    -f $tmp
  if ($LASTEXITCODE -ne 0) {
    throw "psql failed"
  }
} finally {
  Remove-Item Env:\PGPASSWORD -ErrorAction SilentlyContinue
  Remove-Item $tmp -ErrorAction SilentlyContinue
}
