@echo off
SETLOCAL ENABLEDELAYEDEXPANSION

set BASEDIR=%~dp0
echo %BASEDIR%


@REM rem Clear former data
for /d %%i in ("%BASEDIR%validator-*") do (
    echo %%i | findstr /r /c:"[0-9]*$">nul && (
        echo Deleting: %%i
        rmdir /s /q "%%i"
    )
)

for /L %%i in (0,1,1) do (
    mkdir "%BASEDIR%validator-%%i" >nul 2>&1
    "%BASEDIR%..\validator.exe" wallet create --wallet-dir="%BASEDIR%validator-%%i" --keymanager-kind=imported --wallet-password-file="%BASEDIR%artifacts/wallet/password.txt"
    "%BASEDIR%..\validator.exe" accounts import --wallet-dir="%BASEDIR%validator-%%i" --keys-dir="%BASEDIR%artifacts\keyfiles\validator%%i" --wallet-password-file="%BASEDIR%artifacts/wallet/password.txt"

    set /a "rpcport=7000 + %%i"
    set /a "beaconrpcport=4000 + %%i"
    set /a "rpcgatewayport=3500 + %%i"
    set /a "slasherrpc=4002 + %%i"
    set "node_dir=%BASEDIR%validator-%%i\"

    copy "%BASEDIR%artifacts\recipients\recipient%%i.yaml" "%BASEDIR%validator-%%i\recipient.yaml"
    copy "%BASEDIR%artifacts\wallet\password.txt" "%BASEDIR%validator-%%i\password.txt"

    set "script_name=%BASEDIR%\validator-%%i\run_validator.bat"
    (   
        echo @echo off 
        echo SET "BASEDIR=!node_dir!"
        echo SET "VALIDATOR_CLIENT_PATH=%BASEDIR%..\validator.exe"
        echo SET "rpcport=!rpcport!"
        echo SET "beaconrpcport=!beaconrpcport!"
        echo SET "rpcgatewayport=!rpcgatewayport!"
        echo SET "slasherrpc=!slasherrpc!"
    ) >> !script_name!
    type "artifacts\run_validator.bat" >> !script_name!
)

ENDLOCAL