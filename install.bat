@ECHO off

FOR /F "TOKENS=2" %%i IN ('FINDSTR /R /C:"\"Major\": *[0-9]*" version.json') DO SET @major=%%i
SET @major=%@major:~0,-1%

FOR /F "TOKENS=2" %%i IN ('FINDSTR /R /C:"\"Minor\": *[0-9]*" version.json') DO SET @minor=%%i
SET @minor=%@minor:~0,-1%

FOR /F "TOKENS=2" %%i IN ('FINDSTR /R /C:"\"Patch\": *[0-9]*" version.json') DO SET @patch=%%i
SET @patch=%@patch:~0,-1%

go install -ldflags "-X main.version=%@major%.%@minor%.%@patch%"
