#!/bin/bash

SCPUSER=take
D=/tmp
test_scp_remote_local(){
	set -x
	echo "abc" > $D/from/t.txt
	./gscp $SCPUSER@localhost:$D/from/t.txt $D/to/t.txt
	diff $D/from $D/to

	./gscp -r $SCPUSER@localhost:$D/from $D/to
	diff $D/from $D/to/from

	chmod 777 $D/from/t.txt
	sleep 2
	touch -a $D/from/t.txt
	echo "def" > $D/from/a.txt
	./gscp -p -r $SCPUSER@localhost:$D/from $D/to
	diff_deep $D/from $D/to/from
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
main(){
	trap 'set +x;return 1' ERR
	rm_dir
	mkdir ${D}/from ${D}/to
	test_scp_remote_local
	test_scp_local_remote
	#test_scp_remote_remote
}
main
