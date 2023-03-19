@echo off
set NODE_OPTIONS=--openssl-legacy-provider
set GOARCH=386
set CGO_ENABLED=1

echo Cleaning up...
del lardoon.exe 2>NUL
del cmd\lardoon\resource.syso 2>NUL
del /F dist\*.ico 2>NUL
del /F dist\*.mp3 2>NUL
del /F dist\*.LICENSE.txt 2>NUL
del /F dist\*.js 2>NUL
del /F dist\*.html 2>NUL
del /F dist\*.css 2>NUL
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
call yarn
call yarn build
echo Building lardoon.exe ...
cd cmd\lardoon\
go generate
go build -o ..\..\lardoon.exe
cd ..\..
echo Done.
