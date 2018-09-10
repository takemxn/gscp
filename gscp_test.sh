#!/bin/bash

SCPUSER=take
D=/tmp
test_1(){
	echo TEST_1
	init_dir
	head -c 20m /dev/urandom > $D/from/t.txt
	./gscp $SCPUSER@localhost:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to
}
test_2(){
	echo TEST_2
	init_dir
	head -c 1m /dev/urandom > $D/from/t.txt
	./gscp -r $SCPUSER@localhost:$D/from $D/to
	diff $D/from $D/to/from
}
test_3(){
	echo TEST_3
	init_dir
	head -c 200m /dev/urandom > $D/from/t.txt
	chmod 777 $D/from/t.txt
	sleep 2
	head -c 20m /dev/urandom > $D/from/a.txt
	echo "def" > $D/from/a.txt
	./gscp -p -v -r $SCPUSER@localhost:$D/from $D/to
	diff_deep $D/from $D/to/from
}
test_4(){
	echo TEST_4
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
}
test_scp_remote_local(){
	test_1
	test_2
	test_3
	test_4
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
test_scp_local_remote(){
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
	trap 'set +x;return 1' ERR
	test_scp_remote_local
	test_scp_local_remote
	#test_scp_remote_remote
}
main
