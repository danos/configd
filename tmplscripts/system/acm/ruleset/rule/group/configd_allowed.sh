#!/opt/vyatta/bin/cliexec
local -a grps
grps=($VAR(/system/login/group/@@))
echo -n ${grps[@]}
