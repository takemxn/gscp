#!/bin/bash

SCPUSER=take
D=/tmp
test_scp_remote_local(){
	trap 'return 1' ERR

	echo "abc" > $D/from/t.txt
	set -x
	./gscp $SCPUSER@localhost:$D/from/t.txt $D/to/t.txt
	diff $D/from/t.txt $D/to/t.txt
	set +x
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
	rm_dir
	mkdir ${D}/from ${D}/to
	test_scp_remote_local
	test_scp_local_remote
	test_scp_remote_remote
	rm_dir
}
trap 'rm_dir;return 1' ERR
main
