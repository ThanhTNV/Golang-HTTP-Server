$CertDir = Join-Path $PSScriptRoot "cert"
$KeyPath = Join-Path $CertDir "key.pem"
$CertPath = Join-Path $CertDir "cert.pem"
$SanConfig = Join-Path $PSScriptRoot "san.cnf"
$SecretName = "tls-secret"
$Namespace = "default"

if (!(Test-Path $CertDir)) {
    New-Item -ItemType Directory -Path $CertDir | Out-Null
    Write-Host "Created directory: $CertDir"
}

Write-Host "Generating self-signed certificate..."
openssl req -x509 -newkey rsa:4096 `
    -keyout $KeyPath `
    -out $CertPath `
    -days 2650 `
    -nodes `
    -config $SanConfig

if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to generate certificate"
    exit 1
}

Write-Host "Certificate generated successfully"
Write-Host "  cert: $CertPath"
Write-Host "  key:  $KeyPath"

$existing = kubectl get secret $SecretName -n $Namespace 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "Secret '$SecretName' exists, updating..."
    kubectl create secret tls $SecretName `
        --cert=$CertPath `
        --key=$KeyPath `
        -n $Namespace `
        --dry-run=client -o yaml | kubectl apply -f -
} else {
    Write-Host "Creating secret '$SecretName'..."
    kubectl create secret tls $SecretName `
        --cert=$CertPath `
        --key=$KeyPath `
        -n $Namespace
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "Secret '$SecretName' applied successfully"
} else {
    Write-Error "Failed to apply secret '$SecretName'"
    exit 1
}
