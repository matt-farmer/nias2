rem This is the NIAS batch file launcher. Add extra validators to the bottom of this list. 
rem Change the directory as appropriate (go-nias)
rem gnatsd MUST be the first program launched

@echo off

rem Run the NIAS services. Add to the BOTTOM of this list
start nats-streaming-server -p 4223 -sc nias_nss.cfg
timeout /t 3
start go-nias8


rem Run the web client (launch browser here)
start http://localhost:1325/nias
