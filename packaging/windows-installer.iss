#ifndef MyAppVersion
  #define MyAppVersion "dev"
#endif
#ifndef FilesDir
  #define FilesDir "."
#endif
#ifndef OutputDir
  #define OutputDir "."
#endif

[Setup]
AppId={{B5C7D9E1-F3A5-4B7D-9E1F-3A5B7D9E1F3A}
AppName=Carbonio Dockerization Tools
AppVersion={#MyAppVersion}
AppPublisher=Zextras
AppPublisherURL=https://github.com/galvagnimatteo/carbonio-docker-cli
DefaultDirName={autopf}\Carbonio Dockerization
DefaultGroupName=Carbonio Dockerization
OutputDir={#OutputDir}
OutputBaseFilename=carbonio-dockerization-{#MyAppVersion}-windows-amd64-setup
Compression=lzma2
SolidCompression=yes
ChangesEnvironment=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayName=Carbonio Dockerization Tools

[Files]
Source: "{#FilesDir}\carbonio-dockerization-gui.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#FilesDir}\carbonio-dockerization-cli.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Carbonio Dockerization GUI"; Filename: "{app}\carbonio-dockerization-gui.exe"

[Registry]
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath(ExpandConstant('{app}'))

[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKLM, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'Path', OrigPath) then begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Param + ';', ';' + OrigPath + ';') = 0;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path, AppDir: string;
  P: Integer;
begin
  if CurUninstallStep = usPostUninstall then begin
    AppDir := ExpandConstant('{app}');
    if RegQueryStringValue(HKLM, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'Path', Path) then begin
      P := Pos(';' + AppDir, Path);
      if P > 0 then begin
        Delete(Path, P, Length(';' + AppDir));
        RegWriteStringValue(HKLM, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'Path', Path);
      end;
    end;
  end;
end;
