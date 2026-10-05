!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!ifndef BUILD_UNINSTALLER
; Discovering this computer from the phone needs inbound UDP 5353 (mDNS) on the
; local network. Windows asks for that on first launch, but a dismissed prompt -
; or an image with prompts disabled - would silently break discovery, so the rule
; is registered during the install. Only an elevated install (all users, or "run
; as administrator") may change firewall policy; a per-user install logs the
; netsh error and keeps working through the Windows prompt instead.
; The public profile stays untouched on purpose: mDNS on a network Windows
; treats as public is not something this app should widen silently.
!macro customInstall
  DetailPrint "登记局域网发现所需的防火墙规则…"
  nsExec::ExecToLog 'netsh advfirewall firewall add rule name="Questrace LAN discovery" dir=in action=allow protocol=UDP localport=5353 profile=private,domain'
  Pop $0
!macroend
!endif

!ifdef BUILD_UNINSTALLER
Var QuestraceKeepData
Var QuestraceKeepDataCheckbox
Var QuestraceDataDialog
Var QuestraceRoamingData

!macro customUnInit
  ; Every invocation starts with preservation, including silent upgrades.
  StrCpy $QuestraceKeepData ${BST_CHECKED}
  SetShellVarContext current
  StrCpy $QuestraceRoamingData "$APPDATA"
  ${If} $installMode == "all"
    SetShellVarContext all
  ${EndIf}
!macroend

!macro customUnWelcomePage
  !insertmacro MUI_UNPAGE_WELCOME
  UninstPage custom un.QuestraceDataPage un.QuestraceDataPageLeave
!macroend

!macro customUnInstall
  ; The firewall rule is machine-wide, so it leaves with the application.
  nsExec::ExecToLog 'netsh advfirewall firewall delete rule name="Questrace LAN discovery"'
  Pop $0
  ; Never delete data on upgrades or unattended uninstalls. Only the visible
  ; checkbox and its confirmation can authorize this application's cleanup.
  ${IfNot} ${isUpdated}
  ${AndIfNot} ${Silent}
  ${AndIf} $QuestraceKeepData == ${BST_UNCHECKED}
    Call un.QuestraceDeleteData
  ${EndIf}
!macroend

; Define functions after electron-builder has registered its NSIS plugins.
!macro customHeader
Function un.QuestraceDataPage
  ${If} ${isUpdated}
    Abort
  ${EndIf}
  !insertmacro MUI_HEADER_TEXT "保留用户数据" "选择卸载 Questrace 后是否保留题目、附件和设置。"
  nsDialogs::Create 1018
  Pop $QuestraceDataDialog
  ${If} $QuestraceDataDialog == error
    Abort
  ${EndIf}
  ${NSD_CreateCheckbox} 0 4u 100% 20u "保留用户数据（推荐）"
  Pop $QuestraceKeepDataCheckbox
  ${NSD_SetState} $QuestraceKeepDataCheckbox $QuestraceKeepData
  ${NSD_CreateLabel} 0 32u 100% 28u "勾选：仅卸载程序，重新安装后可以继续使用原有数据。"
  Pop $0
  ${NSD_CreateLabel} 0 64u 100% 36u "取消勾选：同时删除本用户的题目、图片附件、账号和设置，无法恢复。"
  Pop $0
  ${NSD_CreateLabel} 0 104u 100% 36u "数据位置：$QuestraceRoamingData\Questrace$\r$\n同时清理程序缓存和可识别的旧版 Notebook 数据。自定义数据目录保留。"
  Pop $0
  nsDialogs::Show
FunctionEnd

Function un.QuestraceDataPageLeave
  ${NSD_GetState} $QuestraceKeepDataCheckbox $QuestraceKeepData
  ${If} $QuestraceKeepData == ${BST_UNCHECKED}
    MessageBox MB_YESNO|MB_ICONEXCLAMATION|MB_DEFBUTTON2 "确定卸载并永久删除题目、附件、账号和设置吗？" IDYES confirmed
    Abort
    confirmed:
  ${EndIf}
FunctionEnd

Function un.QuestraceDeleteData
  ; Only fixed application-owned children of the captured current-user root.
  ; Do not recursively remove AppData itself or a path from an environment override.
  StrCmp $QuestraceRoamingData "" cleanup_failed
  ClearErrors
  RMDir /r "$QuestraceRoamingData\Questrace"
  IfErrors cleanup_failed
  RMDir /r "$QuestraceRoamingData\questrace-desktop"
  IfErrors cleanup_failed
  Delete "$QuestraceRoamingData\Questrace.lock"
  ; Notebook is a legacy name: require our database before deleting that folder.
  IfFileExists "$QuestraceRoamingData\Notebook\questrace.db" legacy_data
  IfFileExists "$QuestraceRoamingData\Notebook\notebook.db" legacy_data cleanup_done
  legacy_data:
    ClearErrors
    RMDir /r "$QuestraceRoamingData\Notebook"
    IfErrors cleanup_failed
    Delete "$QuestraceRoamingData\Notebook.lock"
  cleanup_done:
    Return
  cleanup_failed:
    MessageBox MB_OK|MB_ICONEXCLAMATION "部分用户数据未能删除。请关闭 Questrace 后重试，并检查数据目录的访问权限。卸载已停止。"
    Abort
FunctionEnd
!macroend
!endif
