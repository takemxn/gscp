#!/bin/bash

. /tmp/user.info

CONFIG=/tmp/gscp.conf
cat <<EOS >$CONFIG
[passwords]
$SCPUSER1=$SCPUSER1_PASSWD
$SCPUSER2=$SCPUSER2_PASSWD
EOS

D=/tmp
err_h(){
	set +x
	script=$0
	line=$1
	echo "ERROR:$script:$line:${FUNCNAME[1]}"
	exit 1
}
TEST_REMOTE_TO_LOCAL_1(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	head -c 500000 /dev/urandom > $D/from/t.txt
	set -x
	./gscp -qv $SCPUSER1@localhost:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to
	./gscp -qv $SCPUSER1@localhost:$D/from/t.txt $D/to/b.txt
	diff $D/from/t.txt $D/to/b.txt
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_2(){
	trap "err_h $LINENO" ERR
	echo ${FUNCNAME[0]}
	init_dir
	head -c 1m /dev/urandom > $D/from/t.txt
	set -x
	./gscp -qv -r $SCPUSER1@localhost:$D/from $D/to
	diff $D/from $D/to/from
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_3(){
	trap "err_h $LINENO" ERR
	echo ${FUNCNAME[0]}
	init_dir
	head -c 200m /dev/urandom > $D/from/t.txt
	chmod 777 $D/from/t.txt
	sleep 2
	head -c 600000 /dev/urandom > $D/from/a.txt
	echo "def" > $D/from/a.txt
	set -x
	./gscp -qp -v -r $SCPUSER1@localhost:$D/from $D/to
	diff_deep $D/from $D/to/from
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_4(){
	trap "err_h $LINENO" ERR
	echo ${FUNCNAME[0]}
	init_dir
	head -c 200m /dev/urandom > $D/from/t.txt
	mkdir $D/from/tt
	echo "def" > $D/from/a.txt
	head -c 14098 /dev/urandom > $D/from/tt/tt.txt
	mkdir $D/from/dd
	head -c 900000 /dev/urandom > $D/from/dd/d.txt
	head -c 910000 /dev/urandom > $D/from/dd/d2.txt
	head -c 910000 /dev/urandom > $D/from/dd/d2aaaaaaaa.txt
	sleep 2
	set -x
	./gscp -qp -vr $SCPUSER1@localhost:$D/from/* $D/to
	diff -r $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_5(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	head -c 2m /dev/urandom > $D/from/a.txt
	head -c 8000 /dev/urandom > $D/from/b.txt
	./gscp -qv $SCPUSER1@localhost:$D/from/*.txt $D/to
	diff $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_6(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	head -c 2m /dev/urandom > $D/from/a.txt
	head -c 8000 /dev/urandom > $D/from/b.txt
	./gscp -qv $SCPUSER1@localhost:$D/from/*.txt $D/to/t.txt
	diff $D/from/b.txt $D/to/t.txt
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_7(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	mkdir $D/from/ttt
	touch $D/to/t.txt
	trap '' ERR
	ERR_MSG=`./gscp -qr $SCPUSER1@localhost:$D/from/. $D/to/t.txt 2>&1`
	if [ "${ERR_MSG}" != "scp: \"$D/to/t.txt\": Not a directory" ]; then
		err_h $LINENO
	fi
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_8(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	mkdir $D/from/ttt
	touch $D/to/t.txt
	trap '' ERR
	ERR_MSG=`./gscp -q take@localhost:/tmp/from /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from: not a regular file" ]; then
		return 1
	fi
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_RR_TO_LOCAL_10(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	echo a > /tmp/from/a.txt
	echo b > /tmp/from/b.txt
	set -x
	ls -1 /tmp/from/*
	./gscp -q ${SCPUSER1}@localhost:/tmp/from/a.txt ${SCPUSER2}@localhost:/tmp/from/b.txt /tmp/to
	diff /tmp/from /tmp/to
	set +x
	echo "${FUNCNAME[0]} success"
}
test_scp_to_local(){
	trap "err_h $LINENO" ERR
	cp $CONFIG ~/.gssh
	TEST_REMOTE_TO_LOCAL_1
	TEST_REMOTE_TO_LOCAL_2
	TEST_REMOTE_TO_LOCAL_3
	TEST_REMOTE_TO_LOCAL_4
	TEST_REMOTE_TO_LOCAL_5
	TEST_REMOTE_TO_LOCAL_6
	TEST_REMOTE_TO_LOCAL_7
	TEST_RR_TO_LOCAL_10
}
diff_deep(){
	local A=$1
	local B=$2
	
	diff -r $A $B
	(cd $A; ls --time-style="+%Y-%m-%d %H:%M:%S" -lR |sort) > A.txt
	(cd $B; ls --time-style="+%Y-%m-%d %H:%M:%S" -lR |sort) > B.txt
	diff A.txt B.txt
	return $?
}
TEST_LOCAL_TO_REMOTE_1(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	for ((i=0;$i<3278;i++)); do
		echo "${i} 012345678901234567890123456789012345678901234567890123456789" >> $D/from/a.txt
	done
	set -x
	./gscp -q $D/from/a.txt $SCPUSER1@localhost:$D/to/a.txt
	diff $D/from/a.txt $D/to/a.txt
	echo b > $D/from/b.txt
	./gscp -q $D/from/*.txt $SCPUSER1@localhost:$D/to/.
	diff $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_LOCAL_TO_REMOTE_2(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	mkdir $D/from/ttt
	echo ttt > $D/from/t.txt
	echo ccc > $D/from/ttt/ccc.txt
	head -c 20m /dev/urandom > $D/from/random.bin
	./gscp -qr $D/from $SCPUSER1@localhost:$D/to
	diff -r $D/from $D/to/from
	set +x
	echo "${FUNCNAME[0]} success"
}
test_scp_to_remote(){
	trap "err_h $LINENO" ERR
	cp $CONFIG ~/.gssh
	TEST_LOCAL_TO_REMOTE_1
	TEST_LOCAL_TO_REMOTE_2
	return 0
}
TEST_REMOTE_TO_REMOTE_1(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	head -c 20m /dev/urandom > $D/from/random.bin
	./gscp -q $SCPUSER2@localhost:$D/from/random.bin $SCPUSER1@localhost:$D/to
	diff $D/from/random.bin $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
test_scp_remote_remote(){
	TEST_REMOTE_TO_REMOTE_1
}
test_scp_opt(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	rm -f ~/.gssh
	head -c 500000 /dev/urandom > $D/from/t.txt
	./gscp -q -F ${CONFIG} $SCPUSER1@localhost:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
rm_dir(){
	rm -rf $D/from
	rm -rf $D/to
}
init_dir(){
	rm_dir
	mkdir ${D}/from ${D}/to
}
main(){
	trap "err_h $LINENO" ERR
	test_scp_opt
	test_scp_to_local
	test_scp_to_remote
	test_scp_remote_remote
}
main 2>&1 | tee gscp_test.log
