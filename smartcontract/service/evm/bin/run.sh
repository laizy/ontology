go build -o check
cat evm-diffs.txt | ./check 
