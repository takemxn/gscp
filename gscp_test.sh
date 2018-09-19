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
	./gscp -qv $SCPUSER1@${REMOTE}:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to
	./gscp -qv $SCPUSER1@${REMOTE}:$D/from/t.txt $D/to/b.txt
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
	./gscp -qv -r $SCPUSER1@${REMOTE}:$D/from $D/to
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
	./gscp -qp -v -r $SCPUSER1@${REMOTE}:$D/from $D/to
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
	./gscp -qp -vr $SCPUSER1@${REMOTE}:$D/from/* $D/to
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
	./gscp -qv $SCPUSER1@${REMOTE}:$D/from/*.txt $D/to
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
	./gscp -qv $SCPUSER1@${REMOTE}:$D/from/*.txt $D/to/t.txt
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
	ERR_MSG=`./gscp -qr $SCPUSER1@${REMOTE}:$D/from/. $D/to/t.txt 2>&1`
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
	ERR_MSG=`./gscp take@${REMOTE}:/tmp/from /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from: not a regular file" ]; then
		err_h $LINENO
	fi
	ERR_MSG=`./gscp take@${REMOTE}:/tmp/from/nothing /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from/nothing: No such file or directory" ]; then
		err_h $LINENO
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
	./gscp -q ${SCPUSER1}@${REMOTE}:/tmp/from/a.txt ${SCPUSER2}@${REMOTE}:/tmp/from/b.txt /tmp/to
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
	TEST_REMOTE_TO_LOCAL_8
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
	./gscp -q $D/from/a.txt $SCPUSER1@${REMOTE}:$D/to/a.txt
	diff $D/from/a.txt $D/to/a.txt
	echo b > $D/from/b.txt
	./gscp -q $D/from/*.txt $SCPUSER1@${REMOTE}:$D/to/.
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
	./gscp -qr $D/from $SCPUSER1@${REMOTE}:$D/to
	diff -r $D/from $D/to/from
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_LOCAL_TO_REMOTE_3(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	mkdir $D/from/ttt
	echo ttt > $D/from/t.txt
	echo ccc > $D/from/ttt/ccc.txt
	head -c 20m /dev/urandom > $D/from/random.bin
	./gscp -qr $D/from/* $SCPUSER1@${REMOTE}:$D/to
	diff -r $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_LOCAL_TO_REMOTE_4(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	trap '' ERR
	ERR_MSG=`./gscp /tmp/from take@${REMOTE}:/tmp/to 2>&1`
	if [ "${ERR_MSG}" != "/tmp/from: not a regular file" ]; then
		err_h $LINENO
	fi
	set +x
	echo "${FUNCNAME[0]} success"
}
test_scp_to_remote(){
	trap "err_h $LINENO" ERR
	cp $CONFIG ~/.gssh
	TEST_LOCAL_TO_REMOTE_1
	TEST_LOCAL_TO_REMOTE_2
	TEST_LOCAL_TO_REMOTE_3
	TEST_LOCAL_TO_REMOTE_4
	return 0
}
TEST_REMOTE_TO_REMOTE_1(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	head -c 20m /dev/urandom > $D/from/random.bin
	./gscp -q $SCPUSER2@${REMOTE}:$D/from/random.bin $SCPUSER1@${REMOTE}:$D/to
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
	./gscp -q -F ${CONFIG} $SCPUSER1@${REMOTE}:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
init_dir(){
	rm -rf $D/from
	rm -rf $D/to
	mkdir ${D}/from
	mkdir ${D}/to
	gssh ${SCPUSER1}@${REMOTE} <<EOS
rm -rf $D/from
rm -rf $D/to
mkdir $D/from
mkdir $D/to
chmod 777 $D/from $D/to
EOS
}
TEST_NORM_PTN(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	echo NORMAL1
	echo abcdefg > $D/from/t.txt
	./gscp -qv $D/from/t.txt $SCPUSER1@${REMOTE}:$D/to
	./gscp -qv $SCPUSER1@${REMOTE}:$D/to/t.txt $D/to/.
	diff $D/from/t.txt $D/to/t.txt
	./gscp -qv $D/from/t.txt $SCPUSER1@${REMOTE}:$D/to/b.txt
	./gscp -qv $SCPUSER1@${REMOTE}:$D/to/b.txt $D/to/c.txt
	diff $D/from/t.txt $D/to/c.txt
	set +x
	echo 200m
	init_dir
	set -x
	F=200m.bin
	head -c 200m /dev/urandom > $D/from/${F}
	./gscp -q $D/from/${F} $SCPUSER1@${REMOTE}:$D/to/.
	./gscp -q $SCPUSER1@${REMOTE}:$D/to/${F} $D/to/
	diff $D/from/${F} $D/to/
	echo TEST:WILDCARD
	set +x
	init_dir
	set -x
	echo def > $D/from/t.txt
	F=1G.bin
	head -c 1G /dev/urandom > $D/from/${F}
	./gscp -v $D/from/* $SCPUSER1@${REMOTE}:$D/to
	./gscp -v $SCPUSER1@${REMOTE}:$D/to/* $D/to/
	diff $D/from $D/to
	set +x
	echo TEST:RECURSIVE
	init_dir
	set -x
	mkdir $D/from/d1
	echo a > $D/from/a.txt
	echo b > $D/from/d1/b.txt
	./gscp -qr $D/from $SCPUSER1@${REMOTE}:$D/to
	./gscp -qr $SCPUSER1@${REMOTE}:$D/to/from $D/to/.
	diff -r $D/from $D/to/from
	set +x
	echo TEST:MULTI COPY
	init_dir
	set -x
	mkdir -p $D/from/d1/d2/d3
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	echo c > $D/from/d1/c.txt
	head -c 200m /dev/urandom > $D/from/d1/d2/d3/200m.bin
	./gscp -qr $D/from/*.txt $D/from/d1 $SCPUSER1@${REMOTE}:$D/to
	./gscp -qr $SCPUSER1@${REMOTE}:$D/to/*.txt $SCPUSER1@${REMOTE}:$D/to/d1 $D/to/.
	diff -r $D/from $D/to
	set +x
	echo TEST:REMOTE TO REMOTE
	init_dir
	set -x
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	./gscp -q  $D/from/* $SCPUSER1@${REMOTE}:$D/from/.
	./gscp -q  $SCPUSER1@${REMOTE}:$D/from/* $SCPUSER2@${REMOTE}:$D/to
	./gscp -q  $SCPUSER2@${REMOTE}:$D/to/* $D/to
	diff -r $D/from $D/to
	set +x
	echo TEST:RECURSIVE,PRESERVE
	init_dir
	set -x
	mkdir $D/from/d1
	echo a > $D/from/a.txt
	echo b > $D/from/d1/b.txt
	sleep 2
	./gscp -p -qr $D/from $SCPUSER1@${REMOTE}:$D/
	sleep 2
	./gscp -p -qr $SCPUSER1@${REMOTE}:$D/from $D/to/.
	diff_deep $D/from $D/to/from
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_ERR_PTN(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	mkdir $D/from/ttt
	touch $D/to/t.txt
	trap '' ERR
	ERR_MSG=`./gscp -qr $SCPUSER1@${REMOTE}:$D/from/. $D/to/t.txt 2>&1`
	if [ "${ERR_MSG}" != "scp: \"$D/to/t.txt\": Not a directory" ]; then
		err_h $LINENO
	fi
	set +x
	init_dir
	set -x
	mkdir $D/from/ttt
	touch $D/to/t.txt
	trap '' ERR
	ERR_MSG=`./gscp take@${REMOTE}:/tmp/from /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from: not a regular file" ]; then
		err_h $LINENO
	fi
	ERR_MSG=`./gscp take@${REMOTE}:/tmp/from/nothing /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from/nothing: No such file or directory" ]; then
		err_h $LINENO
	fi
	set +x
	echo "${FUNCNAME[0]} success"
}
main(){
	trap "err_h $LINENO" ERR
	set -x
	init_dir
	TEST_NORM_PTN
	TEST_ERR_PTN
}
main 2>&1 | tee gscp_test.log
