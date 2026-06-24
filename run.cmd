@echo off
chcp 65001 >nul
title Claude Session Cleaner
node "%~dp0bin\claude-session-cleaner.js" %*
exit /b %errorlevel%
