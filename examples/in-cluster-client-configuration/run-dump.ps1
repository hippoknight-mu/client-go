param(
    $ProcID = "",
    $ProcName = ""
    )

if ($ProcID -eq "") {
        $ProcID = (Get-Process | Where-Object {$_.Name -eq "$ProcName"} | Select -Index 0).Id.ToString()
        if ($ProcID -eq "") {
                throw "Process($ProcName) not found."
        }
        write-host "Found PID($ProcID) by ProcName($ProcName)"
}

write-host "================ Start Watson Dump, PID: $ProcID ================="

./JitWatson/start.cosmic.ps1
# ./JitWatson/Dump-CrashReportingProcess.ps1 -UniquePid $ProcID  | Tee-Object -file dir.txt
./JitWatson/Dump-CrashReportingProcess.ps1 -UniquePid $ProcID > log.txt

write-host "=================================================================="