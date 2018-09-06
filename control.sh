#!/bin/sh
#
# /etc/init.d/chat
#
#
# chkconfig:   - 85 15
#

### BEGIN INIT INFO
# Provides:          chat
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Start chat at boot time.
# Description:       Control chat.
### END INIT INFO

# Source function library.
. /etc/init.d/functions

#工作目录
workPath=$(cd $(dirname $0)/; pwd)

bin=$workPath/main
pidFile=$workPath/pid
logFile=$workPath/info.log

function check_pid() {
    if [ -f $pidFile ];then
        pid=`cat $pidFile`
        if [ -n $pid ]; then
            running=`ps -p $pid|grep -v "PID TTY" |wc -l`
            return $running
        fi
    fi
    return 0
}

function start() {
    check_pid
    running=$?
    if [ $running -gt 0 ];then
        echo -n "It's now is running already, pid="
        cat $pidFile
        return 1
    fi

    nohup $bin >> $logFile 2>&1 &
    echo $! > $pidFile
    echo "It's started..., pid=$!"
}

function stop() {
    check_pid
    running=$?
    if [ $running -gt 0 ];then
        kill $pid
        echo "It's stoped..."
    else
        echo "It's not running!"
    fi
}

function restart() {
    stop
    sleep 1
    start
}

function status() {
    check_pid
    running=$?
    if [ $running -gt 0 ];then
        echo -n "It's running. PID is "
	cat $pidFile
    else
        echo "It's stoped"
    fi
}

function help() {
    echo "$0 start|stop|restart|status"
}

function pid() {
    cat $pidFile
}

if [ "$1" == "" ]; then
    help
elif [ "$1" == "stop" ];then
    stop
elif [ "$1" == "start" ];then
    start
elif [ "$1" == "restart" ];then
    restart
elif [ "$1" == "status" ];then
    status
elif [ "$1" == "pid" ];then
    pid
else
    help
fi
