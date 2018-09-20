#!/bin/bash

. /tmp/gscp_test.conf

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
diff_deep(){
	local A=$1
	local B=$2
	
	diff -r $A $B
	#(cd $A; ls --time-style="+%Y-%m-%d %H:%M:%S" -lR |egrep -v "^d"|sort) > A.txt
	#(cd $B; ls --time-style="+%Y-%m-%d %H:%M:%S" -lR |egrep -v "^d"|sort) > B.txt
	(cd $A; ls --time-style="+%Y-%m-%d %H:%M:%S" -lR |sort) > A.txt
	(cd $B; ls --time-style="+%Y-%m-%d %H:%M:%S" -lR |sort) > B.txt
	diff A.txt B.txt
	return $?
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
TEST_NORM_COPY(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	export GSSH_PASSWORDFILE=${CONFIG}
	echo NORMAL1
	echo abcdefg > $D/from/t.txt
	./gscp -v $D/from/t.txt $SCPUSER1@${REMOTE}:$D/to
	./gscp -v $SCPUSER1@${REMOTE}:$D/to/t.txt $D/to/.
	diff $D/from/t.txt $D/to/t.txt
	./gscp -v $D/from/t.txt $SCPUSER1@${REMOTE}:$D/to/b.txt
	./gscp -v $SCPUSER1@${REMOTE}:$D/to/b.txt $D/to/c.txt
	diff $D/from/t.txt $D/to/c.txt
	set +x
	echo 20m
	init_dir
	set -x
	F=20m.bin
	head -c 20m /dev/urandom > $D/from/${F}
	./gscp $D/from/${F} $SCPUSER1@${REMOTE}:$D/to/.
	./gscp $SCPUSER1@${REMOTE}:$D/to/${F} $D/to/
	diff $D/from/${F} $D/to/
	echo TEST:WILDCARD
	set +x
	init_dir
	set -x
	echo def > $D/from/t.txt
	F=1m.bin
	head -c 1m /dev/urandom > $D/from/${F}
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
	./gscp -r $D/from $SCPUSER1@${REMOTE}:$D/to
	./gscp -r $SCPUSER1@${REMOTE}:$D/to/from $D/to/.
	diff -r $D/from $D/to/from
	set +x
	echo TEST:MULTI COPY
	init_dir
	set -x
	mkdir -p $D/from/d1/d2/d3
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	echo c > $D/from/d1/c.txt
	head -c 20m /dev/urandom > $D/from/d1/d2/d3/20m.bin
	./gscp -v -r $D/from/*.txt $D/from/d1 $SCPUSER1@${REMOTE}:$D/to
	./gscp -v -r $SCPUSER1@${REMOTE}:$D/to/*.txt $SCPUSER1@${REMOTE}:$D/to/d1 $D/to/.
	diff -r $D/from $D/to
	set +x
	echo TEST:REMOTE TO REMOTE
	init_dir
	set -x
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	./gscp -q -v $D/from/* $SCPUSER1@${REMOTE}:$D/from/.
	./gscp -q -v $SCPUSER1@${REMOTE}:$D/from/* $SCPUSER2@${REMOTE}:$D/to
	./gscp -q -v $SCPUSER2@${REMOTE}:$D/to/* $D/to
	diff -r $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_P(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	echo TEST:RECURSIVE,PRESERVE
	init_dir
	set -x
	mkdir $D/from/d1
	echo a > $D/from/a.txt
	echo b > $D/from/d1/b.txt
	sleep 2
	./gscp -p -v -r $D/from $SCPUSER1@${REMOTE}:$D/
	sleep 2
	./gscp -p -v -r $SCPUSER1@${REMOTE}:$D/from $D/to/.
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
	while getopts :a OPT
	do
		case $OPT in
			a) FLAG_A=1
		esac
	done
	shift $((OPTIND -1))
	func_list=`typeset -F|cut -d' ' -f3|egrep "^TEST_"`
	if [ -n "${FLAG_A}" ]; then
		for f in ${func_list}
		do
			eval "$f"
		done
	elif [ ${#@} -eq 0 ]; then
		echo :list test function
		echo "${func_list}"
		exit 0
	fi
	eval "$1"
}
main "$@" 2>&1 | tee gscp_test.log
