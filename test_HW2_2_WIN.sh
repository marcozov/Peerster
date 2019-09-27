#!/usr/bin/env bash

go.exe build
cd clientExec
go.exe build
cd ..


#./Peerster.exe -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer > $outFileName &
#./Peerster.exe -UIPort=12345 -gossipAddr=127.0.0.1:5000 -name=A -peers=$peer > $outFileName &

UIPort=12345
gossipPort=5000
name='A'

# General peerster (gossiper) command
#./Peerster -UIPort=12345 -gossipAddr=127.0.0.1:5001 -name=A -peers=127.0.0.1:5002 > A.out &

for i in `seq 1 10`;
do
        outFileName="$name.out"
        peerPort=$((($gossipPort+1)%10+5000))
        peer="127.0.0.1:$peerPort"
        gossipAddr="127.0.0.1:$gossipPort"
        #./Peerster -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer > $outFileName &
        ./Peerster.exe -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer > $outFileName &
        outputFiles+=("$outFileName")
        if [[ "$DEBUG" == "true" ]] ; then
                echo "$name running at UIPort $UIPort and gossipPort $gossipPort"
        fi
        UIPort=$(($UIPort+1))
        gossipPort=$(($gossipPort+1))
        names+=("$name")
        name=$(echo "$name" | tr "A-Y" "B-Z")
done

# sharing a file: taking it from the _SharedFiles folder and load it in memory. After this, the gossiper is able to provide it when another host requests it.
./clientExec/clientExec.exe -UIPort=12345 -file="test_file"

cmd "/C TASKKILL /F /IM Peerster.exe"
