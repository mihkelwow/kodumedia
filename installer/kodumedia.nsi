Unicode True
RequestExecutionLevel user
SetCompressor /SOLID lzma
SetCompressorDictSize 64

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"

!ifndef VERSION
  !error "VERSION is required"
!endif
!ifndef PAYLOAD_DIR
  !error "PAYLOAD_DIR is required"
!endif
!ifndef SOURCE_DIR
  !error "SOURCE_DIR is required"
!endif
!ifndef OUTPUT_FILE
  !error "OUTPUT_FILE is required"
!endif

!define PRODUCT_NAME "KoduMedia"
!define PRODUCT_PUBLISHER "KoduMedia"
!define PRODUCT_WEB_SITE "http://127.0.0.1:8096"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\KoduMedia"
!define PRODUCT_RUN_KEY "Software\Microsoft\Windows\CurrentVersion\Run"
!define PRODUCT_APPROVED_KEY "Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run"

Name "${PRODUCT_NAME} ${VERSION}"
OutFile "${OUTPUT_FILE}"
InstallDir "$LOCALAPPDATA\Programs\KoduMedia"
InstallDirRegKey HKCU "${PRODUCT_UNINST_KEY}" "InstallLocation"
Icon "${SOURCE_DIR}\apps\server\internal\adapters\system\tray\assets\kodu-media.ico"
UninstallIcon "${SOURCE_DIR}\apps\server\internal\adapters\system\tray\assets\kodu-media.ico"
BrandingText "KoduMedia"
ShowInstDetails show
ShowUninstDetails show

VIProductVersion "0.8.54.0"
VIAddVersionKey "ProductName" "KoduMedia"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "KoduMedia paigaldaja"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "CompanyName" "KoduMedia"
VIAddVersionKey "LegalCopyright" "KoduMedia"

!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\kodumedia.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Käivita KoduMedia"
!define MUI_FINISHPAGE_LINK "Ava Kodu brauseris"
!define MUI_FINISHPAGE_LINK_LOCATION "http://127.0.0.1:8096"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Estonian"
!insertmacro MUI_LANGUAGE "English"

Section "KoduMedia" SEC_CORE
  SectionIn RO
  SetShellVarContext current

  ; Stop the portable or previously installed process before replacing files.
  nsExec::ExecToLog '"$SYSDIR\taskkill.exe" /IM kodumedia.exe /T /F'
  Sleep 750

  SetOutPath "$INSTDIR"
  File /r "${PAYLOAD_DIR}\*.*"

  InitPluginsDir
  File "/oname=$PLUGINSDIR\migrate-data.ps1" "${SOURCE_DIR}\installer\migrate-data.ps1"
  nsExec::ExecToStack '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "$PLUGINSDIR\migrate-data.ps1" -InstallDirectory "$INSTDIR" -DataDirectory "$LOCALAPPDATA\KoduMedia\data"'
  Pop $0
  Pop $1
  ${If} $0 != 0
    MessageBox MB_ICONSTOP|MB_OK "Olemasolevate KoduMedia andmete migreerimine ebaõnnestus.$\r$\n$1$\r$\n$\r$\nVana data-kausta ei muudetud."
    Abort
  ${EndIf}

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  CreateDirectory "$SMPROGRAMS\KoduMedia"
  CreateShortcut "$SMPROGRAMS\KoduMedia\KoduMedia.lnk" "$INSTDIR\kodumedia.exe" "" "$INSTDIR\kodumedia.exe" 0 SW_SHOWNORMAL "" "$INSTDIR"
  CreateShortcut "$SMPROGRAMS\KoduMedia\Eemalda KoduMedia.lnk" "$INSTDIR\Uninstall.exe"

  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "DisplayName" "KoduMedia"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\kodumedia.exe,0"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKCU "${PRODUCT_UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
  WriteRegDWORD HKCU "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${PRODUCT_UNINST_KEY}" "NoRepair" 1

  ; A fresh value and a cleared StartupApproved state avoid preserving an old
  ; disabled portable entry. Runtime retries cover Explorer's login race.
  WriteRegStr HKCU "${PRODUCT_RUN_KEY}" "KoduMedia" '"$INSTDIR\kodumedia.exe" --startup'
  DeleteRegValue HKCU "${PRODUCT_APPROVED_KEY}" "KoduMedia"
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  nsExec::ExecToLog '"$SYSDIR\taskkill.exe" /IM kodumedia.exe /T /F'
  Sleep 500
  DeleteRegValue HKCU "${PRODUCT_RUN_KEY}" "KoduMedia"
  DeleteRegValue HKCU "${PRODUCT_APPROVED_KEY}" "KoduMedia"
  DeleteRegKey HKCU "${PRODUCT_UNINST_KEY}"
  RMDir /r "$SMPROGRAMS\KoduMedia"
  RMDir /r "$INSTDIR"

  ; User metadata, settings and watch history intentionally remain in
  ; %LOCALAPPDATA%\KoduMedia\data for a later reinstall or manual backup.
SectionEnd
