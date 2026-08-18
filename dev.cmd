@echo off
cd /d %~dp0
set PATH=%PATH%;C:\Program Files\Go\bin;C:\Users\guivi\go\bin
wails.exe dev
