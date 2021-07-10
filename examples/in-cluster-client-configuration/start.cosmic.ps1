# Copyright (c) Microsoft Corporation. All rights reserved.
# THIS CODE IS MADE AVAILABLE AS IS, WITHOUT WARRANTY OF ANY KIND. THE ENTIRE RISK
# OF THE USE OR THE RESULTS FROM THE USE OF THIS CODE REMAINS WITH THE USER.

Set-StrictMode -Version 'Latest'
$ErrorActionPreference = 'Stop'
$VerbosePreference = 'Continue'
$WarningPreference = 'Continue'

$JitWatsonLocation = $PSScriptRoot

# Defaults to ensure the feature is disabled by default
$CrashHandlerEnabled = $false
$DumpDatacenterRegion = $null
$DumpMdsMaNamespace = $null
$DumpStorageLocation = $null
#$SynchronousUploadString = "False"
$SynchronousUploadString = "True"

# Load the JitWatson configuration helper module
$configModulePath = Join-Path $JitWatsonLocation 'JitWatsonConfig.psm1'
Write-Verbose $configModulePath
Import-Module $configModulePath

Write-Verbose "Setting the Location parameter to $JitWatsonLocation"
Set-JitWatsonLocation -Path $JitWatsonLocation

# Define the default dump storage location
$dataDir = $env:DATADIR
if ($dataDir -and (Test-Path $dataDir)) {
    $DumpStorageLocation = Join-Path $dataDir "JitWatsonDumps"
} else {
    $DumpStorageLocation = Join-Path (Split-Path -Parent $JitWatsonLocation) "JitWatsonDumps"
}

Write-Verbose "Setting default dump location to $DumpStorageLocation"

# Default environment name
$environmentName = "Default"

# Default to using Distributed.
# The other options available are LocalMachine and Disabled
$ThrottlingType = "Distributed"

# Find the default cosmic values.
$userSettings = Join-Path $PSScriptRoot 'config.cosmic.ini'
if (Test-Path $userSettings) {
    $PSDefaultParameterValues = @{
        'Get-IniPropertyValue:Path'        = $userSettings
        'Get-IniPropertyValue:SectionName' = 'JitWatson'
    }

    # Load the service ini helper script
    . (Join-Path $JitWatsonLocation 'Get-IniPropertyValue.ps1')

    $CrashHandlerEnabled = Get-IniPropertyValue -PropertyName 'JitWatsonCrashHandlerEnabled' -DefaultValue $CrashHandlerEnabled
    $DumpDatacenterRegion = Get-IniPropertyValue -PropertyName 'JitWatsonDumpDatacenterRegion' -DefaultValue $DumpDatacenterRegion
    $DumpMdsMaNamespace = Get-IniPropertyValue -PropertyName 'JitWatsonDumpMdsMaNamespace' -DefaultValue $DumpMdsMaNamespace
    $DumpStorageLocation = Get-IniPropertyValue -PropertyName 'JitWatsonDumpStorageLocation' -DefaultValue $DumpStorageLocation
    $environmentName = Get-IniPropertyValue -PropertyName 'JitWatsonEnvironmentName' -DefaultValue $environmentName
    $ThrottlingType = Get-IniPropertyValue -PropertyName 'JitWatsonThrottlingType' -DefaultValue $ThrottlingType
    $SynchronousUploadString = Get-IniPropertyValue -PropertyName 'JitWatsonSynchronousUpload' -DefaultValue $SynchronousUploadString
}

# Enforce the valid set of options for the throttling type
if (-not ($ThrottlingType -eq "LocalMachine" -or $ThrottlingType -eq "Disabled")){
    Write-Verbose "Invalid ThrottlingType parameter ($ThrottlingType) reset to Distributed"
    $ThrottlingType = "Distributed"
}

$SynchronousUpload = ("True" -eq $SynchronousUploadString)

Write-Verbose "CrashHandlerEnabled ($CrashHandlerEnabled)"
Write-Verbose "DumpDatacenterRegion ($DumpDatacenterRegion)"
Write-Verbose "DumpMdsMaNamespace ($DumpMdsMaNamespace)"
Write-Verbose "DumpStorageLocation ($DumpStorageLocation)"
Write-Verbose "EnvironmentName ($environmentName)"
Write-Verbose "ThrottlingType ($ThrottlingType)"
Write-Verbose "SynchronousUpload ($SynchronousUpload)"

# Check the settings and disable JitWatson if dump region, MDS namespace, or enabled settings are not defined.
[bool]$enabled = `
    (-not (`
        [string]::IsNullOrEmpty($DumpDatacenterRegion) `
        -or [string]::IsNullOrEmpty($DumpMdsMaNamespace) `
        -or [string]::IsNullOrEmpty($CrashHandlerEnabled) `
        -or [string]::IsNullOrEmpty($DumpStorageLocation)))

# If configuration looks valid then evaluate CrashHandlerEnabled and set enabled to false if the string value was not valid.
$enabled = $enabled -and ("True" -eq $CrashHandlerEnabled)
Write-Verbose "Enabled evaluated to ($enabled) after validating settings"

$handlerBinaryName = "JitExWatson.exe"
$handlerBinaryPath = Join-Path $PSScriptRoot $handlerBinaryName

if ($enabled) {
    $handlerParameters = "%ld %ld %p true"
    $CollectionScriptPath = Join-Path $PSScriptRoot "Dump-CrashReportingProcess.ps1"
    $ExcludedBinaries = @("lsass", "winlogon", "logonui")

    # Create the dump directory
    if (-Not (Test-Path $DumpStorageLocation))
    {
        New-Item -ItemType Directory -Path $DumpStorageLocation -Force | Out-Null
    }

    Enable-JitWatson `
        -MdsMaNamespace $DumpMdsMaNamespace `
        -DumpDatacenterRegion $DumpDatacenterRegion `
        -HandlerBinaryPath $handlerBinaryPath `
        -HandlerParameters $handlerParameters `
        -CollectionScriptPath $CollectionScriptPath `
        -DumpStorageLocation $DumpStorageLocation `
        -EnvironmentName $environmentName `
        -ThrottlingType $ThrottlingType `
        -SynchronousUpload $SynchronousUpload `
        -ExcludedBinaries $ExcludedBinaries

    # Terminate any running instances of AzureWatsonAgent.exe
    Stop-Process -Name "AzureWatsonAgent" -Force -ErrorAction SilentlyContinue
} else {
    $aeDebugHandler = $null
    try {
        $aeDebugHandler = (Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\AeDebug").Debugger
    }
    catch {
    }

    if ($aeDebugHandler -and $aeDebugHandler -like ("*{0}*" -f $HandlerBinaryName)) {
        # Disable the handler
        Disable-JitWatson
    }
}