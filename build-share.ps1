# Script to build the Cloud Client and update the shared build directory (build-share)

Write-Host "Starting build..." -ForegroundColor Cyan

# 1. Run wails build
wails build
if ($LASTEXITCODE -ne 0) {
    Write-Error "Wails build failed!"
    Exit $LASTEXITCODE
}

# 2. Check if the output file exists
$srcPath = "build/bin/cloud-client.exe"
$destFolder = "build-share"
$destPath = "$destFolder/cloud-client.exe"

if (-not (Test-Path $srcPath)) {
    Write-Error "Could not find built executable at $srcPath"
    Exit 1
}

# 3. Create build-share folder if it doesn't exist (e.g. if deleted)
if (-not (Test-Path $destFolder)) {
    Write-Host "Creating directory $destFolder..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $destFolder -Force | Out-Null
}

# 4. Copy the executable
Write-Host "Copying $srcPath to $destPath..." -ForegroundColor Yellow
Copy-Item -Path $srcPath -Destination $destPath -Force

# 5. Copy assets
if (Test-Path "assets") {
    Write-Host "Syncing assets to $destFolder/assets..." -ForegroundColor Yellow
    Copy-Item -Path "assets" -Destination "$destFolder" -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "Build completed and copied to build-share successfully!" -ForegroundColor Green
