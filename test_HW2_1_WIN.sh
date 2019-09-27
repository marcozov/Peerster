#!/usr/bin/env bash

go.exe build
cd clientExec
go.exe build
cd ..

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
DEBUG="true"

outputFiles=()
message_c1_1=Weather_is_clear
message_c2_1=Winter_is_coming
message_c1_2=No_clouds_really
message_c2_2=Let\'s_go_skiing
message_c3=Is_anybody_here?

message_B_G=From_B_to_G
message_E_G=From_E_to_G
message_C_G=From_C_to_G

message_B_B=From_B_to_B

message_A_B=From_A_to_B
message_A_G=From_A_to_G
message_A_H=From_A_to_H

names=()


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

#./client/client -UIPort=12349 -msg=$message_c1_1
./clientExec/clientExec.exe -UIPort=12349 -msg=$message_c1_1
#./client/client -UIPort=12346 -msg=$message_c2_1
./clientExec/clientExec.exe -UIPort=12346 -msg=$message_c2_1
sleep 2
#./client/client -UIPort=12349 -msg=$message_c1_2
./clientExec/clientExec.exe -UIPort=12349 -msg=$message_c1_2
sleep 1
#./client/client -UIPort=12346 -msg=$message_c2_2
./clientExec/clientExec.exe -UIPort=12346 -msg=$message_c2_2
#./client/client -UIPort=12351 -msg=$message_c3
./clientExec/clientExec.exe -UIPort=12351 -msg=$message_c3

sleep 3
./clientExec/clientExec.exe -UIPort=12349 -dest=G -msg=$message_E_G
sleep 1
./clientExec/clientExec.exe -UIPort=12346 -dest=G -msg=$message_B_G
sleep 1
./clientExec/clientExec.exe -UIPort=12347 -dest=G -msg=$message_C_G

sleep 1
./clientExec/clientExec.exe -UIPort=12346 -dest=B -msg=$message_B_B

sleep 1
./clientExec/clientExec.exe -UIPort=12345 -dest=B -msg=$message_A_B
sleep 1
./clientExec/clientExec.exe -UIPort=12345 -dest=G -msg=$message_A_G
sleep 1
./clientExec/clientExec.exe -UIPort=12345 -dest=H -msg=$message_A_H

sleep 5
#pkill -f Peerster
cmd "/C TASKKILL /F /IM Peerster.exe"


#testing
failed="F"

echo -e "${RED}###CHECK that client messages arrived${NC}"

if !(grep -q "CLIENT MESSAGE $message_c1_1" "E.out") ; then
	failed="T"
fi

if !(grep -q "CLIENT MESSAGE $message_c1_2" "E.out") ; then
	failed="T"
fi

if !(grep -q "CLIENT MESSAGE $message_c2_1" "B.out") ; then
    failed="T"
fi

if !(grep -q "CLIENT MESSAGE $message_c2_2" "B.out") ; then
    failed="T"
fi

if !(grep -q "CLIENT MESSAGE $message_c3" "G.out") ; then
    failed="T"
fi

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK rumor messages ${NC}"

gossipPort=5000
for i in `seq 0 9`;
do
	relayPort=$(($gossipPort-1))
	if [[ "$relayPort" == 4999 ]] ; then
		relayPort=5009
	fi
	nextPort=$((($gossipPort+1)%10+5000))
	msgLine1="RUMOR origin E from 127.0.0.1:[0-9]{4} ID 1 contents $message_c1_1"
	msgLine2="RUMOR origin E from 127.0.0.1:[0-9]{4} ID 2 contents $message_c1_2"
	msgLine3="RUMOR origin B from 127.0.0.1:[0-9]{4} ID 1 contents $message_c2_1"
	msgLine4="RUMOR origin B from 127.0.0.1:[0-9]{4} ID 2 contents $message_c2_2"
	msgLine5="RUMOR origin G from 127.0.0.1:[0-9]{4} ID 1 contents $message_c3"

	if [[ "$gossipPort" != 5004 ]] ; then
		if !(grep -Eq "$msgLine1" "${outputFiles[$i]}") ; then
	        echo "$msgLine1" "${outputFiles[$i]}"
        	failed="T"
    	fi
		if !(grep -Eq "$msgLine2" "${outputFiles[$i]}") ; then
	        echo "$msgLine2" "${outputFiles[$i]}"
        	failed="T"
    	fi
	fi

	if [[ "$gossipPort" != 5001 ]] ; then
		if !(grep -Eq "$msgLine3" "${outputFiles[$i]}") ; then
	        echo "$msgLine3" "${outputFiles[$i]}"
        	failed="T"
    	fi
		if !(grep -Eq "$msgLine4" "${outputFiles[$i]}") ; then
	        echo "$msgLine4" "${outputFiles[$i]}"
        	failed="T"
    	fi
	fi
	
	if [[ "$gossipPort" != 5006 ]] ; then
		if !(grep -Eq "$msgLine5" "${outputFiles[$i]}") ; then
	        echo "$msgLine5" "${outputFiles[$i]}"
        	failed="T"
    	fi
	fi
	gossipPort=$(($gossipPort+1))
done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK mongering${NC}"
gossipPort=5000
for i in `seq 0 9`;
do
    relayPort=$(($gossipPort-1))
    if [[ "$relayPort" == 4999 ]] ; then
        relayPort=5009
    fi
    nextPort=$((($gossipPort+1)%10+5000))

    msgLine1="MONGERING with 127.0.0.1:$relayPort"
    msgLine2="MONGERING with 127.0.0.1:$nextPort"

    if !(grep -q "$msgLine1" "${outputFiles[$i]}") && !(grep -q "$msgLine2" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    gossipPort=$(($gossipPort+1))
done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi


failed="F"
echo -e "${RED}###CHECK status messages ${NC}"
gossipPort=5000
for i in `seq 0 9`;
do
    relayPort=$(($gossipPort-1))
    if [[ "$relayPort" == 4999 ]] ; then
        relayPort=5009
    fi
    nextPort=$((($gossipPort+1)%10+5000))

	msgLine1="STATUS from 127.0.0.1:$relayPort"
	msgLine2="STATUS from 127.0.0.1:$nextPort"
	msgLine3="peer E nextID 3"
	msgLine4="peer B nextID 3"
	msgLine5="peer G nextID 2"	

	if !(grep -q "$msgLine1" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    if !(grep -q "$msgLine2" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    if !(grep -q "$msgLine3" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    if !(grep -q "$msgLine4" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    if !(grep -q "$msgLine5" "${outputFiles[$i]}") ; then
        failed="T"
    fi
	gossipPort=$(($gossipPort+1))
done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK flipped coin${NC}"
gossipPort=5000
for i in `seq 0 9`;
do
    relayPort=$(($gossipPort-1))
    if [[ "$relayPort" == 4999 ]] ; then
        relayPort=5009
    fi
    nextPort=$((($gossipPort+1)%10+5000))

    msgLine1="FLIPPED COIN sending rumor to 127.0.0.1:$relayPort"
    msgLine2="FLIPPED COIN sending rumor to 127.0.0.1:$nextPort"

    if !(grep -q "$msgLine1" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    if !(grep -q "$msgLine2" "${outputFiles[$i]}") ; then
        failed="T"
    fi
	gossipPort=$(($gossipPort+1))

done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK in sync${NC}"
gossipPort=5000
for i in `seq 0 9`;
do
    relayPort=$(($gossipPort-1))
    if [[ "$relayPort" == 4999 ]] ; then
        relayPort=5009
    fi
    nextPort=$((($gossipPort+1)%10+5000))

    msgLine1="IN SYNC WITH 127.0.0.1:$relayPort"
    msgLine2="IN SYNC WITH 127.0.0.1:$nextPort"

    if !(grep -q "$msgLine1" "${outputFiles[$i]}") ; then
        failed="T"
    fi
    if !(grep -q "$msgLine2" "${outputFiles[$i]}") ; then
        failed="T"
    fi
	gossipPort=$(($gossipPort+1))
done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK correct peers${NC}"
gossipPort=5000
for i in `seq 0 9`;
do
    relayPort=$(($gossipPort-1))
    if [[ "$relayPort" == 4999 ]] ; then
        relayPort=5009
    fi
    nextPort=$((($gossipPort+1)%10+5000))

	peersLine1="127.0.0.1:$relayPort,127.0.0.1:$nextPort"
	peersLine2="127.0.0.1:$nextPort,127.0.0.1:$relayPort"

    if !(grep -q "$peersLine1" "${outputFiles[$i]}") && !(grep -q "$peersLine2" "${outputFiles[$i]}") ; then
        failed="T"
    fi
	gossipPort=$(($gossipPort+1))
done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi



failed="F"
echo -e "${RED}###CHECK correct prefix tables${NC}"
gossipPort=5000
for i in `seq 0 9`;
do
    relayPort=$(($gossipPort-1))
    if [[ "$relayPort" == 4999 ]] ; then
        relayPort=5009
    fi
    nextPort=$((($gossipPort+1)%10+5000))

    #echo "$relayPort $nextPort $gossipPort ${names[$i]}"

    #peersLine1="127.0.0.1:$relayPort,127.0.0.1:$nextPort"
    #peersLine1="127.0.0.1:$nextPort,127.0.0.1:$relayPort"
    let j=$i-1
    let k=$i+1
    routingLine1_1="DSDV B 127.0.0.1:$relayPort"
    routingLine1_2="DSDV B 127.0.0.1:$nextPort"
    routingLine2_1="DSDV E 127.0.0.1:$relayPort"
    routingLine2_2="DSDV E 127.0.0.1:$nextPort"
    routingLine3_1="DSDV G 127.0.0.1:$relayPort"
    routingLine3_2="DSDV G 127.0.0.1:$nextPort"

    if !(grep -q "$routingLine1_1" "${outputFiles[$i]}") && !(grep -q "$routingLine1_2" "${outputFiles[$i]}") && [[ "${names[$i]}" != "B" ]] ; then
	echo "$relayPort $nextPort $gossipPort"
	#echo "${names[$i]}"
        #echo "$routingLine1_1"
        #echo "$routingLine1_2"
	failed="T"
    fi
    if !(grep -q "$routingLine2_1" "${outputFiles[$i]}") && !(grep -q "$routingLine2_2" "${outputFiles[$i]}") && [[ "${names[$i]}" != "E" ]] ; then
	failed="T"
    fi
    if !(grep -q "$routingLine3_1" "${outputFiles[$i]}") && !(grep -q "$routingLine3_2" "${outputFiles[$i]}") && [[ "${names[$i]}" != "G" ]] ; then
	failed="T"
    fi


    gossipPort=$(($gossipPort+1))
done

if [[ "$failed" == "T" ]] ; then
    echo -e "${RED}***FAILED***${NC}"
else
    echo -e "${GREEN}***PASSED***${NC}"
fi
