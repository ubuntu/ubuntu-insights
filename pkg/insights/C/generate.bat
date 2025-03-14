@echo off
echo generating Windows shared library...
:: For this to work you need gcc on Windows. Also you need to run `$env:CGO_ENABLED="1"` to make sure Go does something.

go build -o ../../../build/libinsights.dll.1 -buildmode=c-shared libinsights.go
move ../../../build/libinsights.dll.h ../../../build/libinsights.h