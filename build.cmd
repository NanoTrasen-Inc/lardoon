@echo off
set NODE_OPTIONS=--openssl-legacy-provider

echo Cleaning up...
del /F dist\*.ico 2>NUL
del /F dist\*.mp3 2>NUL
del /F dist\*.LICENSE.txt 2>NUL
del /F dist\*.js 2>NUL
del /F dist\*.html 2>NUL
del /F dist\*.css 2>NUL
call yarn
call yarn build
echo Building lardoon.exe ...
go build -o lardoon.exe cmd\lardoon\main.go
echo Done.
