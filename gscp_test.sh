#!/bin/bash

SCPUSER=take
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
	head -c 20m /dev/urandom > $D/from/t.txt
	set -x
	./gscp -v $SCPUSER@localhost:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_REMOTE_TO_LOCAL_2(){
	trap "err_h $LINENO" ERR
	echo ${FUNCNAME[0]}
	init_dir
	head -c 1m /dev/urandom > $D/from/t.txt
	set -x
	./gscp -v -r $SCPUSER@localhost:$D/from $D/to
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
	head -c 20m /dev/urandom > $D/from/a.txt
	echo "def" > $D/from/a.txt
	set -x
	./gscp -p -v -r $SCPUSER@localhost:$D/from $D/to
	set +x
	diff_deep $D/from $D/to/from
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
	sleep 2
	set -x
	./gscp -p -vr $SCPUSER@localhost:$D/from/* $D/to
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
	./gscp -v $SCPUSER@localhost:$D/from/*.txt $D/to
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
	./gscp -v $SCPUSER@localhost:$D/from/*.txt $D/to/t.txt
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
	ERR_MSG=`./gscp -r $SCPUSER@localhost:$D/from/. $D/to/t.txt 2>&1`
	if [ "${ERR_MSG}" != "gscp: \"$D/to/t.txt\": Not a directory" ]; then
		return 1
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
	ERR_MSG=`./gscp take@localhost:/tmp/from /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from: not a regular file" ]; then
		return 1
	fi
	set +x
	echo "${FUNCNAME[0]} success"
}
test_scp_remote_to_local(){
	trap "err_h $LINENO" ERR
	TEST_REMOTE_TO_LOCAL_1
	TEST_REMOTE_TO_LOCAL_2
	TEST_REMOTE_TO_LOCAL_3
	TEST_REMOTE_TO_LOCAL_4
	TEST_REMOTE_TO_LOCAL_5
	TEST_REMOTE_TO_LOCAL_6
	TEST_REMOTE_TO_LOCAL_7
	TEST_REMOTE_TO_LOCAL_8
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
	echo "abc" > $D/from/a.txt
	./gscp -r $D/from/a.txt $SCPUSER@localhost:$D/to/.
	diff $D/from/a.txt $D/to/a.txt
	echo "${FUNCNAME[0]} success"
}
test_scp_local_to_remote(){
	trap "err_h $LINENO" ERR
	TEST_LOCAL_TO_REMOTE_1
	return 0
}
test_scp_remote_remote(){
	return 0
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
	test_scp_remote_to_local
	#test_scp_local_to_remote
	#test_scp_remote_remote
}
main
