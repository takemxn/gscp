#!/bin/bash

# The following environment variables must be set
#  $LRUSER1
#  $LRUSER1_PASSWD
#  $RUSER1
#  $RUSER1_PASSWD
#  $LUSER1
#  $LUSER1_PASSWD
#  $REMOTE
#  $ROOTPASSWD

. /tmp/gscp_test.conf

CONFIG=/tmp/gscp.conf
cat <<EOS >$CONFIG
[passwords]
$LRUSER1=$LRUSER1_PASSWD
$RUSER1=$RUSER1_PASSWD
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
	chmod 777 ${D}/from ${D}/to
	gssh -p ${ROOTPASSWD} root@${REMOTE} <<EOS
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
	export GSSH_PASSWORDS="$LRUSER1=$LRUSER1_PASSWD $RUSER1=$RUSER1_PASSWD"
	init_dir
	set -x
	echo NORMAL1
	echo abcdefg > $D/from/t.txt
	./gscp -v $D/from/t.txt $LRUSER1@${REMOTE}:$D/to
	./gscp -v $LRUSER1@${REMOTE}:$D/to/t.txt $D/to/.
	diff $D/from/t.txt $D/to/t.txt
	./gscp -v $D/from/t.txt $LRUSER1@${REMOTE}:$D/to/b.txt
	./gscp -v $LRUSER1@${REMOTE}:$D/to/b.txt $D/to/c.txt
	diff $D/from/t.txt $D/to/c.txt
	set +x
	echo 20m
	init_dir
	set -x
	F=20m.bin
	head -c 20m /dev/urandom > $D/from/${F}
	./gscp $D/from/${F} $LRUSER1@${REMOTE}:$D/to/.
	./gscp $LRUSER1@${REMOTE}:$D/to/${F} $D/to/
	diff $D/from/${F} $D/to/
	echo TEST:WILDCARD
	set +x
	init_dir
	set -x
	echo def > $D/from/t.txt
	F=1G.bin
	head -c 1G /dev/urandom > $D/from/${F}
	./gscp -v $D/from/* $LRUSER1@${REMOTE}:$D/to
	./gscp -v $LRUSER1@${REMOTE}:$D/to/* $D/to/
	diff $D/from $D/to
	set +x
	echo TEST:RECURSIVE
	init_dir
	set -x
	mkdir -p $D/from/d1/d2/d3
	mkdir -p $D/from/d4/d5/d6
	echo a > $D/from/a.txt
	echo b > $D/from/d1/b.txt
	head -c 10m /dev/urandom > $D/from/d1/d2/d3/10m.bin
	head -c $(date '+%s') /dev/urandom > $D/from/d4/d5/d6/d7.bin
	head -c 200m /dev/urandom > $D/from/d4/d5/d6/200m.bin
	./gscp -r $D/from $LRUSER1@${REMOTE}:$D/to
	./gscp -r $LRUSER1@${REMOTE}:$D/to/from $D/to/.
	diff -r $D/from $D/to/from
	set +x
	echo TEST:REMOTE TO REMOTE
	init_dir
	set -x
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	./gscp -q -v $D/from/* $LRUSER1@${REMOTE}:$D/from/.
	./gscp -q -v $LRUSER1@${REMOTE}:$D/from/* $RUSER1@${REMOTE}:$D/to
	./gscp -q -v $RUSER1@${REMOTE}:$D/to/* $D/to
	diff -r $D/from $D/to
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_P(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	echo TEST:RECURSIVE,PRESERVE
	export GSSH_PASSWORDS="$LRUSER1=$LRUSER1_PASSWD $RUSER1=$RUSER1_PASSWD"
	init_dir
	set -x
	mkdir $D/from/d1
	echo a > $D/from/a.txt
	echo b > $D/from/d1/b.txt
	sleep 2
	./gscp -p -v -r $D/from $LRUSER1@${REMOTE}:$D/to
	sleep 2
	./gscp -p -v -r $LRUSER1@${REMOTE}:$D/to/from $D/to/.
	diff_deep $D/from $D/to/from
	set +x
	echo TEST:RECURSIVE,PRESERVE2
	init_dir
	set -x
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	mkdir -p $D/from/d1/d2
	head -c 800001 /dev/urandom > $D/from/d1/d1.bin
	head -c 800002 /dev/urandom > $D/from/d1/d2.bin
	head -c 800002 /dev/urandom > $D/from/d1/d2/d3.bin
	chmod 722 $D/from/d1/d2
	chmod 722 $D/from/d1/d2.bin
	chmod 600 $D/from/d1/d2/d3.bin
	sleep 2
	./gscp -p -v -r $D/from/*.txt $D/from/d1 $LRUSER1@${REMOTE}:$D/to
	sleep 2
	./gscp -p -v -r $LRUSER1@${REMOTE}:$D/to/*.txt $LRUSER1@${REMOTE}:$D/to/d1 $D/to/.
	diff $D/from/a.txt $D/to/a.txt
	diff $D/from/b.txt $D/to/b.txt
	diff_deep $D/from/d1 $D/to/d1
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_ERR_PTN(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"
	init_dir
	set -x
	export GSSH_PASSWORDS="$LRUSER1=$LRUSER1_PASSWD $RUSER1=$RUSER1_PASSWD"
	mkdir $D/from/ttt
	touch $D/to/t.txt
	trap '' ERR
	ERR_MSG=`./gscp -qr $LRUSER1@${REMOTE}:$D/from/. $D/to/t.txt 2>&1`
	if [ "${ERR_MSG}" != "scp: \"$D/to/t.txt\": Not a directory" ]; then
		err_h $LINENO
	fi
	set +x
	init_dir
	set -x
	mkdir $D/from/ttt
	touch $D/to/t.txt
	trap '' ERR
	ERR_MSG=`./gscp $LRUSER1@${REMOTE}:/tmp/from /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from: not a regular file" ]; then
		err_h $LINENO
	fi
	ERR_MSG=`./gscp $LRUSER1@${REMOTE}:/tmp/from/nothing /tmp/to 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/from/nothing: No such file or directory" ]; then
		err_h $LINENO
	fi
	set +x
	init_dir
	set -x
	echo a> $D/from/a.txt
	echo b> $D/from/b.txt
	./gscp $D/from/a.txt ${LRUSER1}@${REMOTE}:/tmp/to
	trap '' ERR
	ERR_MSG=`./gscp -q $D/from/b.txt ${RUSER1}@${REMOTE}:/tmp/to/a.txt 2>&1`
	if [ "${ERR_MSG}" != "scp: /tmp/to/a.txt: Permission denied" ]; then
		err_h $LINENO
	fi
	ERR_MSG=`./gscp -r ${LRUSER1}@localhost:/tmp/from /tmp/to/from/a 2>&1`
	if [ "${ERR_MSG}" != "mkdir /tmp/to/from/a: no such file or directory" ]; then
		err_h $LINENO
	fi
	set +x
	echo "${FUNCNAME[0]} success"
}
TEST_OPT_PSSWD(){
	trap "err_h $LINENO" ERR
	echo "${FUNCNAME[0]}"

	echo TEST:\$GSSH_PASSWORDS
	set -x
	rm -f ~/.gssh ${CONFIG}
	export GSSH_PASSWORDFILE=
	export GSSH_PASSWORDS="${LRUSER1}=${LRUSER1_PASSWD} ${LUSER1}=${LUSER1_PASSWD}"
	init_dir
	echo a > $D/from/a.txt
	echo b > $D/from/b.txt
	./gscp $D/from/a.txt ${LRUSER1}@localhost:/tmp/to
	./gscp $D/from/b.txt ${LUSER1}@localhost:/tmp/to
	diff $D/from /tmp/to
	set +x

	echo TEST:~/.gssh
	init_dir
	set -x
	rm -f ~/.gssh ${CONFIG}
	export GSSH_PASSWORDFILE=
	cat <<EOS >~/.gssh
[passwords]
$LRUSER1=$LRUSER1_PASSWD
EOS
	export GSSH_PASSWORDS=
	echo a > $D/from/a.txt
	./gscp $D/from/a.txt ${LRUSER1}@localhost:/tmp/to
	diff $D/from /tmp/to
	set +x

	echo TEST:\$GSSH_PASSWORDFILE
	init_dir
	set -x
	rm -f ~/.gssh ${CONFIG}
	export GSSH_PASSWORDFILE=/tmp/p.conf
	cat <<EOS >${GSSH_PASSWORDFILE}
[passwords]
$LRUSER1=$LRUSER1_PASSWD
EOS
	export GSSH_PASSWORDS=
	echo a > $D/from/a.txt
	./gscp $D/from/a.txt ${LRUSER1}@localhost:/tmp/to
	diff $D/from /tmp/to
	rm -rf ${GSSH_PASSWORDFILE}
	set +x

	echo TEST:'-F'
	init_dir
	set -x
	rm -f ~/.gssh ${CONFIG}
	export GSSH_PASSWORDFILE=
	export GSSH_PASSWORDS=
	echo a > $D/from/a.txt
	cat <<EOS >/tmp/p.conf
[passwords]
$LRUSER1=$LRUSER1_PASSWD
EOS
	./gscp -F /tmp/p.conf $D/from/a.txt ${LRUSER1}@localhost:/tmp/to
	diff $D/from /tmp/to
	rm -rf ${GSSH_PASSWORDFILE}
	set +x

	echo TEST:'-w'
	init_dir
	set -x
	rm -f ~/.gssh ${CONFIG}
	export GSSH_PASSWORDFILE=
	export GSSH_PASSWORDS=
	echo a > $D/from/a.txt
	./gscp -w ${LUSER1_PASSWD} $D/from/a.txt ${LUSER1}@localhost:/tmp/to
	diff $D/from /tmp/to
	rm -rf ${GSSH_PASSWORDFILE}
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
exit ${PIPESTATUS[0]}
